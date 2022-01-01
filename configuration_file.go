// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"github.com/damienstuart/trapex/actions"

	g "github.com/gosnmp/gosnmp"
)

type v3Params struct {
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

type TrapexConfig struct {
	Configured bool
	RunLogFile string
	ConfigFile string

	General struct {
		Hostname   string `yaml:"hostname"`
		ListenAddr string `default:"0.0.0.0" yaml:"listen_address"`
		ListenPort string `default:"162" yaml:"listen_port"`

		IgnoreVersions_str []string        `default:"[]" yaml:"ignore_versions"`
		IgnoreVersions     []g.SnmpVersion `default:"[]"`

		PrometheusIp       string `default:"0.0.0.0" yaml:"prometheus_ip"`
		PrometheusPort     string `default:"80" yaml:"prometheus_port"`
		PrometheusEndpoint string `default:"metrics" yaml:"prometheus_endpoint"`
	}

	Logging struct {
		Level         string `default:"debug" yaml:"level"`
		LogMaxSize    int    `default:"1024" yaml:"log_size_max"`
		LogMaxBackups int    `default:"7" yaml:"log_backups_max"`
		LogMaxAge     int    `yaml:"log_age_max"`
		LogCompress   bool   `default:"false" yaml:"compress_rotated_logs"`
	}

	V3Params v3Params `yaml:"snmpv3"`

	FilterPluginsConfig plugin_interface.PluginsConfig `yaml:"filter_plugins_config"`

	IpSets_str []map[string][]string `default:"{}" yaml:"ip_sets"`
	IpSets     map[string]IpSet      `default:"{}"`

	Filters_str []string `default:"[]" yaml:"filters"`
	filters     []trapexFilter
}
