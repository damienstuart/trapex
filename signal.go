package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	g "github.com/gosnmp/gosnmp"
)

func initSigHandlers() {
	// For HUP
	sigHupCh := make(chan os.Signal, 1)
	signal.Notify(sigHupCh, syscall.SIGHUP)
	go handleSIGHUP(sigHupCh)
	// For USR1
	sigUsr1Ch := make(chan os.Signal, 1)
	signal.Notify(sigUsr1Ch, syscall.SIGUSR1)
	go handleSIGUSR1(sigUsr1Ch)
	// For USR2
	sigUsr2Ch := make(chan os.Signal, 1)
	signal.Notify(sigUsr2Ch, syscall.SIGUSR2)
	go handleSIGUSR2(sigUsr2Ch)
}

// On SIGHUP we reload the configuration.
//
func handleSIGHUP(sigCh chan os.Signal) {
	for {
		select {
		case <-sigCh:
			fmt.Printf("Got SIGHUP - Reloading configuration.\n")
			if err := getConfig(); err != nil {
				fmt.Printf("Error parsing configuration: %s\nConfiguration was not changed.", err)
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
			fmt.Printf("Got SIGUSR1\n")
			// Compute uptime
			now := time.Now()
			stats.UptimeInt = now.Unix() - stats.StartTime.Unix()
			fmt.Printf("Trapex stats as of: %s\n", now.Format(time.RFC1123))
			fmt.Printf(" - Uptime..............: %s\n", secondsToDuration(uint(stats.UptimeInt)))
			fmt.Printf(" - Traps Received......: %v\n", stats.TrapCount)
			// Only show ignored trap count if we are ignoring any
			if len(teConfig.ignoreVersions) > 0 {
				fmt.Printf(" - Traps Ignored.......: %v\n", stats.IgnoredTraps)
			}
			fmt.Printf(" - Traps Processed.....: %v\n", stats.HandledTraps)
			fmt.Printf(" - Traps Dropped.......: %v\n", stats.DroppedTraps)
			// No need to show translation stats for version we are ignoring.
			if !isIgnoredVersion(g.Version2c) {
				fmt.Printf(" - Translated from v2c.: %v\n", stats.TranslatedFromV2c)
			}
			if !isIgnoredVersion(g.Version3) {
				fmt.Printf(" - Translated from v3..: %v\n", stats.TranslatedFromV3)
			}
			fmt.Printf(" - Trap Rates (based on all traps received):\n")
			fmt.Printf("    - Last Minute......: %v\n", trapRateTracker.getRate(1))
			fmt.Printf("    - Last 5 Minutes...: %v\n", trapRateTracker.getRate(5))
			fmt.Printf("    - Last 15 Minutes..: %v\n", trapRateTracker.getRate(15))
			fmt.Printf("    - Last Hour........: %v\n", trapRateTracker.getRate(60))
			fmt.Printf("    - Last 4 Hours.....: %v\n", trapRateTracker.getRate(240))
			fmt.Printf("    - Last 8 Hours.....: %v\n", trapRateTracker.getRate(480))
			fmt.Printf("    - Last 24 Hours....: %v\n", trapRateTracker.getRate(1440))
			fmt.Printf("    - Since Start......: %v\n", trapRateTracker.getRate(0))
		}
	}
}

// Use SIGUSR2 to force a rotation of CSV log files.
//
func handleSIGUSR2(sigCh chan os.Signal) {
	for {
		select {
		case <-sigCh:
			fmt.Printf("Got SIGUSR2.\n")
			for _, f := range teConfig.filters {
				if f.actionType == actionCsv || f.actionType == actionCsvBreak {
					f.action.(*trapCsvLogger).rotateLog()
					fmt.Printf("Rotated CSV file: %v\n", f.action.(*trapCsvLogger).logfileName())
				}
			}
		}
	}
}
