// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

/*
 * This plugin sends SNMP traps to a new destination
 */

/*
 TOOD:
    - create a channel-based forwarding architecture to increase concurrency
*/

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	pluginMeta "github.com/damienstuart/trapex/txPlugins"
	g "github.com/gosnmp/gosnmp"

	"github.com/rs/zerolog"
)

type trapForwarder struct {
	destination *g.GoSNMP
	trapex_log  *zerolog.Logger
}

const pluginName = "trap forwarder"

func validateArguments(snmpVersion g.SnmpVersion, actionArgs map[string]string) error {
	validArgs := map[string]bool{"traphost": true, "port": true, "snmp_version": true, "community": true}
	validV3Args := map[string]bool{"engine_id": true, "auth_password": true, "auth_protocol": true, "privacy_protocol": true, "privacy_password": true}

	for key, _ := range actionArgs {
		if _, ok := validArgs[key]; !ok {
			if snmpVersion == g.Version3 {
				if _, ok := validV3Args[key]; !ok {
					return fmt.Errorf("Unrecognized option to %s plugin: %s", pluginName, key)
				}
			}
		}
	}

	return nil
}

func getVersion(snmpVersion string) (g.SnmpVersion, error) {
	switch strings.ToLower(snmpVersion) {
	case "v1", "1", "":
		return g.Version1, nil
	case "v2c", "2c", "2":
		return g.Version2c, nil
	case "v3", "3":
		return g.Version3, nil
	default:
		return g.Version1, fmt.Errorf("Unsupported or invalid value (%s) for SNMP version", snmpVersion)
	}
}

func (a *trapForwarder) Configure(trapexLog *zerolog.Logger, actionArgs map[string]string) error {
	a.trapex_log = trapexLog

	a.trapex_log.Info().Str("plugin", pluginName).Msg("Initialization of plugin")

	snmpVersion, err := getVersion(actionArgs["version"])
	if err != nil {
		return err
	}

	if err := validateArguments(snmpVersion, actionArgs); err != nil {
		return err
	}

	hostname := actionArgs["traphost"]
	port_str := actionArgs["port"]
	port, err := strconv.Atoi(port_str)
	if err != nil {
		panic("Invalid destination port: " + port_str)
	}
	var community string
	community, _ = actionArgs["community"]
	a.destination = &g.GoSNMP{
		Target:             hostname,
		Port:               uint16(port),
		Transport:          "udp",
		Community:          community,
		Version:            snmpVersion,
		Timeout:            time.Duration(2) * time.Second,
		Retries:            3,
		ExponentialTimeout: true,
		MaxOids:            g.MaxOids,
	}
	err = a.destination.Connect()
	if err != nil {
		return err
	}
	a.trapex_log.Info().Str("target", hostname).Str("port", port_str).Msg("Added trap destination")

	return nil
}

func setSnmpV3Args(destination *g.GoSNMP, params map[string]string) error {
	if destination.Version != g.Version3 {
		return nil
	}

	destination.SecurityModel = g.UserSecurityModel

	switch strings.ToLower(params["msg_flags"]) {
	case "noauthnopriv":
		destination.MsgFlags = g.NoAuthNoPriv
	case "authnopriv":
		destination.MsgFlags = g.AuthNoPriv
	case "authpriv":
		destination.MsgFlags = g.AuthPriv
	default:
		return fmt.Errorf("unsupported or invalid value (%s) for snmpv3:msg_flags", strings.ToLower(params["msg_flags"]))
	}

	var securityParams g.UsmSecurityParameters

	securityParams.AuthoritativeEngineID = params["engine_id"]

	switch strings.ToLower(params["auth_protocol"]) {
	case "noauth":
		securityParams.AuthenticationProtocol = g.NoAuth
	case "sha":
		securityParams.AuthenticationProtocol = g.SHA
	case "md5":
		securityParams.AuthenticationProtocol = g.MD5
	default:
		return fmt.Errorf("invalid value for snmpv3:auth_protocol: %s", params["auth_protocol"])
	}

	securityParams.AuthenticationPassphrase = params["auth_password"]

	switch strings.ToLower(params["privacy_protocol"]) {
	case "nopriv":
		securityParams.PrivacyProtocol = g.NoPriv
	case "aes":
		securityParams.PrivacyProtocol = g.AES
	case "des":
		securityParams.PrivacyProtocol = g.DES
	default:
		return fmt.Errorf("invalid value for snmpv3:privacy_protocol: %s", params["privacy_protocol"])
	}

	securityParams.PrivacyPassphrase = params["privacy_password"]

	if (destination.MsgFlags&g.AuthPriv) == 1 && securityParams.AuthenticationProtocol < 2 {
		return fmt.Errorf("v3 config error: no auth protocol set when snmpv3:msg_flags specifies an Auth mode")
	}
	if destination.MsgFlags == g.AuthPriv && securityParams.AuthenticationProtocol < 2 {
		return fmt.Errorf("v3 config error: no privacy protocol mode set when snmpv3:msg_flags specifies an AuthPriv mode")
	}

	destination.SecurityParameters = &securityParams

	return nil
}

func (a trapForwarder) ProcessTrap(trap *pluginMeta.Trap) error {
	// Translate to v1 if needed
	if trap.SnmpVersion > g.Version1 {
		err := pluginMeta.TranslateToV1(trap)
		if err != nil {
			return err
		}
	}

	a.trapex_log.Info().Str("plugin", pluginName).Msg("Processing trap")
	_, err := a.destination.SendTrap(trap.Data)
	return err
}

func (p trapForwarder) SigUsr1() error {
	return nil
}

func (p trapForwarder) SigUsr2() error {
	return nil
}

func (a trapForwarder) Close() error {
	a.destination.Conn.Close()
	return nil
}

// Exported symbol which supports filter.go's FilterAction type
var ActionPlugin trapForwarder
