package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	g "github.com/damienstuart/gosnmp"
)

// Default fallback values
//
const defBindAddr string = "0.0.0.0"
const defListenPort string = "162"

type trapAction interface {
	initAction(data string)
	processTrap(*sgTrap)
}

type trapForwarder struct {
	destination	*g.GoSNMP
}

type trapLogger struct {
	logFile			string
	logHandle		*bufio.Writer
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
	username		string
	authProto		g.SnmpV3AuthProtocol
	authPassword	string
	privacyProto	g.SnmpV3PrivProtocol
	privacyPassword	string
}

type trapexConfig struct {
	cmdArgs		cmdArgs
	listenAddr	string
	listenPort	string
	configFile	string
	v3Params	v3Params
	filters		[]trapexFilter
	debug 		bool
	logDropped  bool
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
	a.logHandle = bufio.NewWriter(fd)
	runLogger.Printf(" -Added log destination: %s\n", logfile)
}

func (a *trapLogger) processTrap(trap *sgTrap) {
	err := logr(trap, a.logHandle)
	if err != nil && !a.isBroken {
		runLogger.Printf("Error writing to logfile: %s\n", a.logFile)
		a.isBroken = true
	}
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
Usage: trapex [-c <config_file>] [-b <bind_ip>] [-p <listen_port>] [-d] [-h]
  -b  - Override the bind IP address on which to listen for incoming traps.
  -c  - Override the location of the trapex configuration file.
  -d  - Enable debug mode - which produces verbose output.
  -p  - Override the UDP port on which to listen for incoming traps.
  -v  - Print the version of trapex and exit.
  -h  - Show this help message.
`
	eprint(usageText)
}

func getConfig() {
	flag.Usage = showUsage
	configFile := flag.String("c", "/etc/trapex.conf", "")
	cmdBindAddr := flag.String("b", "", "")
	cmdListenPort := flag.String("p", "", "")
	debugMode := flag.Bool("d", false, "")
	showVersion := flag.Bool("v", false, "")

	flag.Parse()

	runLogger.Printf("This is trapex version %s.\n", myVersion)
	if *showVersion {
		os.Exit(0)
	}

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
	default:
		return fmt.Errorf("Unknown/unsuppported configuration option: %s", f[0])
	}
	return nil
}