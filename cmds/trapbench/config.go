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

type CommandLine struct {
	configFile string
}

// Global vars
//
var teConfig *replayConfig
var teCmdLine CommandLine

func showUsage() {
	usageText := `
Usage:
  trapbench -h
  trapbench -v
  trapbench [-c <config_file>]

Usage:
  -h  - Show this help message and exit.
  -c  - Override the location of the trapbench configuration file.
  -v  - Print the version of trapbench and exit.
`
	/*
	   -n requests     Number of requests to perform
	   -c concurrency  Number of multiple requests to make at a time
	   -t timelimit    Seconds to max. to spend on benchmarking
	                   This implies -n 50000
	   -B address      Address to bind to when making outgoing connections
	   -v verbosity    How much troubleshooting info to print
	   -g filename     Output collected data to gnuplot format file.
	*/

	fmt.Println(usageText)
}

var myVersion string = "1.0"

func processCommandLine() {
	flag.Usage = showUsage
	c := flag.String("c", "/opt/trapex/etc/replay.yml", "")
	showVersion := flag.Bool("v", false, "")

	flag.Parse()

	if *showVersion {
		fmt.Printf("This is replay version %s\n", myVersion)
		os.Exit(0)
	}

	teCmdLine.configFile = *c
}

func isFile(path string) bool {
	fd, err := os.Stat(path)
	if err != nil {
		fmt.Printf("The argument '%s' is not valid: %s\n", path, err)
		os.Exit(1)
	}

	result := true
	mode := fd.Mode()
	if mode.IsDir() {
		result = false
	}
	return result
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

	if err = setGenerator(&newConfig.Generator, newConfig.Generator.PluginPathExpr); err != nil {
		return err
	}
	if err = setAction(&newConfig.Destination, newConfig.Destination.PluginPathExpr); err != nil {
		return err
	}

	teConfig = &newConfig

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

func setGenerator(generator *GeneratorType, pluginPathExpr string) error {
	var err error

	generator.plugin, err = pluginLoader.LoadGeneratorPlugin(pluginPathExpr, generator.PluginName)
	if err != nil {
		return fmt.Errorf("Unable to load generator plugin %s: %s", generator.PluginName, err)
	}
	pluginDataMapping := args2map(generator.Args)
	if err = generator.plugin.Configure(&replayLog, pluginDataMapping); err != nil {
		return fmt.Errorf("Unable to configure generator plugin %s: %s", generator.PluginName, err)
	}
	return nil
}

func setAction(destination *DestinationType, pluginPathExpr string) error {
	var err error

	destination.plugin, err = pluginLoader.LoadActionPlugin(pluginPathExpr, destination.PluginName)
	if err != nil {
		return fmt.Errorf("Unable to load %s plugin %s: %s", destination.Name, destination.PluginName, err)
	}
	pluginDataMapping := args2map(destination.ReplayArgs)
	if err = destination.plugin.Configure(&replayLog, pluginDataMapping); err != nil {
		return fmt.Errorf("Unable to configure action %s plugin %s: %s", destination.Name, destination.PluginName, err)
	}
	return nil
}
