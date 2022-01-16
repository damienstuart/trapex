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
	sinceStart    = 0
	pastMinute    = 1
	past5Minutes  = 5
	past15Minutes = 15
	pastHour      = 60
	past4Hours    = 240
	past8Hours    = 480
	pastDay       = 1440
)

type stats struct {
	log *zerolog.Logger

	StartTime time.Time
	UptimeInt int64

	definitions []pluginMeta.MetricDef
	counters    []uint

	rollingDay *rateTracker
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

// getCurrentTotal returns the current global counter 'total'
//
func getCurrentTotal() uint {
	return MetricPlugin.counters[0]
}

// setThisMinutesTotal sets the value of 'now' minute-interval bin to the current total
// and advances 'now' to the next minute
//
func (rt *rateTracker) setThisMinutesTotal() {
	rt.lock.Lock()
	rt.now++
	if rt.now >= pastDay {
		rt.now = 0
	}
	rt.totalsByMinute[rt.now] = getCurrentTotal()
	rt.lock.Unlock()
}

// getRate calculates the average of the rates over the interval
// The interval value of zero indicates the entire rolling window
//
func (rt *rateTracker) getRate(interval int) uint {
	if interval == 0 {
		currentTotal := getCurrentTotal()
		if currentTotal == 0 {
			return 0
		}
		return uint(math.Ceil(float64(currentTotal) / float64(MetricPlugin.UptimeInt)))
	}

	rt.lock.Lock()
	beginIntervalMinute := rt.now

	// Calculate the ending index
	endIntervalMinute := beginIntervalMinute - interval
	if endIntervalMinute < 0 {
		endIntervalMinute += pastDay
	}

	periodTotal := float64(rt.totalsByMinute[beginIntervalMinute] - rt.totalsByMinute[endIntervalMinute])
	rt.lock.Unlock()
	periodInterval := float64(interval * 60)
	rate := uint(math.Ceil(periodTotal / periodInterval))
	return rate
}

// startTicker begins a minute-by-minute function that sets each minute's totals
//
func (rt *rateTracker) startTicker() {
	ticker := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-ticker.C:
			rt.setThisMinutesTotal()
		case <-stopRateTrackerChan:
			ticker.Stop()
			return
		}
	}
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
func secondsToDuration(seconds uint) string {
	var days, hours, minutes uint

	if seconds >= 86400 {
		days = seconds / 86400
		seconds %= 86400
	}
	if seconds >= 3600 {
		hours = seconds / 3600
		seconds %= 3600
	}
	if seconds >= 60 {
		minutes = seconds / 60
		seconds %= 60
	}
	return fmt.Sprintf("%vd-%vh-%vm-%vs", days, hours, minutes, seconds)
}

func (rt stats) Report() (string, error) {
	// Only calculate uptime when we need to do so
	MetricPlugin.UptimeInt = time.Now().Unix() - MetricPlugin.StartTime.Unix()

	// Report on counters
	for i, counter := range rt.definitions {
		MetricPlugin.log.Info().
			Uint(counter.Name, rt.counters[i]).Msg("Counter value")
	}

	// Report on the rates
	// FIXME: Because we get the rates one at a time, which lock/unlock operations one at a time,
	// FIXME: there will be differences between the 'now' which is used to calculate the rates.
	// FIXME: The ticker can update between the various calls got getRate()
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
		Uint("rate_all", rt.rollingDay.getRate(sinceStart)).Msg("Current rates")

	return "", nil
}

var MetricPlugin stats
