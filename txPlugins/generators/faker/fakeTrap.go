// Copyright (c) 2022 Kells Kearney. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"io/ioutil"
	"path/filepath"

	"github.com/creasty/defaults"
	"gopkg.in/yaml.v2"
)

// loadTestConfig
// Load a YAML file with configuration, and create a new object
func loadTestConfig(config_file string, newConfig *testDataSpec) error {
	defaults.Set(newConfig)

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

func getTestConfig() (testDataSpec, error) {
	var newConfig testDataSpec
	err := loadTestConfig("testsnmpdata.yml", &newConfig)
	if err != nil {
		return newConfig, err
	}
	if err = validateSnmpV3Args(&newConfig.V3Params); err != nil {
		return newConfig, err
	}
	if err = processTestTraps(&newConfig); err != nil {
		return newConfig, err
	}

	return newConfig, nil
}

func processTestTraps(newConfig *testDataSpec) error {
	/*
		for _, stanza := range newConfig.IpSets_str {
			for ipsName, ips := range stanza {
				trapexLog.Debug().Str("ipset", ipsName).Msg("Loading IpSet")
				newConfig.IpSets[ipsName] = make(map[string]bool)
				for _, ip := range ips {
					if ipRe.MatchString(ip) {
						newConfig.IpSets[ipsName][ip] = true
						trapexLog.Debug().Str("ipset", ipsName).Str("ip", ip).Msg("Adding IP to IpSet")
					} else {
						return fmt.Errorf("Invalid IP address (%s) in ipset: %s", ip, ipsName)
					}
				}
			}
		}
	*/
	return nil
}
