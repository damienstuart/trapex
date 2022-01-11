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

	pluginLoader "github.com/damienstuart/trapex/txPlugins/interfaces"

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
	if err := getConfig(); err != nil {
		replayLog.Fatal().Err(err).Msg("Unable to load configuration")
		os.Exit(1)
	}
	var count int

	if teConfig.Generator.Stream {
		for { // infinte loop
			genAndActionTrap()
		}
	} else {
		for i := 0; i < teConfig.Generator.Count; i++ {
			genAndActionTrap()
		}
	}

	replayLog.Info().Int("replayed_traps", count).Msg("Replayed traps")
	/*
	   	startTime := time.Now()
	   	endTime := time.Now()
	           duration := endTime - startTime
	   		replayLog.Info().Int("replay_duration", duration).Msg("Replayed trap in %v seconds")

	*/
}

func genAndActionTrap() {
	trap, err := teConfig.Generator.plugin.(pluginLoader.GeneratorPlugin).GenerateTrap()
	if err != nil {
		replayLog.Fatal().Err(err).Msg("Unable to generate trap")
	}
	err = teConfig.Destination.plugin.(pluginLoader.ActionPlugin).ProcessTrap(trap)
	if err != nil {
		replayLog.Fatal().Err(err).Msg("Unable to forward trap")
	}
}
