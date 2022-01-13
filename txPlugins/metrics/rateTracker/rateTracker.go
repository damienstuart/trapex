// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

/*
 * Capture internal metrics and rates
 */

import (
	"math"
	"sync"
	"time"

	//	"fmt"

	pluginMeta "github.com/damienstuart/trapex/txPlugins"

	"github.com/rs/zerolog"
)

const countRingMax_default int = 1448
const countRingMax int = 1448

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

type rateStats struct {
	trapex_log *zerolog.Logger

	StartTime         time.Time
	UptimeInt         int64
	Uptime            string
	TrapCount         uint
	HandledTraps      uint
	DroppedTraps      uint
	IgnoredTraps      uint
	TranslatedFromV2c uint
	TranslatedFromV3  uint
	TrapsPerSecond    trapRates

	countRingSize int
	countRing     tcountRingBuf
}

func (rt *rateStats) Configure(trapexLog *zerolog.Logger, args map[string]string, metric_definitions []pluginMeta.MetricDef) error {
	rt.trapex_log = trapexLog

	rt.trapex_log.Info().Msg("Rate tracker")

	return nil
}

func (rt *rateStats) Inc(metricIndex int) {

	/*
		switch metric {
		case pluginMeta.MetricTotal:
			rt.TrapCount++
		case pluginMeta.MetricHandled:
			rt.HandledTraps++
		case pluginMeta.MetricDropped:
			rt.DroppedTraps++
		case pluginMeta.MetricIgnored:
			rt.IgnoredTraps++
		case pluginMeta.MetricFromV2c:
			rt.TranslatedFromV2c++
		case pluginMeta.MetricFromV3:
			rt.TranslatedFromV3++
		}
	*/
}

func (rt rateStats) Report() (string, error) {
	return "", nil
}

var StatsPlugin rateStats

var stopRateTrackerChan = make(chan struct{})

type tcountRingBuf struct {
	mu     sync.Mutex
	cursor int
	buf    [countRingMax]uint

	// Below are junk for migration purposes
	TrapCount uint
	UptimeInt uint
}

func newTrapRateTracker() *tcountRingBuf {
	tbuf := tcountRingBuf{}
	tbuf.init()
	return &tbuf
}

func (b *tcountRingBuf) init() {
	b.mu.Lock()
	b.cursor = 0
	for i := 0; i < len(b.buf); i++ {
		b.buf[i] = 0
	}
	b.mu.Unlock()
}

func (rt *tcountRingBuf) setNextCount() {
	rt.mu.Lock()
	rt.cursor++
	if rt.cursor >= countRingMax {
		rt.cursor = 0
	}
	rt.buf[rt.cursor] = rt.TrapCount
	rt.mu.Unlock()
}

func (rt *tcountRingBuf) getRate(interval int) uint {
	if interval == 0 {
		if rt.TrapCount == 0 {
			return 0
		}
		return uint(math.Ceil(float64(rt.TrapCount) / float64(rt.UptimeInt)))
	}
	rt.mu.Lock()
	e := rt.cursor
	s := e - interval
	if s < 0 {
		s += countRingMax
	}
	rate := uint(math.Ceil(float64(rt.buf[e]-rt.buf[s]) / float64(interval*60)))
	rt.mu.Unlock()
	return rate
}

func (rt *tcountRingBuf) start() {
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
*/
