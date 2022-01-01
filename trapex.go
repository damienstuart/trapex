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
	"time"

	g "github.com/gosnmp/gosnmp"

	"github.com/rs/zerolog"

	"github.com/damienstuart/trapex/actions"
)

var trapRateTracker = newTrapRateTracker()
var trapex_logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	flag.Usage = func() {
		fmt.Printf("Usage:\n")
		fmt.Printf("   %s\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	// Process the command-line and get the configuration.
	//
	processCommandLine()

	if err := getConfig(); err != nil {
		trapex_logger.Fatal().Err(err).Msg("Unable to load configuration")
		os.Exit(1)
	}

	initSigHandlers()
	go exposeMetrics()
	var exporter = fmt.Sprintf("http://%s:%s/%s\n",
		teConfig.General.PrometheusIp, teConfig.General.PrometheusPort, teConfig.General.PrometheusEndpoint)
	trapex_logger.Info().Str("endpoint", exporter).Msg("Prometheus metrics exported")

	stats.StartTime = time.Now()

	go trapRateTracker.start()

	tl := g.NewTrapListener()

	tl.OnNewTrap = trapHandler
	tl.Params = g.Default
	tl.Params.Community = ""

	// Uncomment for debugging gosnmp
	if teConfig.Logging.Level == "debug" {
		trapex_logger.Info().Msg("gosnmp debug mode enabled")
		tl.Params.Logger = g.NewLogger(log.New(os.Stdout, "", 0))
	}

	// SNMP v3 stuff
	tl.Params.SecurityModel = g.UserSecurityModel
	tl.Params.MsgFlags = teConfig.V3Params.MsgFlags
	tl.Params.Version = g.Version3
	tl.Params.SecurityParameters = &g.UsmSecurityParameters{
		UserName:                 teConfig.V3Params.Username,
		AuthenticationProtocol:   teConfig.V3Params.AuthProto,
		AuthenticationPassphrase: teConfig.V3Params.AuthPassword,
		PrivacyProtocol:          teConfig.V3Params.PrivacyProto,
		PrivacyPassphrase:        teConfig.V3Params.PrivacyPassword,
	}

	listenAddr := fmt.Sprintf("%s:%s", teConfig.General.ListenAddr, teConfig.General.ListenPort)
	trapex_logger.Info().Str("listen_address", listenAddr).Msg("Start trapex listener")
	err := tl.Listen(listenAddr)
	if err != nil {
		log.Panicf("error in listen on %s: %s", listenAddr, err)
	}
}

// trapHandler is the callback for handling traps received by the listener.
//
func trapHandler(p *g.SnmpPacket, addr *net.UDPAddr) {
	// Count every trap received
	stats.TrapCount++
	trapsCount.Inc()

	// First thing to do is check for ignored versions
	if isIgnoredVersion(p.Version) {
		stats.IgnoredTraps++
		trapsIgnored.Inc()
		return
	}

	// Also keep track of traps we handle
	stats.HandledTraps++
	trapsHandled.Inc()

	// Make the trap
	trap := plugin_interface.Trap{
		Data: g.SnmpTrap{
			Variables:    p.Variables,
			Enterprise:   p.Enterprise,
			AgentAddress: p.AgentAddress,
			GenericTrap:  p.GenericTrap,
			SpecificTrap: p.SpecificTrap,
			Timestamp:    p.Timestamp,
		},
		SrcIP:   addr.IP,
		TrapVer: p.Version,
	}

	// Translate to v1 if needed
	if p.Version > g.Version1 {
		err := translateToV1(&trap)
		if err != nil {
			var info string
			info = makeTrapLogEntry(&trap)
			trapex_logger.Warn().Err(err).Str("trap", info).Msg("Error translating to v1")
		}
	}

	if teConfig.Logging.Level == "debug" {
		var info string
		info = makeTrapLogEntry(&trap)
		trapex_logger.Debug().Str("trap", info).Msg("Raw trap info")
	}

	processTrap(&trap)
}

// processTrap is the entry point to code that checks the incoming trap
// against the filter list and processes the trap accordingly.
//
func processTrap(sgt *plugin_interface.Trap) {
	for _, f := range teConfig.filters {
		// If this trap is tagged to drop, then continue.
		if sgt.Dropped {
			continue
		}
		// If matchAll is true, just process the action.
		if f.MatchAll == true {
			// We don't expect to see this here (set a wide open filter for
			// drop).... (but...)
			if f.ActionType == actionBreak {
				sgt.Dropped = true
				stats.DroppedTraps++
				trapsDropped.Inc()
				continue
			}
			f.processAction(sgt)
			if f.ActionType == actionForwardBreak || f.ActionType == actionLogBreak || f.ActionType == actionCsvBreak {
				sgt.Dropped = true
				stats.DroppedTraps++
				trapsDropped.Inc()
				continue
			}
		} else {
			// Determine if this trap matches this filter
			if f.isFilterMatch(sgt) {
				if f.ActionType == actionBreak {
					sgt.Dropped = true
					stats.DroppedTraps++
					trapsDropped.Inc()
					continue
				}
				f.processAction(sgt)
				if f.ActionType == actionForwardBreak || f.ActionType == actionLogBreak || f.ActionType == actionCsvBreak {
					sgt.Dropped = true
					stats.DroppedTraps++
					trapsDropped.Inc()
					continue
				}
			}
		}
	}
}
