// Copyright (c) 2022 Kells Kearney. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package pluginLoader

import (
	"errors"
	"plugin"

	pluginMeta "github.com/damienstuart/trapex/txPlugins"
	"github.com/rs/zerolog"
)

type MetricPlugin interface {
	Configure(trapexLog *zerolog.Logger, args map[string]string, metric_definitions []pluginMeta.MetricDef) error
	Inc(int)
	Report() (string, error)
}

func LoadMetricPlugin(pluginPath string, pluginName string) (MetricPlugin, error) {
        plugin_filename := pluginPath + "/metrics/" + pluginName + ".so"

	plug, err := plugin.Open(plugin_filename)
	if err != nil {
		return nil, err
	}

	// Load the class from the plugin
	symAction, err := plug.Lookup("MetricPlugin")
	if err != nil {
		return nil, err
	}

	var initializer MetricPlugin
	// Instantiate the class from the plugin
	initializer, ok := symAction.(MetricPlugin)
	if !ok {
		return nil, errors.New("Unable to load plugin " + pluginName)
	}

	return initializer, nil
}

