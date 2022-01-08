// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

/*
 * This plugin sends SNMP traps as JSON to a webhook server
 */

import (
	"encoding/json"
	"fmt"

	/*
	   "net/http"
	   "bytes"
	*/

	pluginMeta "github.com/damienstuart/trapex/txPlugins"
	"github.com/rs/zerolog"
)

type webhookForwarder struct {
	url       string
	timeout   uint
	trapexLog *zerolog.Logger
}

const pluginName = "webhook"

func validateArguments(actionArgs map[string]string) error {
	validArgs := map[string]bool{"url": true, "timeout": true}

	for key, _ := range actionArgs {
		if _, ok := validArgs[key]; !ok {
			return fmt.Errorf("Unrecognized option to %s plugin: %s", pluginName, key)
		}
	}
	return nil
}

func (a *webhookForwarder) Configure(trapexLog *zerolog.Logger, actionArgs map[string]string) error {
	a.trapexLog = trapexLog

	a.trapexLog.Info().Str("plugin", pluginName).Msg("Initialization of plugin")
	if err := validateArguments(actionArgs); err != nil {
		return err
	}
	a.url = actionArgs["url"]
	a.trapexLog.Info().Str("url", a.url).Msg("Added webhook destination")
	return nil
}

func (a webhookForwarder) ProcessTrap(trap *pluginMeta.Trap) error {
	a.trapexLog.Info().Str("plugin", pluginName).Msg("Processing HTTP post -- fake")
	trapMap := trap.Trap2Map()
	jsonBytes, err := json.Marshal(trapMap)
	if err != nil {
		return err
	}

	a.trapexLog.Info().Str("plugin", pluginName).Str("json", string(jsonBytes[:])).Msg("Converted trap to JSON")
	/*
	   body := new(bytes.Buffer)
	   trap.toJson(&body)
	   // need timeout information
	   req, _ := http.NewRequest("POST", a.url, body)

	   client := &http.Client()
	   result, err := client.Do(req)
	   if err != nil {
	   return err
	   }

	   defer result.Body.Close()
	   a.trapex_log.Info().Str("url", a.url).Int("http_status", result.Status).Msg("Webhook HTTP status")
	   if result.Status == 200 {
	   a.trapex_log.Info().Str("url", a.url).Str("body, result.Body).Msg("Webhook HTTP result")
	   } else {
	   return error.New("Unable to forward to webhook server")
	*/
	return nil
}

func (p webhookForwarder) SigUsr1() error {
	return nil
}

func (p webhookForwarder) SigUsr2() error {
	return nil
}

func (a webhookForwarder) Close() error {
	return nil
}

// Exported symbol which supports filter.go's FilterAction type
var ActionPlugin webhookForwarder
