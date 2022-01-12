// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

/*
 * This plugin streams traps to AWS Kinesis
 */

import (
	"fmt"
	pluginMeta "github.com/damienstuart/trapex/txPlugins"
	"github.com/rs/zerolog"
)

type kinesisConfig struct {
	trapexLog *zerolog.Logger
}

const pluginName = "AWS Kinesis"

func validateArguments(actionArgs map[string]string) error {
	validArgs := map[string]bool{"traphost": true, "port": true, "snmp_version": true, "community": true}

	for key, _ := range actionArgs {
		if _, ok := validArgs[key]; !ok {
			return fmt.Errorf("Unrecognized option to %s plugin: %s", pluginName, key)
		}
	}

	return nil
}

func (p *kinesisConfig) Configure(trapexLog *zerolog.Logger, actionArgs map[string]string) error {
	trapexLog.Info().Str("plugin", pluginName).Msg("Initialization of plugin")
	p.trapexLog = trapexLog
	if err := validateArguments(actionArgs); err != nil {
		return err
	}

	return nil
}

func (p kinesisConfig) ProcessTrap(trap *pluginMeta.Trap) error {
	p.trapexLog.Info().Str("plugin", pluginName).Msg("Noop processing trap")
	return nil
}

func (p kinesisConfig) Close() error {
	return nil
}

func (p kinesisConfig) SigUsr1() error {
	return nil
}

func (p kinesisConfig) SigUsr2() error {
	return nil
}

// Exported symbol which supports filter.go's FilterAction type
var ActionPlugin kinesisConfig
