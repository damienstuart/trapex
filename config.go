// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/creasty/defaults"
	g "github.com/gosnmp/gosnmp"
	"gopkg.in/yaml.v2"
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

type trapexCommandLine struct {
	configFile string
	bindAddr   string
	listenPort string
	debugMode  bool
}

// Global vars
//
var teConfig *TrapexConfig
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
func loadConfig(config_file string, newConfig *TrapexConfig) error {
	defaults.Set(newConfig)

	newConfig.IpSets = make(map[string]IpSet)

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

func applyCliOverrides(newConfig *TrapexConfig) {
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
	var operation string
	// If this is a reconfig close any current handles
	if teConfig != nil && teConfig.Configured {
		operation = "Reloading "
	} else {
		operation = "Loading "
	}
	trapex_logger.Info().Str("version", myVersion).Str("configuration_file", teCmdLine.configFile).Msg(operation + "configuration for trapex")

	var newConfig TrapexConfig
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
	if err = loadFilterActions(&newConfig); err != nil {
		return err
	}
	if err = processFilters(&newConfig); err != nil {
		return err
	}

	// If this is a reconfigure, close the old handles here
	if teConfig != nil && teConfig.Configured {
		closeTrapexHandles()
	}
	// Set our global config pointer to this configuration
	newConfig.Configured = true
	teConfig = &newConfig

	return nil
}

func validateIgnoreVersions(newConfig *TrapexConfig) error {
	var ignorev1, ignorev2c, ignorev3 bool = false, false, false
	for _, candidate := range newConfig.General.IgnoreVersions_str {
		switch strings.ToLower(candidate) {
		case "v1", "1":
			if ignorev1 != true {
				newConfig.General.IgnoreVersions = append(newConfig.General.IgnoreVersions, g.Version1)
				ignorev1 = true
			}
		case "v2c", "2c", "2":
			if ignorev2c != true {
				newConfig.General.IgnoreVersions = append(newConfig.General.IgnoreVersions, g.Version2c)
				ignorev2c = true
			}
		case "v3", "3":
			if ignorev3 != true {
				newConfig.General.IgnoreVersions = append(newConfig.General.IgnoreVersions, g.Version3)
				ignorev3 = true
			}
		default:
			return fmt.Errorf("unsupported or invalid value (%s) for general:ignore_version", candidate)
		}
	}
	if len(newConfig.General.IgnoreVersions) > 2 {
		return fmt.Errorf("All three SNMP versions are ignored -- there will be no traps to process")
	}
	return nil
}

func validateSnmpV3Args(newConfig *TrapexConfig) error {
	switch strings.ToLower(newConfig.V3Params.MsgFlags_str) {
	case "noauthnopriv":
		newConfig.V3Params.MsgFlags = g.NoAuthNoPriv
	case "authnopriv":
		newConfig.V3Params.MsgFlags = g.AuthNoPriv
	case "authpriv":
		newConfig.V3Params.MsgFlags = g.AuthPriv
	default:
		return fmt.Errorf("unsupported or invalid value (%s) for snmpv3:msg_flags", newConfig.V3Params.MsgFlags_str)
	}

	switch strings.ToLower(newConfig.V3Params.AuthProto_str) {
	// AES is *NOT* supported
	case "noauth":
		newConfig.V3Params.AuthProto = g.NoAuth
	case "sha":
		newConfig.V3Params.AuthProto = g.SHA
	case "md5":
		newConfig.V3Params.AuthProto = g.MD5
	default:
		return fmt.Errorf("invalid value for snmpv3:auth_protocol: %s", newConfig.V3Params.AuthProto_str)
	}

	switch strings.ToLower(newConfig.V3Params.PrivacyProto_str) {
	case "nopriv":
		newConfig.V3Params.PrivacyProto = g.NoPriv
	case "aes":
		newConfig.V3Params.PrivacyProto = g.AES
	case "des":
		newConfig.V3Params.PrivacyProto = g.DES
	default:
		return fmt.Errorf("invalid value for snmpv3:privacy_protocol: %s", newConfig.V3Params.PrivacyProto_str)
	}

	if (newConfig.V3Params.MsgFlags&g.AuthPriv) == 1 && newConfig.V3Params.AuthProto < 2 {
		return fmt.Errorf("v3 config error: no auth protocol set when snmpv3:msg_flags specifies an Auth mode")
	}
	if newConfig.V3Params.MsgFlags == g.AuthPriv && newConfig.V3Params.PrivacyProto < 2 {
		return fmt.Errorf("v3 config error: no privacy protocol mode set when snmpv3:msg_flags specifies an AuthPriv mode")
	}

	return nil
}

func processIpSets(newConfig *TrapexConfig) error {
	for _, stanza := range newConfig.IpSets_str {
		for ipsName, ips := range stanza {
			trapex_logger.Debug().Str("ipset", ipsName).Msg("Loading IpSet")
			newConfig.IpSets[ipsName] = make(map[string]bool)
			for _, ip := range ips {
				if ipRe.MatchString(ip) {
					newConfig.IpSets[ipsName][ip] = true
					trapex_logger.Debug().Str("ipset", ipsName).Str("ip", ip).Msg("Adding IP to IpSet")
				} else {
					return fmt.Errorf("Invalid IP address (%s) in ipset: %s", ip, ipsName)
				}
			}
		}
	}
	return nil
}

func processFilters(newConfig *TrapexConfig) error {

	for lineNumber, filter_line := range newConfig.Filters_str {
		trapex_logger.Debug().Str("filter", filter_line).Int("line_number", lineNumber).Msg("Examining filter")
		if err := processFilterLine(strings.Fields(filter_line), newConfig, lineNumber); err != nil {
			return err
		}
	}
	return nil
}

// processFilterLine parses a "filter" line and sets
// the appropriate values in a corresponding trapexFilter struct.
//
func processFilterLine(f []string, newConfig *TrapexConfig, lineNumber int) error {
	var err error
	if len(f) < 7 {
		return fmt.Errorf("not enough fields in filter line(%v): %s", lineNumber, "filter "+strings.Join(f, " "))
	}

	// Process the filter criteria
	//
	filter := TrapexFilter{}
	if strings.HasPrefix(strings.Join(f, " "), "* * * * * *") {
		filter.MatchAll = true
	} else {
		fObj := FilterObj{}
		// Construct the filter criteria
		for i, fi := range f[:6] {
			if fi == "*" {
				continue
			}
			fObj.FilterItem = i
			if i == 0 {
				switch strings.ToLower(fi) {
				case "v1", "1":
					fObj.FilterValue = g.Version1
				case "v2c", "2c", "2":
					fObj.FilterValue = g.Version2c
				case "v3", "3":
					fObj.FilterValue = g.Version3
				default:
					return fmt.Errorf("unsupported or invalid SNMP version (%s) on line %v for filter: %s", fi, lineNumber, f)
				}
				fObj.FilterType = parseTypeInt // Just because we should set this to something.
			} else if i == 1 || i == 2 { // Either of the first 2 is an IP address type
				if strings.HasPrefix(fi, "ipset:") { // If starts with a "ipset:"" it's an IP set
					fObj.FilterType = parseTypeIPSet
					if _, ok := newConfig.IpSets[fi[6:]]; ok {
						fObj.FilterValue = fi[6:]
					} else {
						return fmt.Errorf("Invalid ipset name specified on line %v: %s: %s", lineNumber, fi, f)
					}
				} else if strings.HasPrefix(fi, "/") { // If starts with a "/", it's a regex
					fObj.FilterType = parseTypeRegex
					fObj.FilterValue, err = regexp.Compile(fi[1:])
					if err != nil {
						return fmt.Errorf("unable to compile regexp for IP on line %v: %s: %s\n%s", lineNumber, fi, err, f)
					}
				} else if strings.Contains(fi, "/") {
					fObj.FilterType = parseTypeCIDR
					fObj.FilterValue, err = newNetwork(fi)
					if err != nil {
						return fmt.Errorf("invalid IP/CIDR at line %v: %s, %s", lineNumber, fi, f)
					}
				} else {
					fObj.FilterType = parseTypeString
					fObj.FilterValue = fi
				}
			} else if i > 2 && i < 5 { // Generic and Specific type
				val, e := strconv.Atoi(fi)
				if e != nil {
					return fmt.Errorf("invalid integer value at line %v: %s: %s", lineNumber, fi, e)
				}
				fObj.FilterType = parseTypeInt
				fObj.FilterValue = val
			} else { // The enterprise OID
				fObj.FilterType = parseTypeRegex
				fObj.FilterValue, err = regexp.Compile(fi)
				if err != nil {
					return fmt.Errorf("unable to compile regexp at line %v for OID: %s: %s", lineNumber, fi, err)
				}
			}
			filter.FilterItems = append(filter.FilterItems, fObj)
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
		filter.ActionType = actionBreak
	case "nat":
		filter.ActionType = actionNat
		if actionArg == "" {
			return fmt.Errorf("missing nat argument at line %v", lineNumber)
		}
		filter.ActionArg = actionArg
	case "forward":
		if breakAfter {
			filter.ActionType = actionForwardBreak
		} else {
			filter.ActionType = actionForward
		}
		forwarder := trapForwarder{}
		if err := forwarder.initAction(actionArg); err != nil {
			return err
		}
		filter.Action = &forwarder
	case "log":
		if breakAfter {
			filter.ActionType = actionLogBreak
		} else {
			filter.ActionType = actionLog
		}
		logger := trapLogger{}
		if err := logger.initAction(actionArg, newConfig); err != nil {
			return err
		}
		filter.Action = &logger
	case "csv":
		if breakAfter {
			filter.ActionType = actionCsvBreak
		} else {
			filter.ActionType = actionCsv
		}
		csvLogger := trapCsvLogger{}
		if err := csvLogger.initAction(actionArg, newConfig); err != nil {
			return err
		}
		filter.Action = &csvLogger
	default:
		return fmt.Errorf("unknown action: %s at line %v", action, lineNumber)
	}

	newConfig.filters = append(newConfig.filters, filter)

	return nil
}

func closeTrapexHandles() {
	for _, f := range teConfig.filters {
		if f.ActionType == actionForward || f.ActionType == actionForwardBreak {
			f.Action.(*trapForwarder).close()
		}
		if f.ActionType == actionLog || f.ActionType == actionLogBreak {
			f.Action.(*trapLogger).close()
		}
		if f.ActionType == actionCsv || f.ActionType == actionCsvBreak {
			f.Action.(*trapCsvLogger).close()
		}
	}
}
