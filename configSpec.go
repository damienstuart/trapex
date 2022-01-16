// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"github.com/creasty/defaults"
	pluginLoader "github.com/damienstuart/trapex/txPlugins/interfaces"
	g "github.com/gosnmp/gosnmp"
)

type trapListenerConfig struct {
	Hostname string `yaml:"hostname"`

	ListenAddr string `default:"0.0.0.0" yaml:"listen_address"`
	ListenPort string `default:"162" yaml:"listen_port"`

	GoSnmpDebug        bool   `default:"false" yaml:"gosnmp_debug"`
	GoSnmpDebugLogName string `default:"" yaml:"gosnmp_debug_logfile_name`

	IgnoreVersions_str []string        `default:"[]" yaml:"ignore_versions"`
	IgnoreVersions     []g.SnmpVersion `default:"[]"`

	Community string `default:"" yaml:"snmp_community"`

	// SNMP v3 settings
	MsgFlags_str     string               `default:"NoAuthNoPriv" yaml:"msg_flags"`
	MsgFlags         g.SnmpV3MsgFlags     `default:"g.NoAuthNoPriv"`
	Username         string               `default:"XXv3Username" yaml:"username"`
	AuthProto_str    string               `default:"NoAuth" yaml:"auth_protocol"`
	AuthProto        g.SnmpV3AuthProtocol `default:"g.NoAuth"`
	AuthPassword     string               `default:"XXv3authPass" yaml:"auth_password"`
	PrivacyProto_str string               `default:"NoPriv" yaml:"privacy_protocol"`
	PrivacyProto     g.SnmpV3PrivProtocol `default:"g.NoPriv"`
	PrivacyPassword  string               `default:"XXv3Pass" yaml:"privacy_password"`
}

type IpSet map[string]bool

// filterObj represents one of the filterable items in a filter line from
// the config file (i.e. Src IP, AgentAddress, GenericType, SpecificType,
// and Enterprise OID).
//
type filterObj struct {
	filterItem  int
	filterType  int
	filterValue interface{} // string, *regex.Regexp, *network, int
}

// Get in a set of action arg pairs, convert to a map to pass into plugins
type ActionArgType struct {
	Key   string `default:"" yaml:"key"`
	Value string `default:"" yaml:"value"`
}

// trapexFilter holds the filter data and action for a specfic
// filter line from the config file.
type trapexFilter struct {
	// SnmpVersions - an empty array will indicate ALL versions
	SnmpVersions []string `default:"[]" yaml:"snmp_versions"`
	SourceIp     string   `default:"" yaml:"source_ip"`
	AgentAddress string   `default:"" yaml:"agent_address"`

	// GenericType can have values from 0 - 6: -1 indicates all types
	GenericType int `default:"-1" yaml:"snmp_generic_type"`
	// SpecificType can have values from 0 - n: -1 indicates all types
	SpecificType int `default:"-1" yaml:"snmp_specific_type"`

	EnterpriseOid string          `default:"" yaml:"enterprise_oid"`
	ActionName    string          `default:"" yaml:"action"`
	ActionArg     string          `default:"" yaml:"action_arg"`
	BreakAfter    bool            `default:"false" yaml:"break_after"`
	ActionArgs    []ActionArgType `default:"[]" yaml:"action_args"`

	// Compiled definition of above
	matchAll   bool
	matchers   []filterObj
	actionType int
	plugin     pluginLoader.ActionPlugin
}

// UnmarshalYAML is what enables the setter to work for the trapexFilter
func (s *trapexFilter) UnmarshalYAML(unmarshal func(interface{}) error) error {
	defaults.Set(s)

	type plain trapexFilter
	if err := unmarshal((*plain)(s)); err != nil {
		return err
	}
	return nil
}

type MetricConfig struct {
	PluginName string          `default:"" yaml:"plugin"`
	Args       []ActionArgType `default:"[]" yaml:"args"`
	plugin     pluginLoader.MetricPlugin
}

type trapexConfig struct {
	teConfigured bool

	General struct {
		Bongo      string `default:"bongo" yaml:"bongo"`
		PluginPath string `default:"txPlugins" yaml:"plugin_path"`
	}

	Reporting []MetricConfig `default:"[]" yaml:"metric_reporting"`

	Logging struct {
		Level         string `default:"debug" yaml:"level"`
		LogMaxSize    int    `default:"1024" yaml:"log_size_max"`
		LogMaxBackups int    `default:"7" yaml:"log_backups_max"`
		LogMaxAge     int    `yaml:"log_age_max"`
		LogCompress   bool   `default:"false" yaml:"compress_rotated_logs"`
	}

	TrapReceiverSettings trapListenerConfig `yaml:"trap_receiver_settings"`

	IpSets_str []map[string][]string `default:"{}" yaml:"ip_sets"`
	IpSets     map[string]IpSet      `default:"{}"`

	Filters []trapexFilter `default:"[]" yaml:"filters"`

	// Bad things happen to good plugins. How do you want to handle exceptions?
	PluginErrorActions []trapexFilter `default:"[]" yaml:"plugin_error_actions"`
}
