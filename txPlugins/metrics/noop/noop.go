// Copyright (c) 2022 Kells Kearney. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//

package main

/*
 * This plugin is used for testing and performance benchmarking
 */

import (
	pluginMeta "github.com/damienstuart/trapex/txPlugins"

	"github.com/rs/zerolog"
)

const pluginName = "noop"

type noopStats struct {
	log     *zerolog.Logger
	metrics []pluginMeta.MetricDef
}

func (rt *noopStats) Configure(mainLog *zerolog.Logger, args map[string]string, metric_definitions []pluginMeta.MetricDef) error {
	rt.log = mainLog
	rt.log.Info().Str("plugin", pluginName).Msg("Configured metric plugin")
	rt.metrics = metric_definitions
	return nil
}

func (rt noopStats) Inc(metricIndex int) {
	name := rt.metrics[metricIndex].Name
	rt.log.Info().Str("plugin", pluginName).Str("metric", name).Msg("Counter incremented")

}

func (rt noopStats) Report() (string, error) {
	return "", nil
}

var MetricPlugin noopStats
