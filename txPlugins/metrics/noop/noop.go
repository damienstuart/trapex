// Copyright (c) 2022 Kells Kearney. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//

package main

import (
	pluginMeta "github.com/damienstuart/trapex/txPlugins"

	"github.com/rs/zerolog"
)

type noopStats struct {
	log *zerolog.Logger
}

func (rt *noopStats) Configure(trapexLog *zerolog.Logger, args map[string]string, metric_definitions []pluginMeta.MetricDef) error {
	rt.log = trapexLog
	return nil
}

func (rt noopStats) Inc(metricIndex int) {
}

func (rt noopStats) Report() (string, error) {
	return "", nil
}
