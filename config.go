package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	g "github.com/damienstuart/gosnmp"
)

// Various default configuration fallback values
//
const (
	defBindAddr            string               = "0.0.0.0"
	defListenPort          string               = "162"
	defLogfileMaxSize      int                  = 1024
	defLogfileMaxBackups   int                  = 7
	defCompressRotatedLogs bool                 = false
	defV3MsgFlag           g.SnmpV3MsgFlags     = g.NoAuthNoPriv
	defV3user              string               = "_v3user"
	defV3authProtocol      g.SnmpV3AuthProtocol = g.NoAuth
	defV3authPassword      string               = "_v3password"
	defV3privacyProtocol   g.SnmpV3PrivProtocol = g.NoPriv
	defV3privacyPassword   string               = "_v3password"
)

type v3Params struct {
	msgFlags        g.SnmpV3MsgFlags
	username        string
	authProto       g.SnmpV3AuthProtocol
	authPassword    string
	privacyProto    g.SnmpV3PrivProtocol
	privacyPassword string
}

type trapexConfig struct {
	listenAddr     string
	listenPort     string
	ignoreVersions []g.SnmpVersion
	runLogFile     string
	configFile     string
	v3Params       v3Params
	filters        []trapexFilter
	debug          bool
	logDropped     bool
	logMaxSize     int
	logMaxBackups  int
	logMaxAge      int
	logCompress    bool
	teConfigured   bool
}

type trapexCommandLine struct {
	configFile string
	bindAddr   string
	listenPort string
	debugMode  bool
}

// Global vars
//
var teConfig *trapexConfig
var teCmdLine trapexCommandLine

func showUsage() {
	usageText := `
Usage: trapex [-h] [-c <config_file>] [-b <bind_ip>] [-p <listen_port>]
              [-d] [-v]
  -h  - Show this help message and exit.
  -c  - Override the location of the trapex configuration file.
  -b  - Override the bind IP address on which to listen for incoming traps.
  -p  - Override the UDP port on which to listen for incoming traps.
  -d  - Enable debug mode (note: produces very verbose runtime output).
  -v  - Print the version of trapex and exit.
`
	fmt.Println(usageText)
}

func processCommandLine() {
	flag.Usage = showUsage
	c := flag.String("c", "/etc/trapex.conf", "")
	b := flag.String("b", "", "")
	p := flag.String("p", "", "")
	d := flag.Bool("d", false, "")
	showVersion := flag.Bool("v", false, "")

	flag.Parse()

	if *showVersion {
		fmt.Printf("This is trapex version %s.\n", myVersion)
		os.Exit(0)
	}

	teCmdLine.configFile = *c
	teCmdLine.bindAddr = *b
	teCmdLine.listenPort = *p
	teCmdLine.debugMode = *d
}

func getConfig() error {
	// If this is a reconfig close any current handles
	if teConfig != nil && teConfig.teConfigured {
		fmt.Printf("Reloading ")
		closeTrapexHandles()
	} else {
		fmt.Printf("Loading ")
	}
	fmt.Printf("configuration for trapex version %s from: %s.\n", myVersion, teCmdLine.configFile)

	var newConfig trapexConfig

	// First process the config file
	cf, err := os.Open(teCmdLine.configFile)
	if err != nil {
		return err
	}
	defer cf.Close()

	cfSkipRe := regexp.MustCompile(`^\s*#|^\s*$`)

	scanner := bufio.NewScanner(cf)
	for scanner.Scan() {
		// Scan in the lines from the config file
		line := scanner.Text()
		if cfSkipRe.MatchString(line) {
			continue
		}
		// Split the line into fields
		f := strings.Fields(line)

		if f[0] == "filter" {
			if err := processFilterLine(f[1:], &newConfig); err != nil {
				return err
			}
		} else {
			if err := processConfigLine(f, &newConfig); err != nil {
				return err
			}
		}
	}

	// Residual config file scan error?
	if err := scanner.Err(); err != nil {
		return (err)
	}

	// Override the listen address:port if they were specified on the
	// command line.  If not and the listener values were not set in
	// the config file, fallback to defaults.
	//
	if teCmdLine.bindAddr != "" {
		newConfig.listenAddr = teCmdLine.bindAddr
	} else if newConfig.listenAddr == "" {
		newConfig.listenAddr = defBindAddr
	}
	if teCmdLine.listenPort != "" {
		newConfig.listenPort = teCmdLine.listenPort
	} else if newConfig.listenPort == "" {
		newConfig.listenPort = defListenPort
	}
	if teCmdLine.debugMode {
		newConfig.debug = true
	}
	// Other config fallbacks
	//
	if newConfig.logMaxSize == 0 {
		newConfig.logMaxSize = defLogfileMaxSize
	}
	if newConfig.logMaxBackups == 0 {
		newConfig.logMaxBackups = defLogfileMaxBackups
	}
	if newConfig.v3Params.username == "" {
		newConfig.v3Params.username = defV3user
	}
	if newConfig.v3Params.authProto == 0 {
		newConfig.v3Params.authProto = defV3authProtocol
	}
	if newConfig.v3Params.authPassword == "" {
		newConfig.v3Params.authPassword = defV3authPassword
	}
	if newConfig.v3Params.privacyProto == 0 {
		newConfig.v3Params.privacyProto = defV3privacyProtocol
	}
	if newConfig.v3Params.privacyPassword == "" {
		newConfig.v3Params.privacyPassword = defV3privacyPassword
	}
	// Sanity-check the v3 params
	//
	if (newConfig.v3Params.msgFlags&g.AuthPriv) == 1 && newConfig.v3Params.authProto < 2 {
		return fmt.Errorf("v3 config error: no auth protocol set when msgFlags specifies an Auth mode")
	}
	if newConfig.v3Params.msgFlags == g.AuthPriv && newConfig.v3Params.privacyProto < 2 {
		return fmt.Errorf("v3 config error: no privacy protocol mode set when msgFlags specifies an AuthPriv mode")
	}

	// Set our global config pointer to this configuration
	newConfig.teConfigured = true
	teConfig = &newConfig

	return nil
}

// processFIlterLine parsed a "filter" line from the config file and sets
// the appropriate values in the corresponding trapexFilter struct.
//
func processFilterLine(f []string, newConfig *trapexConfig) error {
	var err error
	if len(f) < 6 {
		return fmt.Errorf("not enough fields in filter line: %s", "filter "+strings.Join(f, " "))
	}

	// Process the filter criteria
	//
	filter := trapexFilter{}
	if strings.HasPrefix(strings.Join(f, " "), "* * * * *") {
		filter.matchAll = true
	} else {
		fObj := filterObj{}
		// Construct the filter criteria
		for i, fi := range f[:5] {
			if fi == "*" {
				continue
			}
			fObj.filterItem = i
			if i < 2 { // Either of the first 2 is an IP address type
				if strings.HasPrefix(fi, "/") { // If starts with a "/", it's a regex
					fObj.filterType = parseTypeRegex
					fObj.filterValue, err = regexp.Compile(fi[1:])
					if err != nil {
						return fmt.Errorf("unable to compile regexp for IP: %s: %s", fi, err)
					}
				} else if strings.Contains(fi, "/") {
					fObj.filterType = parseTypeCIDR
					fObj.filterValue, err = newNetwork(fi)
					if err != nil {
						return fmt.Errorf("invalid IP/CIDR: %s", fi)
					}
				} else {
					fObj.filterType = parseTypeString
					fObj.filterValue = fi
				}
			} else if i > 1 && i < 4 { // Generic and Specific type
				val, e := strconv.Atoi(fi)
				if e != nil {
					return fmt.Errorf("invalid integer value: %s: %s", fi, e)
				}
				fObj.filterType = parseTypeInt
				fObj.filterValue = val
			} else { // The enterprise OID
				fObj.filterType = parseTypeRegex
				fObj.filterValue, err = regexp.Compile(fi)
				if err != nil {
					return fmt.Errorf("unable to compile regexp for OID: %s: %s", fi, err)
				}
			}
			filter.filterItems = append(filter.filterItems, fObj)
		}
	}
	// Process the filter action
	//
	var actionArg string
	if len(f) > 6 {
		actionArg = f[6]
	}
	switch f[5] {
	case "break", "drop":
		filter.actionType = actionDrop
	case "nat":
		filter.actionType = actionNat
		if actionArg == "" {
			return fmt.Errorf("missing nat argument")
		}
		filter.actionArg = actionArg
	case "forward":
		filter.actionType = actionForward
		forwarder := trapForwarder{}
		if err := forwarder.initAction(actionArg); err != nil {
			return err
		}
		filter.action = &forwarder
	case "log":
		filter.actionType = actionLog
		logger := trapLogger{}
		if err := logger.initAction(actionArg, newConfig); err != nil {
			return err
		}
		filter.action = &logger
	default:
		return fmt.Errorf("unknown action: %s", f[5])
	}

	newConfig.filters = append(newConfig.filters, filter)

	return nil
}

func processConfigLine(f []string, newConfig *trapexConfig) error {
	flen := len(f)
	switch f[0] {
	case "debug":
		newConfig.debug = true
	case "logDroppedTraps":
		newConfig.logDropped = true
	case "listenAddress":
		if flen < 2 {
			return fmt.Errorf("missing value for listenAddr")
		}
		newConfig.listenAddr = f[1]
	case "listenPort":
		if flen < 2 {
			return fmt.Errorf("missing value for listenPort")
		}
		p, err := strconv.ParseUint(f[1], 10, 16)
		if err != nil || p < 1 || p > 65535 {
			return fmt.Errorf("invalid listenPort value: %s", err)
		}
		newConfig.listenPort = f[1]
	case "ignoreVersions":
		if flen < 2 {
			return fmt.Errorf("missing value for ignoreVersions")
		}
		// split on commas (if any)
		for _, v := range strings.Split(f[1], ",") {
			switch strings.ToLower(v) {
			case "v1", "1":
				newConfig.ignoreVersions = append(newConfig.ignoreVersions, g.Version1)
			case "v2c", "2c", "2":
				newConfig.ignoreVersions = append(newConfig.ignoreVersions, g.Version2c)
			case "v3", "3":
				newConfig.ignoreVersions = append(newConfig.ignoreVersions, g.Version3)
			default:
				return fmt.Errorf("unsupported or invalid value (%s) for ignoreVersion", v)
			}
		}
		if len(newConfig.ignoreVersions) > 2 {
			return fmt.Errorf("All 3 SNMP versions are ignored. There will be no traps to process")
		}
	case "v3msgFlags":
		if flen < 2 {
			return fmt.Errorf("missing value for v3msgFlags")
		}
		switch f[1] {
		case "NoAuthNoPriv":
			newConfig.v3Params.msgFlags = g.NoAuthNoPriv
		case "AuthNoPriv":
			newConfig.v3Params.msgFlags = g.AuthNoPriv
		case "AuthPriv":
			newConfig.v3Params.msgFlags = g.AuthPriv
		default:
			return fmt.Errorf("unsupported or invalid value (%s) for v3msgFlags", f[1])
		}
	case "v3user":
		if flen < 2 {
			return fmt.Errorf("missing value for v3user")
		}
		newConfig.v3Params.username = f[1]
	case "v3authProtocol":
		if flen < 2 {
			return fmt.Errorf("missing value for v3authProtocol")
		}
		switch f[1] {
		case "SHA":
			newConfig.v3Params.authProto = g.SHA
		case "MD5":
			newConfig.v3Params.authProto = g.MD5
		default:
			return fmt.Errorf("invalid value for v3authProtocol")
		}
	case "v3authPassword":
		if flen < 2 {
			return fmt.Errorf("missing value for v3authPassword")
		}
		newConfig.v3Params.authPassword = f[1]
	case "v3privacyProtocol":
		if flen < 2 {
			return fmt.Errorf("missing value for v3privacyProtocol")
		}
		switch f[1] {
		case "AES":
			newConfig.v3Params.privacyProto = g.AES
		case "DES":
			newConfig.v3Params.privacyProto = g.DES
		default:
			return fmt.Errorf("invalid value for v3privacyProtocol")
		}
	case "v3privacyPassword":
		if flen < 2 {
			return fmt.Errorf("missing value for v3privacyPassword")
		}
		newConfig.v3Params.privacyPassword = f[1]
	case "logfileMaxSize":
		if flen < 2 {
			return fmt.Errorf("missing value for logfileMaxSize")
		}
		p, err := strconv.Atoi(f[1])
		if err != nil || p < 1 {
			return fmt.Errorf("invalid logfileMaxSize value: %s", err)
		}
		newConfig.logMaxSize = p
	case "logfileMaxBackups":
		if flen < 2 {
			return fmt.Errorf("missing value for logfileMaxBackups")
		}
		p, err := strconv.Atoi(f[1])
		if err != nil || p < 1 {
			return fmt.Errorf("invalid logfileMaxBackups value: %s", err)
		}
		newConfig.logMaxBackups = p
	case "compressRotatedLogs":
		newConfig.logCompress = true
	default:
		return fmt.Errorf("Unknown/unsuppported configuration option: %s", f[0])
	}
	return nil
}

func closeTrapexHandles() {
	for _, f := range teConfig.filters {
		if f.actionType == actionForward {
			f.action.(*trapForwarder).close()
		}
	}
}
