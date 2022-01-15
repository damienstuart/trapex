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
	"fmt"
	"math"
	"sync"
	"time"

	pluginMeta "github.com/damienstuart/trapex/txPlugins"

	"github.com/rs/zerolog"
)

const pluginName = "rate tracker"

const (
	pastMinute    = 1
	past5Minutes  = 5
	past15Minutes = 15
	pastHour      = 60
	past4Hours    = 240
	past8Hours    = 480
	pastDay       = 1440
)

type trapRates struct {
	LastMinute    uint
	Last5Minutes  uint
	Last15Minutes uint
	LastHour      uint
	Last4Hours    uint
	Last8Hours    uint
	LastDay       uint
	SinceStart    uint
}

var stopRateTrackerChan = make(chan struct{})

type rateTracker struct {
	lock sync.Mutex

	// Make a rolling window of 24 hours, split up by minute-interval totals (ie counters)
	// The totals are always increasing in value (ie point in time totals)
	// so the last minute interval will always be larger than any previous minute's interval
	now            int
	totalsByMinute [pastDay]uint
}

func newTrapRateTracker() *rateTracker {
	rt := rateTracker{}
	rt.init()
	return &rt
}

// init does a memset of the rolling window (1 day)
//
func (today *rateTracker) init() {
	today.lock.Lock()
	today.now = 0
	for i := 0; i < pastDay; i++ {
		today.totalsByMinute[i] = 0
	}
	today.lock.Unlock()
}

// rollup sets the value of 'now' minute-interval bin to the current total
// and advances 'now' to the next minute
//
func (rt *rateTracker) rollup() {
	rt.lock.Lock()
	rt.now++
	if rt.now >= pastDay {
		rt.now = 0
	}
	rt.totalsByMinute[rt.now] = MetricPlugin.counters[0]
	rt.lock.Unlock()
}

// getRate calculates the average of the rates over the interval
// The interval value of zero indicates the entire rolling window
//
func (rt *rateTracker) getRate(interval int) uint {
	if interval == 0 {
		if MetricPlugin.counters[0] == 0 {
			return 0
		}
		return uint(math.Ceil(float64(MetricPlugin.counters[0]) / float64(MetricPlugin.UptimeInt)))
	}
	rt.lock.Lock()
	beginIntervalMinute := rt.now

	// Calculate the ending index
	endIntervalMinute := beginIntervalMinute - interval
	if endIntervalMinute < 0 {
		endIntervalMinute += pastDay
	}
	rate := uint(math.Ceil(float64(rt.totalsByMinute[beginIntervalMinute]-rt.totalsByMinute[endIntervalMinute]) / float64(interval*60)))
	rt.lock.Unlock()
	return rate
}

// startTicker begins a minute-by-minute function that sets each minute's totals
//
func (rt *rateTracker) startTicker() {
	ticker := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-ticker.C:
			rt.rollup()
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

	}
}
*/

type stats struct {
	log *zerolog.Logger

	StartTime time.Time
	UptimeInt int64
	Uptime    string

	definitions []pluginMeta.MetricDef
	counters    []uint

	TrapsPerSecond trapRates

	rollingDay *rateTracker
}

func (rt *stats) Configure(mainLog *zerolog.Logger, args map[string]string, metric_definitions []pluginMeta.MetricDef) error {
	rt.log = mainLog

	rt.definitions = metric_definitions
	rt.counters = make([]uint, len(rt.definitions))
	rt.StartTime = time.Now()

	rt.rollingDay = newTrapRateTracker()
	rt.log.Info().Str("plugin", pluginName).Msg("Configured metric plugin")

	return nil
}

func (rt *stats) Inc(metricIndex int) {

	name := rt.definitions[metricIndex].Name
	rt.log.Debug().Str("plugin", pluginName).Str("name", name).Msg("Counter incremented")
}

// secondsToDuration converts the given number of seconds into a more
// human-readable formatted string.
//
func secondsToDuration(s uint) string {
	var d uint
	var h uint
	var m uint
	if s >= 86400 {
		d = s / 86400
		s %= 86400
	}
	if s >= 3600 {
		h = s / 3600
		s %= 3600
	}
	if s >= 60 {
		m = s / 60
		s %= 60
	}
	return fmt.Sprintf("%vd-%vh-%vm-%vs", d, h, m, s)
}

func (rt stats) Report() (string, error) {
	MetricPlugin.UptimeInt = time.Now().Unix() - MetricPlugin.StartTime.Unix()
	for i, counter := range rt.definitions {
		MetricPlugin.log.Info().
			Uint(counter.Name, rt.counters[i]).Msg("Counter value")
	}
	MetricPlugin.log.Info().
		Str("uptime_str", secondsToDuration(uint(MetricPlugin.UptimeInt))).
		Uint("uptime", uint(MetricPlugin.UptimeInt)).
		Uint("rate_1min", rt.rollingDay.getRate(pastMinute)).
		Uint("rate_5min", rt.rollingDay.getRate(past5Minutes)).
		Uint("rate_15min", rt.rollingDay.getRate(past15Minutes)).
		Uint("rate_1hour", rt.rollingDay.getRate(pastHour)).
		Uint("rate_4hour", rt.rollingDay.getRate(past4Hours)).
		Uint("rate_8hour", rt.rollingDay.getRate(past8Hours)).
		Uint("rate_1day", rt.rollingDay.getRate(pastDay)).
		Uint("rate_all", rt.rollingDay.getRate(0)).Msg("Current rates")

	return "", nil
}

var MetricPlugin stats
