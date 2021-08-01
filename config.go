// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	g "github.com/gosnmp/gosnmp"
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
	defV3user              string               = "XXv3Username"
	defV3authProtocol      g.SnmpV3AuthProtocol = g.NoAuth
	defV3authPassword      string               = "XXv3authPass"
	defV3privacyProtocol   g.SnmpV3PrivProtocol = g.NoPriv
	defV3privacyPassword   string               = "XXv3Pass"
)

type v3Params struct {
	msgFlags        g.SnmpV3MsgFlags
	username        string
	authProto       g.SnmpV3AuthProtocol
	authPassword    string
	privacyProto    g.SnmpV3PrivProtocol
	privacyPassword string
}

type ipSet map[string]bool

type trapexConfig struct {
	listenAddr     string
	listenPort     string
	ignoreVersions []g.SnmpVersion
	runLogFile     string
	configFile     string
	v3Params       v3Params
	filters        []trapexFilter
	debug          bool
	logMaxSize     int
	logMaxBackups  int
	logMaxAge      int
	logCompress    bool
	teConfigured   bool
	trapexHost     string
	ipSets         map[string]ipSet
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
		fmt.Printf("This is trapex version %s\n", myVersion)
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
	} else {
		fmt.Printf("Loading ")
	}
	fmt.Printf("configuration for trapex version %s from: %s.\n", myVersion, teCmdLine.configFile)

	var newConfig trapexConfig

	newConfig.ipSets = make(map[string]ipSet)

	// First process the config file
	cf, err := os.Open(teCmdLine.configFile)
	if err != nil {
		return err
	}
	defer cf.Close()

	cfSkipRe := regexp.MustCompile(`^\s*#|^\s*$`)
	ipRe := regexp.MustCompile(`^(?:\d{1,3}\.){3}\d{1,3}$`)

	var lineNumber uint = 0
	var processingIPSet bool = false
	var ipsName string

	scanner := bufio.NewScanner(cf)
	for scanner.Scan() {
		// Scan in the lines from the config file
		line := scanner.Text()
		lineNumber++
		if cfSkipRe.MatchString(line) {
			continue
		}

		// Split the line into fields
		f := strings.Fields(line)

		if processingIPSet {
			if f[0] == "}" {
				processingIPSet = false
				fmt.Printf("IP count: %v\n", len(newConfig.ipSets[ipsName]))
				continue
			}
			// Assume all fields are IP addresses
			for _, ip := range f {
				if ipRe.MatchString(ip) {
					newConfig.ipSets[ipsName][ip] = true
				} else {
					return fmt.Errorf("Invalid IP address (%s) in ipset: %s at line: %v", ip, ipsName, lineNumber)
				}
			}
		} else if f[0] == "ipset" {
			if len(f) < 3 || f[2] != "{" {
				return fmt.Errorf("Invalid format for ipset at line: %v: '%s'", lineNumber, line)
			}
			ipsName = f[1]
			newConfig.ipSets[ipsName] = make(map[string]bool)
			processingIPSet = true
			fmt.Printf(" -Add IPSet: %s - ", ipsName)
			continue
		} else if f[0] == "filter" {
			if err := processFilterLine(f[1:], &newConfig, lineNumber); err != nil {
				return err
			}
		} else {
			if err := processConfigLine(f, &newConfig, lineNumber); err != nil {
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
	if newConfig.trapexHost == "" {
		myName, err := os.Hostname()
		if err != nil {
			newConfig.trapexHost = "_undefined"
		} else {
			newConfig.trapexHost = myName
		}
	}
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

	// If this is a reconfigure, close the old handles here
	if teConfig != nil && teConfig.teConfigured {
		closeTrapexHandles()
	}
	// Set our global config pointer to this configuration
	newConfig.teConfigured = true
	teConfig = &newConfig

	return nil
}

// processFilterLine parsed a "filter" line from the config file and sets
// the appropriate values in the corresponding trapexFilter struct.
//
func processFilterLine(f []string, newConfig *trapexConfig, lineNumber uint) error {
	var err error
	if len(f) < 7 {
		return fmt.Errorf("not enough fields in filter line(%v): %s", lineNumber, "filter "+strings.Join(f, " "))
	}

	// Process the filter criteria
	//
	filter := trapexFilter{}
	if strings.HasPrefix(strings.Join(f, " "), "* * * * * *") {
		filter.matchAll = true
	} else {
		fObj := filterObj{}
		// Construct the filter criteria
		for i, fi := range f[:6] {
			if fi == "*" {
				continue
			}
			fObj.filterItem = i
			if i == 0 {
				switch strings.ToLower(fi) {
				case "v1", "1":
					fObj.filterValue = g.Version1
				case "v2c", "2c", "2":
					fObj.filterValue = g.Version2c
				case "v3", "3":
					fObj.filterValue = g.Version3
				default:
					return fmt.Errorf("unsupported or invalid SNMP version (%s) on line %v for filter", fi, lineNumber)
				}
				fObj.filterType = parseTypeInt // Just because we should set this to something.
			} else if i == 1 || i == 2 { // Either of the first 2 is an IP address type
				if strings.HasPrefix(fi, "ipset:") { // If starts with a "ipset:"" it's an IP set
					fObj.filterType = parseTypeIPSet
					if _, ok := newConfig.ipSets[fi[6:]]; ok {
						fObj.filterValue = fi[6:]
					} else {
						return fmt.Errorf("Invalid ipset name specified on line %v: %s", lineNumber, fi)
					}
				} else if strings.HasPrefix(fi, "/") { // If starts with a "/", it's a regex
					fObj.filterType = parseTypeRegex
					fObj.filterValue, err = regexp.Compile(fi[1:])
					if err != nil {
						return fmt.Errorf("unable to compile regexp for IP on line %v: %s: %s", lineNumber, fi, err)
					}
				} else if strings.Contains(fi, "/") {
					fObj.filterType = parseTypeCIDR
					fObj.filterValue, err = newNetwork(fi)
					if err != nil {
						return fmt.Errorf("invalid IP/CIDR at line %v: %s", lineNumber, fi)
					}
				} else {
					fObj.filterType = parseTypeString
					fObj.filterValue = fi
				}
			} else if i > 2 && i < 5 { // Generic and Specific type
				val, e := strconv.Atoi(fi)
				if e != nil {
					return fmt.Errorf("invalid integer value at line %v: %s: %s", lineNumber, fi, e)
				}
				fObj.filterType = parseTypeInt
				fObj.filterValue = val
			} else { // The enterprise OID
				fObj.filterType = parseTypeRegex
				fObj.filterValue, err = regexp.Compile(fi)
				if err != nil {
					return fmt.Errorf("unable to compile regexp at line %v for OID: %s: %s", lineNumber, fi, err)
				}
			}
			filter.filterItems = append(filter.filterItems, fObj)
		}
	}
	// Process the filter action
	//
	var actionArg string
	var breakAfter bool
	if len(f) > 8 && f[8] == "break" {
		breakAfter = true
	} else {
		breakAfter = false
	}

	var action = f[6]

	if len(f) > 7 {
		actionArg = f[7]
	}

	switch action {
	case "break", "drop":
		filter.actionType = actionBreak
	case "nat":
		filter.actionType = actionNat
		if actionArg == "" {
			return fmt.Errorf("missing nat argument at line %v", lineNumber)
		}
		filter.actionArg = actionArg
	case "forward":
		if breakAfter {
			filter.actionType = actionForwardBreak
		} else {
			filter.actionType = actionForward
		}
		forwarder := trapForwarder{}
		if err := forwarder.initAction(actionArg); err != nil {
			return err
		}
		filter.action = &forwarder
	case "log":
		if breakAfter {
			filter.actionType = actionLogBreak
		} else {
			filter.actionType = actionLog
		}
		logger := trapLogger{}
		if err := logger.initAction(actionArg, newConfig); err != nil {
			return err
		}
		filter.action = &logger
	case "csv":
		if breakAfter {
			filter.actionType = actionCsvBreak
		} else {
			filter.actionType = actionCsv
		}
		csvLogger := trapCsvLogger{}
		if err := csvLogger.initAction(actionArg, newConfig); err != nil {
			return err
		}
		filter.action = &csvLogger
	default:
		return fmt.Errorf("unknown action: %s at line %v", action, lineNumber)
	}

	newConfig.filters = append(newConfig.filters, filter)

	return nil
}

func processConfigLine(f []string, newConfig *trapexConfig, lineNumber uint) error {
	flen := len(f)
	switch f[0] {
	case "debug":
		newConfig.debug = true
	case "trapexHost":
		if flen < 2 {
			return fmt.Errorf("missing value for trapexHost at line %v", lineNumber)
		}
		newConfig.trapexHost = f[1]
	case "listenAddress":
		if flen < 2 {
			return fmt.Errorf("missing value for listenAddr at line %v", lineNumber)
		}
		newConfig.listenAddr = f[1]
	case "listenPort":
		if flen < 2 {
			return fmt.Errorf("missing value for listenPort at line %v", lineNumber)
		}
		p, err := strconv.ParseUint(f[1], 10, 16)
		if err != nil || p < 1 || p > 65535 {
			return fmt.Errorf("invalid listenPort value: %s at line %v", err, lineNumber)
		}
		newConfig.listenPort = f[1]
	case "ignoreVersions":
		if flen < 2 {
			return fmt.Errorf("missing value for ignoreVersions at line %v", lineNumber)
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
				return fmt.Errorf("unsupported or invalid value (%s) for ignoreVersion at line %v", v, lineNumber)
			}
		}
		if len(newConfig.ignoreVersions) > 2 {
			return fmt.Errorf("All 3 SNMP versions are ignored at line %v. There will be no traps to process", lineNumber)
		}
	case "v3msgFlags":
		if flen < 2 {
			return fmt.Errorf("missing value for v3msgFlags at line %v", lineNumber)
		}
		switch f[1] {
		case "NoAuthNoPriv":
			newConfig.v3Params.msgFlags = g.NoAuthNoPriv
		case "AuthNoPriv":
			newConfig.v3Params.msgFlags = g.AuthNoPriv
		case "AuthPriv":
			newConfig.v3Params.msgFlags = g.AuthPriv
		default:
			return fmt.Errorf("unsupported or invalid value (%s) for v3msgFlags at line %v", f[1], lineNumber)
		}
	case "v3user":
		if flen < 2 {
			return fmt.Errorf("missing value for v3user at line %v", lineNumber)
		}
		newConfig.v3Params.username = f[1]
	case "v3authProtocol":
		if flen < 2 {
			return fmt.Errorf("missing value for v3authProtocol at line %v", lineNumber)
		}
		switch f[1] {
		case "SHA":
			newConfig.v3Params.authProto = g.SHA
		case "MD5":
			newConfig.v3Params.authProto = g.MD5
		default:
			return fmt.Errorf("invalid value for v3authProtocol at line %v", lineNumber)
		}
	case "v3authPassword":
		if flen < 2 {
			return fmt.Errorf("missing value for v3authPassword at line %v", lineNumber)
		}
		newConfig.v3Params.authPassword = f[1]
	case "v3privacyProtocol":
		if flen < 2 {
			return fmt.Errorf("missing value for v3privacyProtocol at line %v", lineNumber)
		}
		switch f[1] {
		case "AES":
			newConfig.v3Params.privacyProto = g.AES
		case "DES":
			newConfig.v3Params.privacyProto = g.DES
		default:
			return fmt.Errorf("invalid value for v3privacyProtocol at line %v", lineNumber)
		}
	case "v3privacyPassword":
		if flen < 2 {
			return fmt.Errorf("missing value for v3privacyPassword at line %v", lineNumber)
		}
		newConfig.v3Params.privacyPassword = f[1]
	case "logfileMaxSize":
		if flen < 2 {
			return fmt.Errorf("missing value for logfileMaxSize at line %v", lineNumber)
		}
		p, err := strconv.Atoi(f[1])
		if err != nil || p < 1 {
			return fmt.Errorf("invalid logfileMaxSize value: %s at line %v", err, lineNumber)
		}
		newConfig.logMaxSize = p
	case "logfileMaxBackups":
		if flen < 2 {
			return fmt.Errorf("missing value for logfileMaxBackups at line %v", lineNumber)
		}
		p, err := strconv.Atoi(f[1])
		if err != nil || p < 1 {
			return fmt.Errorf("invalid logfileMaxBackups value: %s at line %v", err, lineNumber)
		}
		newConfig.logMaxBackups = p
	case "compressRotatedLogs":
		newConfig.logCompress = true
	default:
		return fmt.Errorf("Unknown/unsuppported configuration option: %s at line %v", f[0], lineNumber)
	}
	return nil
}

func closeTrapexHandles() {
	for _, f := range teConfig.filters {
		if f.actionType == actionForward || f.actionType == actionForwardBreak {
			f.action.(*trapForwarder).close()
		}
		if f.actionType == actionLog || f.actionType == actionLogBreak {
			f.action.(*trapLogger).close()
		}
		if f.actionType == actionCsv || f.actionType == actionCsvBreak {
			f.action.(*trapCsvLogger).close()
		}
	}
}
