// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

/*
 * Capture internal definitions and rates
 *
 * Note:
 *   This plugin ***ASSUMES*** that the very first counter (index 0) is the total
 *   number of incoming things (ie traps).
 */

import (
	"math"
	"sync"
	"time"

	pluginMeta "github.com/damienstuart/trapex/txPlugins"

	"github.com/rs/zerolog"
)

const pluginName = "rate tracker"

// 60 minutes/hour * 24 hours = 1440
const maxHistoryBins int = 1440

type trapRates struct {
	Last1min   uint
	Last5min   uint
	Last15min  uint
	Last1hour  uint
	Last4hour  uint
	Last8hour  uint
	Last1day   uint
	SinceStart uint
}

var stopRateTrackerChan = make(chan struct{})

type rateBin struct {
	mu     sync.Mutex
	cursor int
	buf    [maxHistoryBins]uint

}

func newTrapRateTracker() *rateBin {
	tbuf := rateBin{}
	tbuf.init()
	return &tbuf
}

func (b *rateBin) init() {
	b.mu.Lock()
	b.cursor = 0
	for i := 0; i < len(b.buf); i++ {
		b.buf[i] = 0
	}
	b.mu.Unlock()
}

func (rt *rateBin) setNextCount() {
	rt.mu.Lock()
	rt.cursor++
	if rt.cursor >= maxHistoryBins {
		rt.cursor = 0
	}
	rt.buf[rt.cursor] = MetricPlugin.counters[0]
	rt.mu.Unlock()
}

func (rt *rateBin) getRate(interval int) uint {
	if interval == 0 {
		if MetricPlugin.counters[0] == 0 {
			return 0
		}
		return uint(math.Ceil(float64(MetricPlugin.counters[0]) / float64(MetricPlugin.UptimeInt)))
	}
	rt.mu.Lock()
	e := rt.cursor
	s := e - interval
	if s < 0 {
		s += maxHistoryBins
	}
	rate := uint(math.Ceil(float64(rt.buf[e]-rt.buf[s]) / float64(interval*60)))
	rt.mu.Unlock()
	return rate
}

func (rt *rateBin) start() {
	ticker := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-ticker.C:
			rt.setNextCount()
		case <-stopRateTrackerChan:
			ticker.Stop()
			return
		}
	}
}

/*
// Use SIGUSR1 to dump current stats to STDOUT.
//
func handleSIGUSR1(sigCh chan os.Signal) {
	for {
		select {
		case <-sigCh:
			// Compute uptime
			MetricPlugin.UptimeInt = time.Now().Unix() - MetricPlugin.StartTime.Unix()
			MetricPlugin.log.Info().
				Str("uptime_str", secondsToDuration(uint(MetricPlugin.UptimeInt))).
				Uint("uptime", uint(MetricPlugin.UptimeInt)).
				Uint("traps_received", MetricPlugin.counters[0]).
				Uint("traps_ignored", stats.IgnoredTraps).
				Uint("traps_processed", MetricPlugin.HandledTraps).
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
		}
	}
}
*/

type stats struct {
	log *zerolog.Logger

	StartTime         time.Time
	UptimeInt         int64
	Uptime            string

        definitions []pluginMeta.MetricDef
        counters []uint

	TrapsPerSecond    trapRates

	rateBins     *rateBin
}

func (rt *stats) Configure(mainLog *zerolog.Logger, args map[string]string, metric_definitions []pluginMeta.MetricDef) error {
	rt.log = mainLog

        rt.definitions = metric_definitions
        rt.counters = make([]uint, len(rt.definitions))
        rt.StartTime = time.Now()

rt.rateBins = newTrapRateTracker()
	rt.log.Info().Str("plugin", pluginName).Msg("Configured metric plugin")

	return nil
}

func (rt *stats) Inc(metricIndex int) {

name := rt.definitions[metricIndex].Name
	rt.log.Debug().Str("plugin", pluginName).Str("name", name).Msg("Counter incremented")
}

func (rt stats) Report() (string, error) {
	return "", nil
}


var MetricPlugin stats

