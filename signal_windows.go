// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"os"
	"os/signal"
	"syscall"
)

// For now we only need to handle SIGHUP to force a configuration reload.
func initSigHandlers() {
	sigHupCh := make(chan os.Signal, 1)
	signal.Notify(sigHupCh, syscall.SIGHUP)
	go handleSIGHUP(sigHupCh)
}
