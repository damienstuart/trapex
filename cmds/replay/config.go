// Copyright (c) 2022 Kells Kearney. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	pluginMeta "github.com/damienstuart/trapex/txPlugins"
	pluginLoader "github.com/damienstuart/trapex/txPlugins/interfaces"

	"github.com/creasty/defaults"
	"gopkg.in/yaml.v2"
)


type replayCommandLine struct {
	configFile string
	debugMode  bool
}

// Global vars
//
var teConfig *replayConfig
var teCmdLine replayCommandLine

func showUsage() {
	usageText := `
Usage: replay [-h] [-c <config_file>] [-d] [-v]
  -h  - Show this help message and exit.
  -c  - Override the location of the replay configuration file.
  -d  - Enable debug mode (note: produces very verbose runtime output).
  -v  - Print the version of replay and exit.
`
	fmt.Println(usageText)
}

var myVersion string = "1.0"

func processCommandLine() {
	flag.Usage = showUsage
	c := flag.String("c", "/opt/trapex/etc/replay.yml", "")
	d := flag.Bool("d", false, "")
	showVersion := flag.Bool("v", false, "")

	flag.Parse()

	if *showVersion {
		fmt.Printf("This is replay version %s\n", myVersion)
		os.Exit(0)
	}

	teCmdLine.configFile = *c
	teCmdLine.debugMode = *d
}

// loadConfig
// Load a YAML file with configuration, and create a new object
func loadConfig(config_file string, newConfig *replayConfig) error {
	defaults.Set(newConfig)

	filename, _ := filepath.Abs(config_file)
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	err = yaml.UnmarshalStrict(yamlFile, newConfig)
	if err != nil {
		return err
	}

	return nil
}

func applyCliOverrides(newConfig *replayConfig) {
	if teCmdLine.debugMode {
		newConfig.General.LogLevel = "debug"
	}
	if newConfig.General.Hostname == "" {
		myName, err := os.Hostname()
		if err != nil {
			newConfig.General.Hostname = "_undefined"
		} else {
			newConfig.General.Hostname = myName
		}
	}
}

func getConfig() error {
	replayLog.Info().Str("version", myVersion).Str("configuration_file", teCmdLine.configFile).Msg("Loaded configuration for replay")

	var newConfig replayConfig
	err := loadConfig(teCmdLine.configFile, &newConfig)
	if err != nil {
		return err
	}
	applyCliOverrides(&newConfig)

	teConfig = &newConfig

	return nil
}

func replayToAllDestinations(newConfig *replayConfig) error {
	var err error
	for _, destination := range newConfig.Destinations {
		if err = replayToDestination(destination, newConfig.General.PluginPathExpr); err != nil {
			return err
		}
	}
	return nil
}


func replayToDestination(destination DestinationType, pluginPathExpr string) error {

		plugin, err := pluginLoader.LoadActionPlugin(pluginPathExpr, destination.Plugin)
		if err != nil {
			return fmt.Errorf("Unable to load plugin %s: %s", destination, err)
		}
		pluginDataMapping := args2map(destination.ReplayArgs)
		if err = plugin.Configure(&replayLog, pluginDataMapping); err != nil {
			return fmt.Errorf("Unable to configure plugin %s: %s", destination, err)
		}
//plugin.processTrap(trap)
	return nil
}

func args2map(data []ReplayArgType) map[string]string {
	var pluginDataMapping map[string]string

	pluginDataMapping = make(map[string]string)
	for _, pair := range data {
		if strings.Contains(pair.Key, "secret") ||
			strings.Contains(pair.Key, "password") {
			plaintext, err := pluginMeta.GetSecret(pair.Value)
			if err != nil {
				replayLog.Warn().Err(err).Str("secret", pair.Key).Str("cipher_text", pair.Value).Msg("Unable to decode secret")
			} else {
				pair.Value = plaintext
			}
		}
		pluginDataMapping[pair.Key] = pair.Value
	}
	return pluginDataMapping
}

