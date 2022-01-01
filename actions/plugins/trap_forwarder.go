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
	g "github.com/gosnmp/gosnmp"

	"github.com/damienstuart/trapex/actions"
	"github.com/rs/zerolog"
)

type trapForwarder struct {
	destination *g.GoSNMP
	trapex_log  zerolog.Logger
}

const plugin_name = "trap forwarder"

func (p trapForwarder) Configure(logger zerolog.Logger, actionArg string, pluginConfig *plugin_interface.PluginsConfig) error {
	p.trapex_log = logger

	//logger.Info().Str("plugin", plugin_name).Msg("Initialization of plugin")

	/*
	   func (a *trapForwarder) initAction(dest string) error {
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
	           logger.Info().Str("target", s[0]).Str("port", s[1]).Msg("Added trap destination")

	*/
	return nil
}

func (a trapForwarder) ProcessTrap(trap *plugin_interface.Trap) error {
	_, err := a.destination.SendTrap(trap.Data)
	return err

	//logger.Info().Str("plugin", plugin_name).Msg("Processing trap")
	return nil
}

func (p trapForwarder) SigUsr1() error {
	return nil
}

func (p trapForwarder) SigUsr2() error {
	return nil
}

func (a trapForwarder) close() {
	a.destination.Conn.Close()
}

// Exported symbol which supports filter.go's FilterAction type
var FilterPlugin trapForwarder
