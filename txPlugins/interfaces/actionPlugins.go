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

// Filter action plugin interface
type ActionPlugin interface {
	Configure(trapexLog *zerolog.Logger, actionArgs map[string]string) error
	ProcessTrap(trap *pluginMeta.Trap) error
	SigUsr1() error
	SigUsr2() error
	Close() error
}

func LoadActionPlugin(pluginPath string, plugin_name string) (ActionPlugin, error) {
	plugin_filename := pluginPath + "/actions/" + plugin_name + ".so"

	plug, err := plugin.Open(plugin_filename)
	if err != nil {
		return nil, err
	}

	// Load the class from the plugin
	symAction, err := plug.Lookup("ActionPlugin")
	if err != nil {
		return nil, err
	}

	var initializer ActionPlugin
	// Instantiate the class from the plugin
	initializer, ok := symAction.(ActionPlugin)
	if !ok {
		return nil, errors.New("Unable to load plugin " + plugin_name)
	}

	return initializer, nil
}

