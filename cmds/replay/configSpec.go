// Copyright (c) 2022 Kells Kearney. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main


type ReplayArgType struct {
	Key   string `default:"" yaml:"key"`
	Value string `default:"" yaml:"value"`
}

type DestinationType struct {
   Name string
   Plugin string
ReplayArgs []ReplayArgType `default:"[]" yaml:"replay_args"`
}
   

type replayConfig struct {

	General struct {
		Hostname   string `yaml:"hostname"`

		GoSnmpDebug bool `default:"false" yaml:"gosnmp_debug"`

		PluginPathExpr string `default:"txPlugins/filter_actions/%s.so" yaml:"plugin_path"`
		LogLevel         string `default:"debug" yaml:"log_level"`
	}

	Destinations map[string]DestinationType `default:"{}" yaml:"destinations"`
}

