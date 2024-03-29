// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"math"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const tBufSize int = 1448

var stopRateTrackerChan = make(chan struct{})

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

// teStats is a structure for holding trapex stats.
//
type teStats struct {
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
}

var stats teStats

// Prometheus statistics
var (
	trapsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "trapex_incoming_traps_total",
		Help: "The total number of incoming SNMP traps",
	})
	trapsHandled = promauto.NewCounter(prometheus.CounterOpts{
		Name: "trapex_handled_traps_total",
		Help: "The total number of handled SNMP traps",
	})
	trapsDropped = promauto.NewCounter(prometheus.CounterOpts{
		Name: "trapex_dropped_traps_total",
		Help: "The total number of dropped SNMP traps",
	})
	trapsIgnored = promauto.NewCounter(prometheus.CounterOpts{
		Name: "trapex_ignored_traps_total",
		Help: "The total number of ignored SNMP traps",
	})
	trapsFromV2c = promauto.NewCounter(prometheus.CounterOpts{
		Name: "trapex_v2c_traps_total",
		Help: "The total number of SNMPv2c traps translated",
	})
	trapsFromV3 = promauto.NewCounter(prometheus.CounterOpts{
		Name: "trapex_v3_traps_total",
		Help: "The total number of SNMPv3 traps translated",
	})
)

type tcountRingBuf struct {
	mu  sync.Mutex
	ndx int
	buf [tBufSize]uint
}

func newTrapRateTracker() *tcountRingBuf {
	tbuf := tcountRingBuf{}
	tbuf.init()
	return &tbuf
}

func (b *tcountRingBuf) init() {
	b.mu.Lock()
	b.ndx = 0
	for i := 0; i < len(b.buf); i++ {
		b.buf[i] = 0
	}
	b.mu.Unlock()
}

func (b *tcountRingBuf) setNextCount() {
	b.mu.Lock()
	b.ndx++
	if b.ndx >= tBufSize {
		b.ndx = 0
	}
	b.buf[b.ndx] = stats.TrapCount
	b.mu.Unlock()
}

func (b *tcountRingBuf) getRate(interval int) uint {
	if interval == 0 {
		if stats.TrapCount == 0 {
			return 0
		}
		return uint(math.Ceil(float64(stats.TrapCount) / float64(stats.UptimeInt)))
	}
	b.mu.Lock()
	e := b.ndx
	s := e - interval
	if s < 0 {
		s += tBufSize
	}
	rate := uint(math.Ceil(float64(b.buf[e]-b.buf[s]) / float64(interval*60)))
	b.mu.Unlock()
	return rate
}

func (b *tcountRingBuf) start() {
	ticker := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-ticker.C:
			b.setNextCount()
		case <-stopRateTrackerChan:
			ticker.Stop()
			return
		}
	}
}
