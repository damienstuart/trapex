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
	"strconv"
	"strings"
	"time"

	plugin_data "github.com/damienstuart/trapex/actions"
	g "github.com/gosnmp/gosnmp"

	"github.com/rs/zerolog"
)

type trapForwarder struct {
	destination *g.GoSNMP
	trapex_log  *zerolog.Logger
}

const plugin_name = "trap forwarder"

func (a trapForwarder) Configure(trapexLog *zerolog.Logger, actionArg string, pluginConfig *plugin_data.PluginsConfig) error {
	a.trapex_log = trapexLog

	a.trapex_log.Info().Str("plugin", plugin_name).Msg("Initialization of plugin")

	dest := actionArg
	s := strings.Split(dest, ":")
	port, err := strconv.Atoi(s[1])
	if err != nil {
		panic("Invalid destination port: " + s[1])
	}
	a.destination = &g.GoSNMP{
		Target:             s[0],
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
		return (err)
	}
	a.trapex_log.Info().Str("target", s[0]).Str("port", s[1]).Msg("Added trap destination")

	return nil
}

func (a trapForwarder) ProcessTrap(trap *plugin_data.Trap) error {
	a.trapex_log.Info().Str("plugin", plugin_name).Msg("Processing trap")
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
