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
	/*
	   "net/http"
	   "bytes"
	*/

	"github.com/damienstuart/trapex/actions"
	"github.com/rs/zerolog"
)

type webhookForwarder struct {
	url        string
	timeout    uint
	trapex_log zerolog.Logger
}

const plugin_name = "webhook"

func (a webhookForwarder) Configure(logger zerolog.Logger, actionArg string, pluginConfig *plugin_data.PluginsConfig) error {
	a.trapex_log = logger

	logger.Info().Str("plugin", plugin_name).Msg("Initialization of plugin")
	a.url = actionArg
	logger.Info().Str("url", a.url).Msg("Added webhook destination")
	return nil
}

func (a webhookForwarder) ProcessTrap(trap *plugin_data.Trap) error {
	a.trapex_log.Info().Str("plugin", plugin_name).Msg("Processing HTTP post -- fake")
	trapMap := trap.V1Trap2Map()
	jsonBytes, err := json.Marshal(trapMap)
	if err != nil {
		return err
	}

	a.trapex_log.Info().Str("plugin", plugin_name).Str("json", string(jsonBytes[:])).Msg("Converted trap to JSON")
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
var FilterPlugin webhookForwarder