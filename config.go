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
	"strings"

	pluginMeta "github.com/damienstuart/trapex/txPlugins"
	pluginLoader "github.com/damienstuart/trapex/txPlugins/interfaces"

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
     * Per convention, the text versions have a suffix of _str
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
	c := flag.String("c", "/opt/trapex/etc/trapex.yml", "")
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

func applyCliOverrides(newConfig *trapexConfig) {
	// Override the listen address:port if they were specified on the
	// command line.  If not and the listener values were not set in
	// the config file, fallback to defaults.
	if teCmdLine.bindAddr != "" {
		newConfig.TrapReceiverSettings.ListenAddr = teCmdLine.bindAddr
	}
	if teCmdLine.listenPort != "" {
		newConfig.TrapReceiverSettings.ListenPort = teCmdLine.listenPort
	}
	if teCmdLine.debugMode {
		newConfig.Logging.Level = "debug"
	}
	if newConfig.TrapReceiverSettings.Hostname == "" {
		myName, err := os.Hostname()
		if err != nil {
			newConfig.TrapReceiverSettings.Hostname = "_undefined"
		} else {
			newConfig.TrapReceiverSettings.Hostname = myName
		}
	}
}

func getConfig() error {
	var operation string
	// If this is a reconfig close any current handles
	if teConfig != nil && teConfig.teConfigured {
		operation = "Reloading"
	} else {
		operation = "Loading"
	}
	trapexLog.Info().Str("version", myVersion).Str("configuration_file", teCmdLine.configFile).Msg(operation + " configuration for trapex")

	var newConfig trapexConfig
	err := loadConfig(teCmdLine.configFile, &newConfig)
	if err != nil {
		return err
	}
	applyCliOverrides(&newConfig)

	if err = validateIgnoreVersions(&newConfig); err != nil {
		return err
	}
	if err = validateSnmpV3Args(&newConfig.TrapReceiverSettings); err != nil {
		return err
	}
	if err = addIpSets(&newConfig); err != nil {
		return err
	}
	if err = addFilters(&newConfig); err != nil {
		return err
	}

	// Obviously, the user really shouldn't use the same plugins, but....
	if err = addPluginErrorActions(&newConfig); err != nil {
		return err
	}

	if err = addReportingPlugins(&newConfig); err != nil {
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
	for _, candidate := range newConfig.TrapReceiverSettings.IgnoreVersions_str {
		switch strings.ToLower(candidate) {
		case "v1", "1":
			if !ignorev1 {
				newConfig.TrapReceiverSettings.IgnoreVersions = append(newConfig.TrapReceiverSettings.IgnoreVersions, g.Version1)
				ignorev1 = true
			}
		case "v2c", "2c", "2":
			if !ignorev2c {
				newConfig.TrapReceiverSettings.IgnoreVersions = append(newConfig.TrapReceiverSettings.IgnoreVersions, g.Version2c)
				ignorev2c = true
			}
		case "v3", "3":
			if !ignorev3 {
				newConfig.TrapReceiverSettings.IgnoreVersions = append(newConfig.TrapReceiverSettings.IgnoreVersions, g.Version3)
				ignorev3 = true
			}
		default:
			return fmt.Errorf("unsupported or invalid value (%s) for general:ignore_version", candidate)
		}
	}
	if len(newConfig.TrapReceiverSettings.IgnoreVersions) > 2 {
		return fmt.Errorf("all three SNMP versions are ignored -- there will be no traps to process")
	}
	return nil
}

func validateSnmpV3Args(params *trapListenerConfig) error {
	switch strings.ToLower(params.MsgFlags_str) {
	case "noauthnopriv":
		params.MsgFlags = g.NoAuthNoPriv
	case "authnopriv":
		params.MsgFlags = g.AuthNoPriv
	case "authpriv":
		params.MsgFlags = g.AuthPriv
	default:
		return fmt.Errorf("unsupported or invalid value (%s) for snmpv3:msg_flags", params.MsgFlags_str)
	}

	switch strings.ToLower(params.AuthProto_str) {
	case "noauth":
		params.AuthProto = g.NoAuth
	case "sha":
		params.AuthProto = g.SHA
	case "md5":
		params.AuthProto = g.MD5
	default:
		return fmt.Errorf("invalid value for snmpv3:auth_protocol: %s", params.AuthProto_str)
	}

	var err error
	var plaintext string
	plaintext, err = pluginMeta.GetSecret(params.AuthPassword)
	if err != nil {
		return fmt.Errorf("unable to decode secret for auth password: %s", params.AuthPassword)
	}
	params.AuthPassword = plaintext

	switch strings.ToLower(params.PrivacyProto_str) {
	case "nopriv":
		params.PrivacyProto = g.NoPriv
	case "aes":
		params.PrivacyProto = g.AES
	case "des":
		params.PrivacyProto = g.DES
	default:
		return fmt.Errorf("invalid value for snmpv3:privacy_protocol: %s", params.PrivacyProto_str)
	}

	plaintext, err = pluginMeta.GetSecret(params.PrivacyPassword)
	if err != nil {
		return fmt.Errorf("unable to decode secret for privacy password: %s", params.PrivacyPassword)
	}
	params.PrivacyPassword = plaintext

	if (params.MsgFlags&g.AuthPriv) == 1 && params.AuthProto < 2 {
		return fmt.Errorf("v3 config error: no auth protocol set when snmpv3:msg_flags specifies an Auth mode")
	}
	if params.MsgFlags == g.AuthPriv && params.PrivacyProto < 2 {
		return fmt.Errorf("v3 config error: no privacy protocol mode set when snmpv3:msg_flags specifies an AuthPriv mode")
	}

	return nil
}

func addIpSets(newConfig *trapexConfig) error {
	for _, stanza := range newConfig.IpSets_str {
		for ipsName, ips := range stanza {
			trapexLog.Debug().Str("ipset", ipsName).Msg("Loading IpSet")
			newConfig.IpSets[ipsName] = make(map[string]bool)
			for _, ip := range ips {
				if ipRe.MatchString(ip) {
					newConfig.IpSets[ipsName][ip] = true
					trapexLog.Debug().Str("ipset", ipsName).Str("ip", ip).Msg("Adding IP to IpSet")
				} else {
					return fmt.Errorf("invalid IP address (%s) in ipset: %s", ip, ipsName)
				}
			}
		}
	}
	return nil
}

func addFilters(newConfig *trapexConfig) error {
	var err error
	for i, _ := range newConfig.Filters {
		if err = addFilterObjs(&newConfig.Filters[i], newConfig.IpSets, i); err != nil {
			return err
		}
		if err = setAction(&newConfig.Filters[i], newConfig.General.PluginPath, i); err != nil {
			return err
		}
	}
	trapexLog.Info().Int("num_filters", len(newConfig.Filters)).Msg("Configured filter conditions")
	return nil
}

func addPluginErrorActions(newConfig *trapexConfig) error {
	var err error
	for i, _ := range newConfig.PluginErrorActions {
		if err = addFilterObjs(&newConfig.PluginErrorActions[i], newConfig.IpSets, i); err != nil {
			return err
		}
		if err = setAction(&newConfig.PluginErrorActions[i], newConfig.General.PluginPath, i); err != nil {
			return err
		}
	}
	trapexLog.Info().Int("num_filters", len(newConfig.PluginErrorActions)).Msg("Configured plugin error conditions")
	return nil
}

// addFilterObjs parses a "filter" line and sets
// the appropriate values in a corresponding trapexFilter struct.
//
func addFilterObjs(filter *trapexFilter, ipSets map[string]IpSet, lineNumber int) error {
	var err error

	// If we find something that is specifies a condition, then reset
	filter.matchAll = true
	if err = addSnmpFilterObj(filter, lineNumber); err != nil {
		return err
	}

	if err = addIpFilterObj(filter, filterBySrcIP, filter.SourceIp, ipSets, lineNumber); err != nil {
		return err
	}
	if err = addIpFilterObj(filter, filterByAgentAddr, filter.AgentAddress, ipSets, lineNumber); err != nil {
		return err
	}

	if err = addTrapTypeFilterObj(filter, filterByGenericType, filter.GenericType, lineNumber); err != nil {
		return err
	}
	if err = addTrapTypeFilterObj(filter, filterBySpecificType, filter.SpecificType, lineNumber); err != nil {
		return err
	}

	if err = addOidFilterObj(filter, filter.EnterpriseOid, lineNumber); err != nil {
		return err
	}
	return err
}

func setAction(filter *trapexFilter, pluginPathExpr string, lineNumber int) error {
	var err error

	switch filter.ActionName {
	case "break", "drop":
		filter.actionType = actionBreak
	case "nat":
		filter.actionType = actionNat
		if filter.ActionArg == "" {
			pluginDataMapping := args2map(filter.ActionArgs)
			natIp, ok := pluginDataMapping["natIp"]
			if ok {
				filter.ActionArg = natIp
			} else {
				return fmt.Errorf("missing NAT argument at line %v", lineNumber)
			}
		}
	default:
		filter.actionType = actionPlugin
		filter.plugin, err = pluginLoader.LoadActionPlugin(pluginPathExpr, filter.ActionName)
		if err != nil {
			return fmt.Errorf("unable to load plugin %s at line %v: %s", filter.ActionName, lineNumber, err)
		}
		pluginDataMapping := args2map(filter.ActionArgs)
		if err = filter.plugin.Configure(&trapexLog, pluginDataMapping); err != nil {
			return fmt.Errorf("unable to configure plugin %s at line %v: %s", filter.ActionName, lineNumber, err)
		}
	}
	return nil
}

func args2map(data []ActionArgType) map[string]string {
	pluginDataMapping := make(map[string]string)
	for _, pair := range data {
		if strings.Contains(pair.Key, "secret") ||
			strings.Contains(pair.Key, "password") {
			plaintext, err := pluginMeta.GetSecret(pair.Value)
			if err != nil {
				trapexLog.Warn().Err(err).Str("secret", pair.Key).Str("cipher_text", pair.Value).Msg("Unable to decode secret")
			} else {
				pair.Value = plaintext
			}
		}
		pluginDataMapping[pair.Key] = pair.Value
	}
	return pluginDataMapping
}

// addSnmpFilterObj adds a filter if necessary
// An empty arry of filters is interpreted to mean "All versions should match"
func addSnmpFilterObj(filter *trapexFilter, lineNumber int) error {
	for _, candidate := range filter.SnmpVersions {
		fObj := filterObj{filterItem: filterByVersion}
		switch strings.ToLower(candidate) {
		case "v1", "1":
			fObj.filterValue = g.Version1
		case "v2c", "2c", "2":
			fObj.filterValue = g.Version2c
		case "v3", "3":
			fObj.filterValue = g.Version3
		default:
			return fmt.Errorf("unsupported or invalid SNMP version (%s) on line %v", candidate, lineNumber)
		}
		fObj.filterType = parseTypeInt
		filter.matchAll = false
		filter.matchers = append(filter.matchers, fObj)
	}
	return nil
}

// addIpFilterObj returns a filter object for IP addresses, IP sets, CIDR
// If starts with a "ipset:"" it's an IP set
// If starts with a "/", it's a regex
func addIpFilterObj(filter *trapexFilter, source int, networkEntry string, ipSets map[string]IpSet, lineNumber int) error {
	var err error

	if networkEntry == "" {
		return nil
	}
	filter.matchAll = false

	fObj := filterObj{filterItem: source}
	if strings.HasPrefix(networkEntry, "ipset:") {
		fObj.filterType = parseTypeIPSet
		ipSetName := networkEntry[6:]
		if _, ok := ipSets[ipSetName]; ok {
			fObj.filterValue = ipSetName
		} else {
			return fmt.Errorf("invalid IP set name specified on for %v on line %v: %s", source, lineNumber, networkEntry)
		}
	} else if strings.HasPrefix(networkEntry, "/") {
		fObj.filterType = parseTypeRegex
		fObj.filterValue, err = regexp.Compile(networkEntry[1:])
		if err != nil {
			return fmt.Errorf("unable to compile regular expressions for IP for %v on line %v: %s: %s", source, lineNumber, networkEntry, err)
		}
	} else if strings.Contains(networkEntry, "/") {
		fObj.filterType = parseTypeCIDR
		fObj.filterValue, err = newNetwork(networkEntry)
		if err != nil {
			return fmt.Errorf("invalid IP/CIDR for %v at line %v: %s", source, lineNumber, networkEntry)
		}
	} else {
		fObj.filterType = parseTypeString
		fObj.filterValue = networkEntry
	}
	filter.matchers = append(filter.matchers, fObj)
	return nil
}

func addTrapTypeFilterObj(filter *trapexFilter, source int, trapTypeEntry int, lineNumber int) error {
	// -1 means to match everything
	if trapTypeEntry == -1 {
		return nil
	}
	filter.matchAll = false
	fObj := filterObj{filterItem: source, filterType: parseTypeInt, filterValue: trapTypeEntry}
	filter.matchers = append(filter.matchers, fObj)
	return nil
}

func addOidFilterObj(filter *trapexFilter, oid string, lineNumber int) error {
	var err error

	if oid == "" {
		return nil
	}
	filter.matchAll = false
	fObj := filterObj{filterItem: filterByOid, filterType: parseTypeRegex}
	fObj.filterValue, err = regexp.Compile(oid)
	if err != nil {
		return fmt.Errorf("unable to compile regular expression at line %v for OID: %s: %s", lineNumber, oid, err)
	}
	filter.matchers = append(filter.matchers, fObj)
	return nil
}

func closeTrapexHandles() {
	for _, f := range teConfig.Filters {
		if f.actionType == actionPlugin {
			f.plugin.Close()
		}
	}
}

func addReportingPlugins(newConfig *trapexConfig) error {
	var err error

	counters := pluginMeta.CreateMetricDefs()
	for i, config := range newConfig.Reporting {
		config.plugin, err = pluginLoader.LoadMetricPlugin(teConfig.General.PluginPath, config.PluginName)
		if err != nil {
			trapexLog.Fatal().Err(err).Str("plugin_name", config.PluginName).Msg("Unable to load metric reporting plugin")
			return err
		}
		pluginDataMapping := args2map(config.Args)
		if err = config.plugin.Configure(&trapexLog, pluginDataMapping, counters); err != nil {
			return fmt.Errorf("unable to configure plugin %s at line %v: %s", config.PluginName, i, err)
		}
	}
	trapexLog.Info().Int("num_reporters", len(newConfig.Reporting)).Msg("Configured metric reporting plugins")
	return nil
}
