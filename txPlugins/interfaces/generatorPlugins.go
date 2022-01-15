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

type GeneratorPlugin interface {
	Configure(trapexLog *zerolog.Logger, actionArgs map[string]string) error
	GenerateTrap() (*pluginMeta.Trap, error)
	Close() error
}

func LoadGeneratorPlugin(pluginPath string, pluginName string) (GeneratorPlugin, error) {
        plugin_filename := pluginPath + "/generators/" + pluginName + ".so"

	plug, err := plugin.Open(plugin_filename)
	if err != nil {
		return nil, err
	}

	// Load the class from the plugin
	symAction, err := plug.Lookup("GeneratorPlugin")
	if err != nil {
		return nil, err
	}

	var initializer GeneratorPlugin
	// Instantiate the class from the plugin
	initializer, ok := symAction.(GeneratorPlugin)
	if !ok {
		return nil, errors.New("Unable to load plugin " + pluginName)
	}

	return initializer, nil
}

