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
	//zlog "github.com/rs/zerolog/log"
)

// sgTrap holds a pointer to a trap and the source IP of
// the incoming trap.
//
type sgTrap struct {
	trapNumber uint64
	data       g.SnmpTrap
	trapVer    g.SnmpVersion
	srcIP      net.IP
	translated bool
	dropped    bool
}

var trapRateTracker = newTrapRateTracker()
var logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

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
		logger.Fatal().Err(err).Msg("Unable to load configuration")
		os.Exit(1)
	}

	initSigHandlers()
	go exposeMetrics()
	var exporter = fmt.Sprintf("http://%s:%s/%s\n",
		teConfig.General.PrometheusIp, teConfig.General.PrometheusPort, teConfig.General.PrometheusEndpoint)
	logger.Info().Str("endpoint", exporter).Msg("Prometheus metrics exported")

	stats.StartTime = time.Now()

	go trapRateTracker.start()

	tl := g.NewTrapListener()

	tl.OnNewTrap = trapHandler
	tl.Params = g.Default
	tl.Params.Community = ""

	// Uncomment for debugging gosnmp
	if teConfig.Logging.Level == "debug" {
		logger.Info().Msg("gosnmp debug mode enabled")
		tl.Params.Logger = g.NewLogger(log.New(os.Stdout, "", 0))
	}

	// SNMP v3 stuff
	tl.Params.SecurityModel = g.UserSecurityModel
	tl.Params.MsgFlags = teConfig.V3Params.msgFlags
	tl.Params.Version = g.Version3
	tl.Params.SecurityParameters = &g.UsmSecurityParameters{
		UserName:                 teConfig.V3Params.Username,
		AuthenticationProtocol:   teConfig.V3Params.authProto,
		AuthenticationPassphrase: teConfig.V3Params.AuthPassword,
		PrivacyProtocol:          teConfig.V3Params.privacyProto,
		PrivacyPassphrase:        teConfig.V3Params.PrivacyPassword,
	}

	listenAddr := fmt.Sprintf("%s:%s", teConfig.General.ListenAddr, teConfig.General.ListenPort)
	logger.Info().Str("listen_address", listenAddr).Msg("Start trapex listener")
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
	trap := sgTrap{
		data: g.SnmpTrap{
			Variables:    p.Variables,
			Enterprise:   p.Enterprise,
			AgentAddress: p.AgentAddress,
			GenericTrap:  p.GenericTrap,
			SpecificTrap: p.SpecificTrap,
			Timestamp:    p.Timestamp,
		},
		srcIP:   addr.IP,
		trapVer: p.Version,
	}

	// Translate to v1 if needed
	/*
	 */
	if p.Version > g.Version1 {
		err := translateToV1(&trap)
		if err != nil {
			var info string
			info = makeTrapLogEntry(&trap)
			logger.Warn().Err(err).Str("trap", info).Msg("Error translating to v1")
		}
	}

	if teConfig.Logging.Level == "debug" {
		var info string
		info = makeTrapLogEntry(&trap)
		logger.Debug().Str("trap", info).Msg("Raw trap info")
	}

	processTrap(&trap)
}

// processTrap is the entry point to code that checks the incoming trap
// against the filter list and processes the trap accordingly.
//
func processTrap(sgt *sgTrap) {
	for _, f := range teConfig.filters {
		// If this trap is tagged to drop, then continue.
		if sgt.dropped {
			continue
		}
		// If matchAll is true, just process the action.
		if f.matchAll == true {
			// We don't expect to see this here (set a wide open filter for
			// drop).... (but...)
			if f.actionType == actionBreak {
				sgt.dropped = true
				stats.DroppedTraps++
				trapsDropped.Inc()
				continue
			}
			f.processAction(sgt)
			if f.actionType == actionForwardBreak || f.actionType == actionLogBreak || f.actionType == actionCsvBreak {
				sgt.dropped = true
				stats.DroppedTraps++
				trapsDropped.Inc()
				continue
			}
		} else {
			// Determine if this trap matches this filter
			if f.isFilterMatch(sgt) {
				if f.actionType == actionBreak {
					sgt.dropped = true
					stats.DroppedTraps++
					trapsDropped.Inc()
					continue
				}
				f.processAction(sgt)
				if f.actionType == actionForwardBreak || f.actionType == actionLogBreak || f.actionType == actionCsvBreak {
					sgt.dropped = true
					stats.DroppedTraps++
					trapsDropped.Inc()
					continue
				}
			}
		}
	}
}
