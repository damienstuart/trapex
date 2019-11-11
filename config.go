package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/damienstuart/lumberjack"
	g "github.com/damienstuart/gosnmp"
)

// Default fallback values
//
const (
	defBindAddr string = "0.0.0.0"
	defListenPort string = "162"
	defLogfileMaxSize int = 1024
	defLogfileMaxBackups int = 7
	defCompressRotatedLogs bool = false
	defV3MsgFlag g.SnmpV3MsgFlags = g.NoAuthNoPriv
	defV3user string = "_v3user"
	defV3authProtocol g.SnmpV3AuthProtocol = g.NoAuth
	defV3authPassword string = "_v3password"
	defV3privacyProtocol g.SnmpV3PrivProtocol =  g.NoPriv
	defV3privacyPassword string =  "_v3password"
)

type trapAction interface {
	initAction(data string)
	processTrap(*sgTrap)
}

type trapForwarder struct {
	destination	*g.GoSNMP
}

type trapLogger struct {
	logFile			string
	//logHandle		*bufio.Writer
	logHandle		*log.Logger
	isBroken		bool
}

// Filter types
const (
	parseTypeAny int = iota	// Match anything (wildcard)
	parseTypeString			// Direct String comparison
	parseTypeInt			// Direct Integer comparison
	parseTypeRegex			// Regular Expression
	parseTypeCIDR			// CIDR IP/Netmask 
	parseTypeRange			// Integer range x:y
)

// Filter items
const (
	srcIP int = iota
	agentAddr
	genericType
	specificType
	enterprise
)

const (
	actionDrop int = iota
	actionNat
	actionForward
	actionLog
)

type filterObj struct {
	filterItem	int
	filterType	int
	filterValue	interface{}		// string, *regex.Regexp, *network, int
}

type trapexFilter struct {
	filterItems	[]filterObj
	matchAll	bool
	action		interface{}
	actionType	int
	actionArg	string
}

type cmdArgs struct {
	bindAddr	string
	listenPort	string
	configFile	string
}

type v3Params struct {
	msgFlags		g.SnmpV3MsgFlags
	username		string
	authProto		g.SnmpV3AuthProtocol
	authPassword	string
	privacyProto	g.SnmpV3PrivProtocol
	privacyPassword	string
}

type trapexConfig struct {
	cmdArgs			cmdArgs
	listenAddr		string
	listenPort		string
	runLogFile		string
	runLogger		*log.Logger
	configFile		string
	v3Params		v3Params
	filters			[]trapexFilter
	debug			bool
	logDropped	 	bool
	logMaxSize		int
	logMaxBackups	int
	logMaxAge		int
	logCompress		bool
}

// Global vars
var teConfig trapexConfig

func (a *trapForwarder) initAction(dest string) {
	s := strings.Split(dest, ":")
	port, err := strconv.Atoi(s[1])
	if err != nil {
		panic("Invalid destination port: " + s[1])
	}
	a.destination = &g.GoSNMP{
		Target:             s[0],
		Port:               uint16(port),
		Transport:          "udp",
		Community:          "",
		Version:            g.Version1,
		Timeout:            time.Duration(2) * time.Second,
		Retries:            3,
		ExponentialTimeout: true,
		MaxOids:            g.MaxOids,
	}
	err = a.destination.Connect()
	if err != nil {
		panic(err)
	}
	runLogger.Printf(" -Added trap destination: %s, port %s\n", s[0], s[1])
}

func (a trapForwarder) processTrap(trap *sgTrap) error {
	_, err := a.destination.SendTrap(trap.data)
	return err
}

func (a *trapLogger) initAction(logfile string) {
	fd, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	checkErr(err)
	a.logFile = logfile
	a.logHandle = log.New(fd, "", 0)
	a.logHandle.SetOutput(makeLogger(logfile))
	runLogger.Printf(" -Added log destination: %s\n", logfile)
}

func (a *trapLogger) processTrap(trap *sgTrap) {
	logTrap(trap, a.logHandle)
}

func makeLogger(logfile string) *lumberjack.Logger {
	// --DSS TODO: use config params
	l := lumberjack.Logger{
		Filename:	logfile,
		MaxSize: 	teConfig.logMaxSize,
		MaxBackups: teConfig.logMaxBackups,
		Compress:	teConfig.logCompress,
	}
	return &l
}

func checkErr(e error) {
	if e != nil {
		panic(e)
	}
}

func eprint(msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", msg)
}

func showUsage() {
	usageText := `
Usage: trapex [-h] [-c <config_file>] [-b <bind_ip>] [-p <listen_port>]
              [-d] [-r <runtime_logfil>] [-v]
  -h  - Show this help message and exit.
  -b  - Override the bind IP address on which to listen for incoming traps.
  -c  - Override the location of the trapex configuration file.
  -d  - Enable debug mode (note: produces very verbose runtime output).
  -p  - Override the UDP port on which to listen for incoming traps.
  -r  - Speficy the run logfile (runtime and debug output). If not
        set, output goes to stdout.
  -v  - Print the version of trapex and exit.
`
	eprint(usageText)
}

func getConfig() {
	flag.Usage = showUsage
	configFile := flag.String("c", "/etc/trapex.conf", "")
	cmdBindAddr := flag.String("b", "", "")
	cmdListenPort := flag.String("p", "", "")
	cmdRunLogFile := flag.String("r", "", "")
	debugMode := flag.Bool("d", false, "")
	showVersion := flag.Bool("v", false, "")

	flag.Parse()

	if *showVersion {
		fmt.Printf("This is trapex version %s.\n", myVersion)
		os.Exit(0)
	}

	if *cmdRunLogFile != "" {
		fd, err := os.OpenFile(*cmdRunLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		checkErr(err)
		runLogger = log.New(fd, "", 0)
	} else {
		runLogger = log.New(os.Stdout, "", 0)
	}

	runLogger.Printf("This is trapex version %s.\n", myVersion)
	runLogger.Printf("-Reading configuration from: %s.\n", *configFile)

	// First process the config file
	cf, err := os.Open(*configFile)
	checkErr(err)
	defer cf.Close()

	cfSkipRe := regexp.MustCompile(`^\s*#|^\s*$`)

	scanner := bufio.NewScanner(cf)
	for scanner.Scan(){
		// Scan in the lines from the config file
		line := scanner.Text()
		if cfSkipRe.MatchString(line) {
			continue
		}
		// Split the line into fields
		f := strings.Fields(line)

		if f[0] == "filter" {
			err := processFilterLine(f[1:])
			checkErr(err)
		} else {
			err := processConfigLine(f)
			checkErr(err)
		}
	}

	// Residual config file scan error?
	if err := scanner.Err(); err != nil {
		checkErr(err)
	}

	// Override the listen address:port if they were specified on the
	// command line.  If not and the listener values were not set in
	// the config file, fallback to defaults.
	//
	if *cmdBindAddr != "" {
		teConfig.listenAddr = *cmdBindAddr
	} else if teConfig.listenAddr == "" {
			teConfig.listenAddr = defBindAddr
	}
	if *cmdListenPort != "" {
		teConfig.listenPort = *cmdListenPort
	} else if teConfig.listenPort == "" {
		teConfig.listenPort = defListenPort
	}
	if *debugMode == true {
		teConfig.debug = true
	}
	// Other config fallbacks
	if teConfig.logMaxSize == 0 {
		teConfig.logMaxSize = defLogfileMaxSize
	}
	if teConfig.logMaxBackups == 0 {
		teConfig.logMaxBackups = defLogfileMaxBackups
	}
	if *cmdRunLogFile != "" {
		runLogger.SetOutput(makeLogger(*cmdRunLogFile))
	}
	if teConfig.v3Params.username == "" {
		teConfig.v3Params.username = defV3user
	}
	if teConfig.v3Params.authProto == 0 {
		teConfig.v3Params.authProto = defV3authProtocol
	}
	if teConfig.v3Params.authPassword == "" {
		teConfig.v3Params.authPassword = defV3authPassword
	}
	if teConfig.v3Params.privacyProto == 0 {
		teConfig.v3Params.privacyProto = defV3privacyProtocol
	}
	if teConfig.v3Params.privacyPassword == "" {
		teConfig.v3Params.privacyPassword = defV3privacyPassword
	}
	// Now we need to sanity-check the v3 params
	if (teConfig.v3Params.msgFlags & g.AuthPriv) == 1 && teConfig.v3Params.authProto < 2 {
		checkErr(fmt.Errorf("v3 config error: no auth protocol set when msgFlags specifies an Auth mode"))
	}
	if teConfig.v3Params.msgFlags == g.AuthPriv && teConfig.v3Params.privacyProto < 2 {
		checkErr(fmt.Errorf("v3 config error: no privacy protocol mode set when msgFlags specifies an AuthPriv mode"))
	}
}

func processFilterLine(f []string) error {
	var err error
	if len(f) < 6 {
		return fmt.Errorf("not enough fields in filter line: %s", "filter " + strings.Join(f, " "))
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
			if i < 2 { // This is an IP address type
				if strings.Contains(fi, "/") {
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
					return fmt.Errorf("unable to compile regexp for: %s: %s", fi, err)
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
		forwarder.initAction(actionArg)
		filter.action = &forwarder 
	case "log":
		filter.actionType = actionLog
		logger := trapLogger{}
		logger.initAction(actionArg)
		filter.action = &logger
	default:
		return fmt.Errorf("unknown action: %s", f[5])
	}

	teConfig.filters = append(teConfig.filters, filter)

	return nil
}

func processConfigLine(f []string) error {
	flen := len(f)
	switch f[0] {
	case "debug":
		teConfig.debug = true
	case "logDroppedTraps":
		teConfig.logDropped = true
	case "listenAddress":
		if flen < 2 {
			return fmt.Errorf("missing value for listenAddr")
		}
		teConfig.listenAddr = f[1]
	case "listenPort":
		if flen < 2 {
			return fmt.Errorf("missing value for listenPort")
		}
		p, err := strconv.ParseUint(f[1],10,16)
		if err != nil || p < 1 || p > 65535 {
			return fmt.Errorf("invalid listenPort value: %s", err)
		}
		teConfig.listenPort = f[1]
	case "v3msgFlags":
		if flen < 2 {
			return fmt.Errorf("missing value for v3msgFlags")
		}
		switch f[1] {
		case "NoAuthNoPriv":
			teConfig.v3Params.msgFlags = g.NoAuthNoPriv
		case "AuthNoPriv":
			teConfig.v3Params.msgFlags = g.AuthNoPriv
		case "AuthPriv":
			teConfig.v3Params.msgFlags = g.AuthPriv
		default:
			return fmt.Errorf("unsupported or invalid value (%s) for v3msgFlags", f[1])
		}
	case "v3user":
		if flen < 2 {
			return fmt.Errorf("missing value for v3user")
		}
		teConfig.v3Params.username = f[1]
	case "v3authProtocol":
		if flen < 2 {
			return fmt.Errorf("missing value for v3authProtocol")
		}
		if f[1] == "SHA" {
			teConfig.v3Params.authProto = g.SHA
		} else if f[1] == "MD5" {
			teConfig.v3Params.authProto = g.MD5
		} else {
			return fmt.Errorf("invalid value for v3authProtocol")
		}
	case "v3authPassword":
		if flen < 2 {
			return fmt.Errorf("missing value for v3authPassword")
		}
		teConfig.v3Params.authPassword = f[1]
	case "v3privacyProtocol":
		if flen < 2 {
			return fmt.Errorf("missing value for v3privacyProtocol")
		}
		if f[1] == "AES" {
			teConfig.v3Params.privacyProto = g.AES
		} else if f[1] == "DES" {
			teConfig.v3Params.privacyProto = g.DES
		} else {
			return fmt.Errorf("invalid value for v3privacyProtocol")
		}
	case "v3privacyPassword":
		if flen < 2 {
			return fmt.Errorf("missing value for v3privacyPassword")
		}
		teConfig.v3Params.privacyPassword = f[1]
	case "logfileMaxSize":
		if flen < 2 {
			return fmt.Errorf("missing value for logfileMaxSize")
		}
		p, err := strconv.Atoi(f[1])
		if err != nil || p < 1 {
			return fmt.Errorf("invalid logfileMaxSize value: %s", err)
		}
		teConfig.logMaxSize = p
	case "logfileMaxBackups":
		if flen < 2 {
			return fmt.Errorf("missing value for logfileMaxBackups")
		}
		p, err := strconv.Atoi(f[1])
		if err != nil || p < 1 {
			return fmt.Errorf("invalid logfileMaxBackups value: %s", err)
		}
		teConfig.logMaxBackups = p
	case "compressRotatedLogs":
		teConfig.logCompress = true
	default:
		return fmt.Errorf("Unknown/unsuppported configuration option: %s", f[0])
	}
	return nil
}