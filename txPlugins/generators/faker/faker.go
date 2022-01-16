// Copyright (c) 2022 Kells Kearney. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

/*
 * This plugin generates traps (possibly cycling between previous entries) based
 * on a sample trap definition
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

type fakerData struct {
	replayLog *zerolog.Logger

	cursor int

	size  int
	faked []pluginMeta.Trap
}

const pluginName = "faker"
const maxFakeTrapsDefault = 100

func validateArguments(actionArgs map[string]string) error {
	validArgs := map[string]bool{"conf": true, "count": true}

	for key, _ := range actionArgs {
		if _, ok := validArgs[key]; !ok {
			return fmt.Errorf("Unrecognized option to %s plugin: %s", pluginName, key)
		}
	}

	return nil
}

func (p *fakerData) Configure(replayLog *zerolog.Logger, actionArgs map[string]string) error {
	p.replayLog = replayLog
	p.replayLog.Info().Str("plugin", pluginName).Msg("Initialization of plugin")

	var err error
	if err = validateArguments(actionArgs); err != nil {
		return err
	}

	var maxFakeTraps int
	maxFakeTrapsStr := actionArgs["count"]
	if maxFakeTrapsStr == "" {
		maxFakeTraps = maxFakeTrapsDefault
	} else {
		maxFakeTraps, err = strconv.Atoi(maxFakeTrapsStr)
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

	err = p.preLoadTraps(actionArgs["dir"], maxFakeTraps, "."+format)
	return err
}

func (p fakerData) GenerateTrap() (*pluginMeta.Trap, error) {
	p.replayLog.Info().Str("plugin", pluginName).Msg("Replaying trap")

	var trap pluginMeta.Trap
	trap = p.faked[p.cursor]
	p.cursor++
	if p.cursor > p.size {
		p.cursor = 0
	}
	return &trap, nil
}

func (p fakerData) Close() error {
	return nil
}

func (p *fakerData) preLoadTraps(dir string, maxFakeTraps int, suffix string) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	var i int
	var fd os.FileInfo
	for i, fd = range files {
		if i >= maxFakeTraps {
			break
		}
		filename := fd.Name()
		if strings.HasSuffix(filename, suffix) {
			fullpath := dir + "/" + filename
			trap, err := loadCaptureGob(fullpath)
			if err != nil {
				return err
			}
			p.faked = append(p.faked, trap)
		}
	}
	p.size = len(p.faked)
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

var GeneratorPlugin fakerData
