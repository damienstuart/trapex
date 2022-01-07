// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"fmt"
	"os"
	"time"

	pluginLoader "github.com/damienstuart/trapex/txPlugins/interfaces"
)

// On SIGHUP we reload the configuration.
//
func handleSIGHUP(sigCh chan os.Signal) {
	for {
		select {
		case <-sigCh:
			fmt.Printf("Got SIGHUP - Reloading configuration.\n")
			if err := getConfig(); err != nil {
				trapexLog.Info().Err(err).Msg("Error parsing configuration\nConfiguration was not changed")
			}
		}
	}
}

// Use SIGUSR1 to dump current stats to STDOUT.
//
func handleSIGUSR1(sigCh chan os.Signal) {
	for {
		select {
		case <-sigCh:
			// Compute uptime
			stats.UptimeInt = time.Now().Unix() - stats.StartTime.Unix()
			trapexLog.Info().
				Str("uptime_str", secondsToDuration(uint(stats.UptimeInt))).
				Uint("uptime", uint(stats.UptimeInt)).
				Uint("traps_received", stats.TrapCount).
				Uint("traps_ignored", stats.IgnoredTraps).
				Uint("traps_processed", stats.HandledTraps).
				Uint("traps_dropped", stats.DroppedTraps).
				Uint("traps_tranlated_from_v2c", stats.TranslatedFromV2c).
				Uint("traps_tranlated_from_v3", stats.TranslatedFromV3).
				Uint("trap_rate_1min", trapRateTracker.getRate(1)).
				Uint("trap_rate_5min", trapRateTracker.getRate(5)).
				Uint("trap_rate_15min", trapRateTracker.getRate(15)).
				Uint("trap_rate_1hour", trapRateTracker.getRate(60)).
				Uint("trap_rate_4hour", trapRateTracker.getRate(240)).
				Uint("trap_rate_4hour", trapRateTracker.getRate(480)).
				Uint("trap_rate_1day", trapRateTracker.getRate(1440)).
				Uint("trap_rate_all", trapRateTracker.getRate(0)).
				Msg("Got SIGUSR1 for trapex stats")
		}
	}
}

// Use SIGUSR2 to force a rotation of log files.
//
func handleSIGUSR2(sigCh chan os.Signal) {
	for {
		select {
		case <-sigCh:
			trapexLog.Info().Msg("Got SIGUSR2")
			for _, f := range teConfig.Filters {
				if f.actionType == actionPlugin {
					f.plugin.(pluginLoader.ActionPlugin).SigUsr2()
				}
			}
		}
	}
}
