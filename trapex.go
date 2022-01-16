// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	g "github.com/gosnmp/gosnmp"

	"github.com/rs/zerolog"

	pluginMeta "github.com/damienstuart/trapex/txPlugins"
	pluginLoader "github.com/damienstuart/trapex/txPlugins/interfaces"
)

var trapexLog = zerolog.New(os.Stdout).With().Timestamp().Logger()

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	flag.Usage = func() {
		fmt.Printf("Usage:\n")
		fmt.Printf("   %s\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	// Process the command-line and get the configuration.
	processCommandLine()

	if err := getConfig(); err != nil {
		trapexLog.Fatal().Err(err).Msg("Unable to load configuration")
		os.Exit(1)
	}

	initSigHandlers()
	startTrapListener()
}

// startTrapListener configures the SNMP service information and starts actively
// processing traps via callback function (trapHandler)
// The listener will be able to receive SNMP v1/v2c traps, and if SNMP v3 information
// is configured correctly, SNMP v3 traps.
//
func startTrapListener() {
	trapListener := g.NewTrapListener()

	// Callback: trapHandler
	trapListener.OnNewTrap = trapHandler

	if teConfig.TrapReceiverSettings.GoSnmpDebug {
		trapexLog.Info().Msg("gosnmp debug mode enabled")
		if teConfig.TrapReceiverSettings.GoSnmpDebugLogName == "" {
			trapListener.Params.Logger = g.NewLogger(log.New(os.Stdout, "", 0))
		} else {
			fd, err := os.Open(teConfig.TrapReceiverSettings.GoSnmpDebugLogName)
			if err != nil {
				trapexLog.Fatal().Err(err).Str("filename", teConfig.TrapReceiverSettings.GoSnmpDebugLogName).Msg("Unable to open up debug log")
				os.Exit(1)
			}
			trapListener.Params.Logger = g.NewLogger(log.New(fd, "", 0))
		}
	}

	trapListener.Params = g.Default
	trapListener.Params.Community = teConfig.TrapReceiverSettings.Community

	// SNMP v3 stuff
	trapListener.Params.SecurityModel = g.UserSecurityModel
	trapListener.Params.MsgFlags = teConfig.TrapReceiverSettings.MsgFlags
	trapListener.Params.Version = g.Version3
	trapListener.Params.SecurityParameters = &g.UsmSecurityParameters{
		UserName:                 teConfig.TrapReceiverSettings.Username,
		AuthenticationProtocol:   teConfig.TrapReceiverSettings.AuthProto,
		AuthenticationPassphrase: teConfig.TrapReceiverSettings.AuthPassword,
		PrivacyProtocol:          teConfig.TrapReceiverSettings.PrivacyProto,
		PrivacyPassphrase:        teConfig.TrapReceiverSettings.PrivacyPassword,
	}

	listenAddr := fmt.Sprintf("%s:%s", teConfig.TrapReceiverSettings.ListenAddr, teConfig.TrapReceiverSettings.ListenPort)
	trapexLog.Info().Str("listen_address", listenAddr).Msg("Start trapex listener")
	err := trapListener.Listen(listenAddr)
	if err != nil {
		log.Panicf("error in listen on %s: %s", listenAddr, err)
	}
}

// counterInc increment the specified counter (reference to counter defintions)
//
func counterInc(counter int) {
	for _, reporter := range teConfig.Reporting {
		reporter.plugin.(pluginLoader.MetricPlugin).Inc(counter)
	}
}

// Keep track of total number of traps received
var totalTraps int

// trapHandler is the callback for handling traps received by the listener.
//
func trapHandler(p *g.SnmpPacket, addr *net.UDPAddr) {
	// Count every trap received
	counterInc(TrapCount)
	totalTraps++

	switch p.Version {
	case g.Version1:
		counterInc(V1Traps)
	case g.Version2c:
		counterInc(V2cTraps)
	case g.Version3:
		counterInc(V3Traps)
	}

	// First thing to do is check for ignored versions
	if isIgnoredVersion(p.Version) {
		counterInc(IgnoredTraps)
		return
	}

	// Also keep track of traps we handle
	counterInc(HandledTraps)

	// Make the trap
	trap := pluginMeta.Trap{
		Data: g.SnmpTrap{
			Variables:    p.Variables,
			Enterprise:   p.Enterprise,
			AgentAddress: p.AgentAddress,
			GenericTrap:  p.GenericTrap,
			SpecificTrap: p.SpecificTrap,
			Timestamp:    p.Timestamp,
		},
		SrcIP:       addr.IP,
		SnmpVersion: p.Version,
		Hostname:    teConfig.TrapReceiverSettings.Hostname,
		TrapNumber:  uint(totalTraps),
	}

	if teConfig.Logging.Level == "debug" {
		var info string
		info = makeTrapLogEntry(&trap)
		trapexLog.Debug().Str("trap", info).Msg("Raw trap info")
	}

	processTrap(&trap)
}

// processTrap is the entry point to code that checks the incoming trap
// against the filter list and processes the trap accordingly.
//
func processTrap(trap *pluginMeta.Trap) {
	for _, filterDef := range teConfig.Filters {
		if trap.Dropped {
			continue
		}

		if filterDef.matchAll || filterDef.isFilterMatch(trap) {
			if filterDef.actionType == actionBreak {
				trap.Dropped = true
				counterInc(DroppedTraps)
				continue
			}

			err := filterDef.processAction(trap)
			if err != nil {
				for _, pluginErrorFilters := range teConfig.PluginErrorActions {
					go pluginErrorFilters.processAction(trap)
				}
			}

			if filterDef.BreakAfter {
				trap.Dropped = true
				counterInc(DroppedTraps)
				continue
			}
		}
	}
}
