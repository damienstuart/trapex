// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"flag"
        "io/ioutil"
        "path/filepath"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

        "github.com/creasty/defaults"
        "gopkg.in/yaml.v2"
	g "github.com/gosnmp/gosnmp"
)


/* ===========================================================
Notes on YAML configuration processing:
 * Variables that start with capital letters are processed (at least, for JSON)
 * Renaming of variables for the YAML file is done with the `yaml:` directives
 * Renamed variables *must* be in quotes to be recognized correctly (at least for underscores)
 * Default values are being applied with the creasty/defaults module
 * Non-basic types and classes can't be instantiated directly (eg g.SHA)
     * Configuration data structures have two sets of variables: text and usable
     * Per convention, the text versions start with uppercase, the usable ones start lowercase
 * Filter lines are very problematic for YAML
     * some characters (I'm looking at you ':' -- also regex) cause YAML to barf
     * using a more YAML-like structure will eat up huge chunks of configuration lines
        eg
         * * * * * ^1\.3\.6\.1\.4.1\.546\.1\.1 break

         vs

         - snmpversions: *
           source_ip: *
           agent_address: *
           ...
   ===========================================================
*/

type v3Params struct {
	MsgFlags        string `default:"NoAuthNoPriv" yaml:"msg_flags"`
	msgFlags        g.SnmpV3MsgFlags `default:"g.NoAuthNoPriv"`
	Username        string `default:"XXv3Username" yaml:"username"`
	AuthProto       string `default:"NoAuth" yaml:"auth_protocol"`
	authProto       g.SnmpV3AuthProtocol `default:"g.NoAuth"`
	AuthPassword    string `default:"XXv3authPass" yaml:"auth_password"`
	PrivacyProto    string `default:"NoPriv" yaml:"privacy_protocol"`
	privacyProto    g.SnmpV3PrivProtocol `default:"g.NoPriv"`
	PrivacyPassword string `default:"XXv3Pass" yaml:"privacy_password"`
}

type ipSet map[string]bool

type trapexConfig struct {
	teConfigured   bool
	runLogFile     string
	configFile     string

  General struct {
	Hostname     string `yaml:"hostname"`
	ListenAddr     string `default:"0.0.0.0" yaml:"listen_address"`
	ListenPort     string `default:"162" yaml:"listen_port"`

	IgnoreVersions []string `default:"[]" yaml:"ignore_versions"`
	ignoreVersions []g.SnmpVersion `default:"[]"`

	PrometheusIp   string `default:"0.0.0.0" yaml:"prometheus_ip"`
	PrometheusPort string `default:"80" yaml:"prometheus_port"`
	PrometheusEndpoint string `default:"metrics" yaml:"prometheus_endpoint"`
  }

  Logging struct {
	Level          string `default:"debug" yaml:"level"`
	LogMaxSize     int `default:"1024" yaml:"log_size_max"`
	LogMaxBackups  int `default:"7" yaml:"log_backups_max"`
	LogMaxAge      int `yaml:"log_age_max"`
	LogCompress    bool `default:"false" yaml:"compress_rotated_logs"`
  }

	V3Params       v3Params `yaml:"snmpv3"`

	IpSets         []map[string][]string `default:"{}" yaml:"ipsets"`
	ipSets         map[string]ipSet `default:"{}"`

	RawFilters     []string `default:"[]" yaml:"filters"`
	filters        []trapexFilter
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
var ipRe = regexp.MustCompile(`^(?:\d{1,3}\.){3}\d{1,3}$`)


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

// loadConfig
// Load a YAML file with configuration, and create a new object
func loadConfig(config_file string, newConfig *trapexConfig) error {
        defaults.Set(newConfig)

	newConfig.ipSets = make(map[string]ipSet)

        filename, _ := filepath.Abs(config_file)
        yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
            return err
	}
	err = yaml.UnmarshalStrict(yamlFile, newConfig)
	if err != nil {
            return err
	}

    return nil
}


func applyCliOverrides(newConfig *trapexConfig) {
        // Override the listen address:port if they were specified on the
        // command line.  If not and the listener values were not set in
        // the config file, fallback to defaults.
        if teCmdLine.bindAddr != "" {
                newConfig.General.ListenAddr = teCmdLine.bindAddr
        }
        if teCmdLine.listenPort != "" {
                newConfig.General.ListenPort = teCmdLine.listenPort
        }
        if teCmdLine.debugMode {
                newConfig.Logging.Level = "debug"
        }
        if newConfig.General.Hostname == "" {
                myName, err := os.Hostname()
                if err != nil {
                        newConfig.General.Hostname = "_undefined"
                } else {
                        newConfig.General.Hostname = myName
                }
        }
}


func getConfig() error {
	// If this is a reconfig close any current handles
	if teConfig != nil && teConfig.teConfigured {
		fmt.Printf("Reloading ")
	} else {
		fmt.Printf("Loading ")
	}
	fmt.Printf("configuration for trapex version %s from %s\n", myVersion, teCmdLine.configFile)

	var newConfig trapexConfig
        err := loadConfig(teCmdLine.configFile, &newConfig)
        if err != nil {
            return err
        }
        applyCliOverrides(&newConfig)

        if err = validateIgnoreVersions(&newConfig); err != nil {
            return err
        }
        if err = validateSnmpV3Args(&newConfig); err != nil {
            return err
        }
        if err = processIpSets(&newConfig); err != nil {
            return err
        }
        if err = processFilters(&newConfig); err != nil {
            return err
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

func validateIgnoreVersions(newConfig *trapexConfig) error {
    var ignorev1, ignorev2c, ignorev3 bool = false, false, false
    for _, candidate := range newConfig.General.IgnoreVersions {
        switch strings.ToLower(candidate) {
            case "v1", "1":
                if ignorev1 != true {
                  newConfig.General.ignoreVersions = append(newConfig.General.ignoreVersions, g.Version1)
                  ignorev1 = true
                }
            case "v2c", "2c", "2":
                if ignorev2c != true {
                  newConfig.General.ignoreVersions = append(newConfig.General.ignoreVersions, g.Version2c)
                  ignorev2c = true
                }
            case "v3", "3":
                if ignorev3 != true {
                  newConfig.General.ignoreVersions = append(newConfig.General.ignoreVersions, g.Version3)
                  ignorev3 = true
                }
            default:
                  return fmt.Errorf("unsupported or invalid value (%s) for general:ignore_version", candidate)
            }
    }
    if len(newConfig.General.ignoreVersions) > 2 {
        return fmt.Errorf("All three SNMP versions are ignored -- there will be no traps to process")
    }
    return nil
}


func validateSnmpV3Args(newConfig *trapexConfig) error {
    switch strings.ToLower(newConfig.V3Params.MsgFlags) {
       case "noauthnopriv":
           newConfig.V3Params.msgFlags = g.NoAuthNoPriv
       case "authnopriv":
           newConfig.V3Params.msgFlags = g.AuthNoPriv
       case "authpriv":
           newConfig.V3Params.msgFlags = g.AuthPriv
       default:
           return fmt.Errorf("unsupported or invalid value (%s) for snmpv3:msg_flags", newConfig.V3Params.MsgFlags)
    }

    switch strings.ToLower(newConfig.V3Params.AuthProto) {
        // AES is *NOT* supported
        case "aes":
            //newConfig.V3Params.authProto = g.AES
            //  cannot use gosnmp.AES (type gosnmp.SnmpV3PrivProtocol) as type gosnmp.SnmpV3AuthProtocol in assignment
            return fmt.Errorf("AES is not a supported value for snmpv3:auth_protocol")
        case "sha":
            newConfig.V3Params.authProto = g.SHA
        case "md5":
            newConfig.V3Params.authProto = g.MD5
        default:
            return fmt.Errorf("invalid value for snmpv3:auth_protocol")
    }

    switch strings.ToLower(newConfig.V3Params.PrivacyProto) {
        case "aes":
            newConfig.V3Params.privacyProto = g.AES
        case "des":
            newConfig.V3Params.privacyProto = g.DES
        default:
            return fmt.Errorf("invalid value for snmpv3:privacy_protocol")
    }

    if (newConfig.V3Params.msgFlags & g.AuthPriv) == 1 && newConfig.V3Params.authProto < 2 {
            return fmt.Errorf("v3 config error: no auth protocol set when snmpv3:msg_flags specifies an Auth mode")
    }
    if newConfig.V3Params.msgFlags == g.AuthPriv && newConfig.V3Params.privacyProto < 2 {
            return fmt.Errorf("v3 config error: no privacy protocol mode set when snmpv3:msg_flags specifies an AuthPriv mode")
    }

    return nil
}

func processIpSets(newConfig *trapexConfig) error {
   for _, stanza := range newConfig.IpSets {
   //for stanza_num, stanza := range newConfig.IpSets {
       //fmt.Printf("IpSet stanza %d: %s\n", stanza_num, stanza)
       for ipsName, ips := range stanza {
           //fmt.Printf("IpSet entry %s: %s\n", ipsName, ips)
           newConfig.ipSets[ipsName] = make(map[string]bool)
           //fmt.Printf(" -Add IPSet: %s - ", ipsName)
           for _, ip := range ips {
               if ipRe.MatchString(ip) {
                   newConfig.ipSets[ipsName][ip] = true
                   //fmt.Printf(" -IPSet %s:  %s\n", ipsName, ip)
               } else {
                   return fmt.Errorf("Invalid IP address (%s) in ipset: %s", ip, ipsName)
               }
           }
       }
   }
  return nil
}

func processFilters(newConfig *trapexConfig) error {

   for lineNumber, filter_line := range newConfig.RawFilters {
       //fmt.Printf("Filter %d: %s\n", lineNumber, filter_line)
       if err := processFilterLine(strings.Fields(filter_line), newConfig, lineNumber); err != nil {
           return err
       }
   }
  return nil
}


// processFilterLine parses a "filter" line and sets
// the appropriate values in a corresponding trapexFilter struct.
//
func processFilterLine(f []string, newConfig *trapexConfig, lineNumber int) error {
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
					return fmt.Errorf("unsupported or invalid SNMP version (%s) on line %v for filter: %s", fi, lineNumber, f)
				}
				fObj.filterType = parseTypeInt // Just because we should set this to something.
			} else if i == 1 || i == 2 { // Either of the first 2 is an IP address type
				if strings.HasPrefix(fi, "ipset:") { // If starts with a "ipset:"" it's an IP set
					fObj.filterType = parseTypeIPSet
/*
					if _, ok := newConfig.IpSets[fi[6:]]; ok {
						fObj.filterValue = fi[6:]
					} else {
						return fmt.Errorf("Invalid ipset name specified on line %v: %s: %s", lineNumber, fi, f)
					}
*/
				} else if strings.HasPrefix(fi, "/") { // If starts with a "/", it's a regex
					fObj.filterType = parseTypeRegex
					fObj.filterValue, err = regexp.Compile(fi[1:])
					if err != nil {
						return fmt.Errorf("unable to compile regexp for IP on line %v: %s: %s\n%s", lineNumber, fi, err, f)
					}
				} else if strings.Contains(fi, "/") {
					fObj.filterType = parseTypeCIDR
					fObj.filterValue, err = newNetwork(fi)
					if err != nil {
						return fmt.Errorf("invalid IP/CIDR at line %v: %s, %s", lineNumber, fi, f)
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

