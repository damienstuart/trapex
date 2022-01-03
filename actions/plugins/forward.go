// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

/*
This plugin sends SNMP traps to a new destination
*/

import (
	"fmt"
	"strconv"
	"time"

	plugin_data "github.com/damienstuart/trapex/actions"
	g "github.com/gosnmp/gosnmp"

	"github.com/rs/zerolog"
)

type trapForwarder struct {
	destination *g.GoSNMP
	trapex_log  *zerolog.Logger
}

const pluginName = "trap forwarder"

func validateArguments(actionArgs map[string]string) error {
	validArgs := map[string]bool{"traphost": true, "port": true}

	for key, _ := range actionArgs {
		if _, ok := validArgs[key]; !ok {
			return fmt.Errorf("Unrecognized option to %s plugin: %s", pluginName, key)
		}
	}
	return nil
}

func (a *trapForwarder) Configure(trapexLog *zerolog.Logger, actionArgs map[string]string) error {
	a.trapex_log = trapexLog

	a.trapex_log.Info().Str("plugin", pluginName).Msg("Initialization of plugin")

	if err := validateArguments(actionArgs); err != nil {
		return err
	}

	hostname := actionArgs["traphost"]
	port_str := actionArgs["port"]
	port, err := strconv.Atoi(port_str)
	if err != nil {
		panic("Invalid destination port: " + port_str)
	}
	a.destination = &g.GoSNMP{
		Target:             hostname,
		Port:               uint16(port),
		Transport:          "udp",
		Community:          "",
		Version:            g.Version1,
		Timeout:            time.Duration(2) * time.Second,
		Retries:            3,
		ExponentialTimeout: true,
		MaxOids:            g.MaxOids,
	}
	err = a.destination.Connect()
	if err != nil {
		return err
	}
	a.trapex_log.Info().Str("target", hostname).Str("port", port_str).Msg("Added trap destination")

	return nil
}

func (a trapForwarder) ProcessTrap(trap *plugin_data.Trap) error {
	a.trapex_log.Info().Str("plugin", pluginName).Msg("Processing trap")
	_, err := a.destination.SendTrap(trap.Data)
	return err
}

func (p trapForwarder) SigUsr1() error {
	return nil
}

func (p trapForwarder) SigUsr2() error {
	return nil
}

func (a trapForwarder) Close() error {
	a.destination.Conn.Close()
	return nil
}

// Exported symbol which supports filter.go's FilterAction type
var FilterPlugin trapForwarder
