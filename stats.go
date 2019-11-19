package main

import (
	"sync"
	"time"
)

const tBufSize int = 3608

type trapRates struct {
	TrapRate1  uint
	TrapRate5  uint
	TrapRate15 uint
	TrapRate60 uint
}

// teStats is a structure for holding trapex stats.
//
type teStats struct {
	StartTime         time.Time
	UptimeInt         int64
	Uptime            string
	TrapCount         uint64
	DroppedTraps      uint64
	TranslatedFromV2c uint64
	TranslatedFromV3  uint64
	trapRates         trapRates
}

var stats teStats

type tcountRingBuf struct {
	mu  sync.Mutex
	ndx int
	buf [tBufSize]uint64
}

func (b *tcountRingBuf) init() {
	b.mu.Lock()
	b.ndx = 0
	for i, _ := range b.buf {
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
	b.mu.Lock()
	e := b.ndx
	s := e - interval
	if s < 0 {
		s += tBufSize
	}
	rate := uint((b.buf[e] - b.buf[s]) / uint64(interval))
	b.mu.Unlock()
	return rate
}

func startRateTracker() {

}
