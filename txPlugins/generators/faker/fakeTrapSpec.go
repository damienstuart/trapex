// Copyright (c) 2022 Kells Kearney. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

type testTrapVariableDef struct {
	oid      string `default:"" yaml:"oid"`
	trapType string `default:"octetstring" yaml:"type"`
	value    string `default:"" yaml:"value"`
}

type testTrapDef struct {
	enterprise   string `default:"" yaml:"enterprise"`
	snmpVersion  string `default:"1" yaml:"snmp_version"`
	genericTrap  int    `default:"0" yaml:"generic_trap"`
	specificTrap int    `default:"0" yaml:"specific_trap"`

	variables []testTrapVariableDef `default:"[]" yaml:"variables"`
}

type testDataSpec struct {
	General struct {
		agentAddress string `yaml:"agent_address"`
		sourceIp     string `yaml:"source_ip"`
	}

	V3Params v3Params `yaml:"snmpv3"`

	traps []map[string][]testTrapDef `default:"{}" yaml:"traps"`
}
