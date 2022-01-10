// Copyright (c) 2022 Kells Kearney. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

/*
 * This plugin generates traps (possibly cycling between previous entries) based
 * on previously captured traps.
 */

import (
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	pluginMeta "github.com/damienstuart/trapex/txPlugins"

	"github.com/rs/zerolog"
)

type replayData struct {
	replayLog *zerolog.Logger

	// count - if count == -1, that means to run until infinity
	maxFiles int

	// Where are we in going through out capture log?
	cursor   int
	captured []pluginMeta.Trap
}

const pluginName = "replay"

func validateArguments(actionArgs map[string]string) error {
	validArgs := map[string]bool{"dir": true, "count": true, "format": true}

	for key, _ := range actionArgs {
		if _, ok := validArgs[key]; !ok {
			return fmt.Errorf("Unrecognized option to %s plugin: %s", pluginName, key)
		}
	}

	return nil
}

func (p *replayData) Configure(replayLog *zerolog.Logger, actionArgs map[string]string) error {
	p.replayLog = replayLog
	p.replayLog.Info().Str("plugin", pluginName).Msg("Initialization of plugin")

	var err error
	if err = validateArguments(actionArgs); err != nil {
		return err
	}

	var maxFiles int
	maxFiles, err = strconv.Atoi(actionArgs["count"])
	if err != nil {
		return err
	}
	p.maxFiles = maxFiles

	err = p.preLoadTraps(actionArgs["dir"], maxFiles)
	return err
}

func (p replayData) GenerateTrap() (*pluginMeta.Trap, error) {
	p.replayLog.Info().Str("plugin", pluginName).Msg("Replaying trap")

	p.cursor++
	if p.cursor > p.maxFiles {
		p.cursor = 0
	}
	return &p.captured[p.cursor], nil
}

func (p replayData) Close() error {
	return nil
}

func (p *replayData) preLoadTraps(dir string, maxFiles int) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		p.replayLog.Fatal().Err(err).Str("dir", dir).Msg("Unable to process capture file directory")
	}

	var i int
	var fd os.FileInfo
	for i, fd = range files {
		if maxFiles != -1 && i > maxFiles {
			break
		}
		filename := fd.Name()
		if strings.HasSuffix(".gob", filename) {

			trap, err := loadCaptureGob(filename)
			if err != nil {
				return err
			}
			p.captured = append(p.captured, trap)
		}
	}
	if maxFiles == -1 {
		p.maxFiles = i
	}
	return nil
}

func loadCaptureGob(filename string) (pluginMeta.Trap, error) {
	var trap pluginMeta.Trap
	fd, err := os.Open(filename)
	if err != nil {
		return trap, err
	}
	defer fd.Close()

	decoder := gob.NewDecoder(fd)
	err = decoder.Decode(&trap)
	return trap, err
}

var ActionPlugin replayData
