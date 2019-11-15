package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
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
			fmt.Printf("Got SIGHUP\n")
			getConfig()
		}
	}
}

// TODO: Decide with to do with a SIGUSR1. Maybe print stats.
//
func handleSIGUSR1(sigCh chan os.Signal) {
	for {
		select {
		case <-sigCh:
			fmt.Printf("Got SIGUSR1\n")
		}
	}
}

// TODO: Decide with to do with a SIGUSR2. Maybe print stats.
//
func handleSIGUSR2(sigCh chan os.Signal) {
	for {
		select {
		case <-sigCh:
			fmt.Printf("Got SIGUSR2\n")
		}
	}
}
