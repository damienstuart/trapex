// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

/*
This plugin performs no useful action, but can be used for unit testing or performance
testing purposes.
*/

import (
	"github.com/damienstuart/trapex/actions"
	"github.com/rs/zerolog"
)

type noopFilter struct {
	trapex_log zerolog.Logger
}

const plugin_name = "no op"

func (p noopFilter) Configure(logger zerolog.Logger, actionArg string, pluginConfig *plugin_data.PluginsConfig) error {
	logger.Info().Str("plugin", plugin_name).Str("test1", pluginConfig.Noop.Test1).Str("test2", pluginConfig.Noop.Test2).Msg("Initialization of plugin")
	p.trapex_log = logger
	return nil
}

func (p noopFilter) ProcessTrap(trap *plugin_data.Trap) error {
	//logger.Info().Str("plugin", plugin_name).Msg("Noop processing trap")
	return nil
}

func (p noopFilter) Close() error {
	return nil
}

func (p noopFilter) SigUsr1() error {
	return nil
}

func (p noopFilter) SigUsr2() error {
	return nil
}

// Exported symbol which supports filter.go's FilterAction type
var FilterPlugin noopFilter
