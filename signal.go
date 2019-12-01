package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// For now we only need to handle SIGHUP to force a configuration reload.
func initSigHandlers() {
	sigHupCh := make(chan os.Signal, 1)
	signal.Notify(sigHupCh, syscall.SIGHUP)
	go func() {
		for {
			select {
			case <-sigHupCh:
				fmt.Println("Got SIGHUP")
				getConfig()
			}
		}
	}()
}
