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
        "encoding/gob"

        pluginMeta "github.com/damienstuart/trapex/txPlugins"
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

/*
	processCommandLine()
	if err := getConfig(); err != nil {
		replayLog.Fatal().Err(err).Msg("Unable to load configuration")
		os.Exit(1)
	}
*/

filename := "captured-0.gob"
 trap, err := loadCaptureGob(filename)
 if err != nil {
		replayLog.Fatal().Err(err).Str("format", "gob").Msg("Unable to load capture file")
		os.Exit(1)
 }

var destination DestinationType
  destination.Name = "hello world"
  destination.Plugin = "noop"
  destination.ReplayArgs = make([]ReplayArgType, 0)

  destination.plugin, err = pluginLoader.LoadActionPlugin("/Users/kellskearney/go/src/trapex/txPlugins/actions/%s.so", "logfile")
  if err != nil {
replayLog.Error().Err(err).Str("plugin", "logfile").Msg("Unable to load plugin")
}

actionArgs := map[string]string{"filename": "/Users/kellskearney/go/src/trapex/cmds/replay/replayed.log"}

destination.plugin.(pluginLoader.ActionPlugin).Configure(&replayLog, actionArgs)

			destination.plugin.(pluginLoader.ActionPlugin).ProcessTrap(&trap)
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
/*
func replayTrap(trap pluginMeta.Trap) {
	for _, action := range teConfig.Destinations {
			action.processAction(trap)
	}
}
*/

func loadCaptureGob(filename string) (pluginMeta.Trap, error) {
  var  trap pluginMeta.Trap
        fd, err := os.Open(filename)
        if err != nil {
                return trap, err
        }
        defer fd.Close()

        decoder := gob.NewDecoder(fd)
        err = decoder.Decode(&trap)
        return trap, err
}

