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

	// Where are we in going through out capture log?
	cursor int

	size     int
	captured []pluginMeta.Trap
}

const pluginName = "replay"
const maxFilesDefault = 100

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
	maxFilesStr := actionArgs["count"]
	if maxFilesStr == "" {
		maxFiles = maxFilesDefault
	} else {
		maxFiles, err = strconv.Atoi(maxFilesStr)
		if err != nil {
			return err
		}
	}

	format := actionArgs["format"]
	switch format {
	case "gob", "":
		format = "gob"
	default:
		return fmt.Errorf("Unknown file format %s", format)
	}

	err = p.preLoadTraps(actionArgs["dir"], maxFiles, "."+format)
	return err
}

func (p replayData) GenerateTrap() (*pluginMeta.Trap, error) {
	p.replayLog.Info().Str("plugin", pluginName).Msg("Replaying trap")

	var trap pluginMeta.Trap
	trap = p.captured[p.cursor]
	p.cursor++
	if p.cursor > p.size {
		p.cursor = 0
	}
	return &trap, nil
}

func (p replayData) Close() error {
	return nil
}

func (p *replayData) preLoadTraps(dir string, maxFiles int, suffix string) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	var i int
	var fd os.FileInfo
	for i, fd = range files {
		if i >= maxFiles {
			break
		}
		filename := fd.Name()
		if strings.HasSuffix(filename, suffix) {
			fullpath := dir + "/" + filename
			trap, err := loadCaptureGob(fullpath)
			if err != nil {
				return err
			}
			p.captured = append(p.captured, trap)
		}
	}
	p.size = len(p.captured)
	if p.size == 0 {
		return fmt.Errorf("No %s format capture files found in directory", suffix)
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

var GeneratorPlugin replayData
