// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"fmt"
	"testing"
)

func TestPluginInterfacess(t *testing.T) {
	var err error
	plugins := []string{"noop", "logfile", "forward", "clickhouse"}

	for _, plugin_name := range plugins {
		fmt.Printf("Verifying plugin interface: %s\n", plugin_name)
		_, err = loadActionPlugin("txPlugins/actions/plugins/%s.so", plugin_name)

		if err != nil {
			t.Errorf("Unable to load plugin %s", plugin_name)
		}
	}
	/*
	                   if err == nil {
	                           filter.action.Configure(trapexLog, actionArg, &newConfig.FilterPluginsConfig)
	                   }

	   	var testConfig trapexConfig
	   	loadConfig("tests/config/general.yml", &testConfig)
	*/

}
