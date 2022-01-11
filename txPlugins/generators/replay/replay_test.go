// Copyright (c) 2022 Kells Kearney. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"os"
	"testing"

	pluginMeta "github.com/damienstuart/trapex/txPlugins"

	"github.com/rs/zerolog"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

var testLog = zerolog.New(os.Stdout).With().Timestamp().Logger()

func TestGenerator(t *testing.T) {
	var data replayData

	data.replayLog = &testLog
	data.cursor = 0
	data.size = 2
	data.captured = make([]pluginMeta.Trap, 2)
	var trap1, trap2 pluginMeta.Trap
	trap1.Hostname = "tweedle dee"
	trap2.Hostname = "tweedle dum"

	data.captured[0] = trap1
	data.captured[1] = trap2

	var i int
	var err error
	requestedCount := 5
	for i = 0; i < requestedCount; i++ {
		_, err = data.GenerateTrap()
		if err != nil {
			t.Errorf("Received error while iterating through generator")
		}
	}
	if i != 5 {
		t.Errorf("Not able to keep generating traps")
	}
}
