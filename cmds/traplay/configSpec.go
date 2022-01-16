// Copyright (c) 2022 Kells Kearney. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	pluginLoader "github.com/damienstuart/trapex/txPlugins/interfaces"
)

type ReplayArgType struct {
	Key   string `default:"" yaml:"key"`
	Value string `default:"" yaml:"value"`
}

type DestinationType struct {
	Name       string
	Plugin     string
	ReplayArgs []ReplayArgType `default:"[]" yaml:"replay_args"`
	plugin     pluginLoader.ActionPlugin
}

type replayConfig struct {
	General struct {
		Hostname       string `yaml:"hostname"`
		PluginPathExpr string `default:"txPlugins/actions/%s.so" yaml:"plugin_path"`
		LogLevel       string `default:"debug" yaml:"log_level"`
	}

	Destinations []DestinationType `default:"{}" yaml:"destinations"`
}
