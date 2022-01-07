// Copyright (c) 2022 Kells Kearney. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

        pluginMeta "github.com/damienstuart/trapex/txPlugins"

	"github.com/rs/zerolog"
)

var replayLog = zerolog.New(os.Stdout).With().Timestamp().Logger()

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	flag.Usage = func() {
		fmt.Printf("Usage:\n")
		fmt.Printf("   %s\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	processCommandLine()
/*
	if err := getConfig(); err != nil {
		replayLog.Fatal().Err(err).Msg("Unable to load configuration")
		os.Exit(1)
	}
*/

/*
	startTime := time.Now()
	endTime := time.Now()
        duration := endTime - startTime
		replayLog.Info().Int("replay_duration", duration).Msg("Replayed trap in %v seconds")

*/
}

// replayTrap is the entry point to code that checks the incoming trap
// against the filter list and processes the trap accordingly.
//
func replayTrap(sgt *pluginMeta.Trap) {
	for _, action := range teConfig.Destinations {
			action.processAction(sgt)
	}
}
