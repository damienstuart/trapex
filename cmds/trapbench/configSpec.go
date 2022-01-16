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

type GeneratorType struct {
	PluginName     string          `default:"replay" yaml:"plugin"`
	PluginPathExpr string          `default:"txPlugins/generators/%s.so" yaml:"plugin_path"`
	Stream         bool            `default:"false" yaml:"stream"`
	Count          int             `default:"100" yaml:"count"`
	Args           []ReplayArgType `default:"[]" yaml:"args"`
	plugin         pluginLoader.GeneratorPlugin
}

type DestinationType struct {
	Name           string
	PluginPathExpr string          `default:"txPlugins/actions/%s.so" yaml:"plugin_path"`
	PluginName     string          `yaml:"plugin"`
	ReplayArgs     []ReplayArgType `default:"[]" yaml:"replay_args"`
	plugin         pluginLoader.ActionPlugin
}

type replayConfig struct {
	General struct {
		Hostname string `yaml:"hostname"`
		LogLevel string `default:"debug" yaml:"log_level"`
	}

	Generator GeneratorType `default:"{}" yaml:"generator"`

	Destination DestinationType `default:"{}" yaml:"destination"`
}
