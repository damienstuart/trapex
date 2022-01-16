// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"testing"

	"github.com/rs/zerolog"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

func TestTrapReceiverSection(t *testing.T) {
	var testConfig trapexConfig
	loadConfig("tests/config/general.yml", &testConfig)

	if testConfig.TrapReceiverSettings.Hostname != "trapex_test1" {
		t.Errorf("Hostname is not set correctly: %s", testConfig.TrapReceiverSettings.Hostname)
	}
	if testConfig.TrapReceiverSettings.ListenAddr != "127.0.0.1" {
		t.Errorf("Listen address is not set correctly: %s", testConfig.TrapReceiverSettings.ListenAddr)
	}
	if testConfig.TrapReceiverSettings.ListenPort != "168" {
		t.Errorf("Listen port is not set correctly: %s", testConfig.TrapReceiverSettings.ListenPort)
	}

	if len(testConfig.TrapReceiverSettings.IgnoreVersions_str) != 2 {
		t.Errorf("Ignore versions is not set correctly: %s", testConfig.TrapReceiverSettings.IgnoreVersions_str)
	}
}

func TestIgnoreVersions(t *testing.T) {
	var testConfig trapexConfig
	var err error

	loadConfig("tests/config/ignore_versions_bad.yml", &testConfig)
	err = validateIgnoreVersions(&testConfig)
	if err == nil {
		t.Errorf("general:ignore_versions did not detect invalid version: %s", testConfig.TrapReceiverSettings.IgnoreVersions_str)
	}

	loadConfig("tests/config/ignore_versions_multiple.yml", &testConfig)
	err = validateIgnoreVersions(&testConfig)
	if len(testConfig.TrapReceiverSettings.IgnoreVersions) != 2 {
		t.Errorf("general:ignore_versions unable to deduplicate versions: %s", testConfig.TrapReceiverSettings.IgnoreVersions_str)
	}

	loadConfig("tests/config/ignore_versions_all.yml", &testConfig)
	err = validateIgnoreVersions(&testConfig)
	if err == nil {
		t.Errorf("general:ignore_versions did not detect all versions: %s", testConfig.TrapReceiverSettings.IgnoreVersions_str)
	}
}

func TestLogging(t *testing.T) {
	var testConfig trapexConfig
	var err error
	if err = loadConfig("tests/config/logging.yml", &testConfig); err != nil {
		t.Errorf("Logging configuration broken: %s", err)
	}

	if testConfig.Logging.Level != "info" {
		t.Errorf("Logging level is not set correctly: %s", testConfig.Logging.Level)
	}
	if testConfig.Logging.LogMaxSize != 4096 {
		t.Errorf("logging max file size is not set correctly: %d", testConfig.Logging.LogMaxSize)
	}
	if testConfig.Logging.LogMaxBackups != 10 {
		t.Errorf("logging max backups is not set correctly: %d", testConfig.Logging.LogMaxBackups)
	}
	if testConfig.Logging.LogCompress != true {
		t.Errorf("LogCompress is not set correctly: %t", testConfig.Logging.LogCompress)
	}
}

func TestSnmpv3(t *testing.T) {
	var testConfig trapexConfig
	loadConfig("tests/config/snmpv3.yml", &testConfig)

	if testConfig.TrapReceiverSettings.MsgFlags_str != "AuthPriv" {
		t.Errorf("username is not set correctly: %s", testConfig.TrapReceiverSettings.MsgFlags_str)
	}
	if testConfig.TrapReceiverSettings.Username != "myuser" {
		t.Errorf("username is not set correctly: %s", testConfig.TrapReceiverSettings.Username)
	}
	if testConfig.TrapReceiverSettings.AuthProto_str != "SHA" {
		t.Errorf("auth proto is not set correctly: %s", testConfig.TrapReceiverSettings.AuthProto_str)
	}
	if testConfig.TrapReceiverSettings.AuthPassword != "v3authPass" {
		t.Errorf("auth password is not set correctly: %s", testConfig.TrapReceiverSettings.AuthPassword)
	}
	if testConfig.TrapReceiverSettings.PrivacyProto_str != "AES" {
		t.Errorf("Privacy proto is not set correctly: %s", testConfig.TrapReceiverSettings.PrivacyProto_str)
	}
	if testConfig.TrapReceiverSettings.PrivacyPassword != "v3privPW" {
		t.Errorf("Privacy password is not set correctly: %s", testConfig.TrapReceiverSettings.PrivacyPassword)
	}
}

func TestIpSetsGood(t *testing.T) {
	var testConfig trapexConfig
	loadConfig("tests/config/ipsets.yml", &testConfig)

	var numsets = len(testConfig.IpSets_str)
	if numsets != 3 {
		t.Errorf("Have different number of expected ip sets (expected 3, got %d)", numsets)
	}

	var err error
	if err = addIpSets(&testConfig); err != nil {
		t.Errorf("%s", err)
	}
}

func TestIpSetsBadIps(t *testing.T) {
	var testConfig trapexConfig
	loadConfig("tests/config/ipsets_bad_ips.yml", &testConfig)

	var err error
	if err = addIpSets(&testConfig); err == nil {
		t.Errorf("Unable to detect bad IP entries in IpSets")
	}
}

func TestFiltersGood(t *testing.T) {
	var testConfig trapexConfig
	loadConfig("tests/config/filters.yml", &testConfig)

	var numfilters = len(testConfig.Filters)
	if numfilters != 8 {
		t.Errorf("filters are missing entries (expected 8): %d", numfilters)
	}

	var err error
	if err = addFilters(&testConfig); err != nil {
		t.Errorf("%s", err)
	}
}

func TestFiltersMissingLogDir(t *testing.T) {
	var testConfig trapexConfig
	loadConfig("tests/config/filters_bad_logfile.yml", &testConfig)

	var err error
	if err = addFilters(&testConfig); err == nil {
		t.Errorf("Should have found a bad log directory")
	}
}

/* --- missing IP set checks are commented out in code
func TestFiltersMissingIpSet(t *testing.T) {
    var testConfig trapexConfig
    loadConfig( "tests/config/filters_missing_wipset.yml" , &testConfig)

    var err error
    if err = processFilters(&testConfig); err == nil {
        t.Errorf("Should have detected a missing ipset")
    }
}
*/
