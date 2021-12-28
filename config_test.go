// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main


import (
    "testing"
)

func TestGeneralSection(t *testing.T) {
        var testConfig trapexConfig
    loadConfig( "tests/config/general.yml", &testConfig)

    if testConfig.General.Hostname != "trapex_test1" {
        t.Errorf("Hostname is not set correctly: %s", testConfig.General.Hostname)
    }
    if testConfig.General.ListenAddr != "127.0.0.1" {
        t.Errorf("Listen address is not set correctly: %s", testConfig.General.ListenAddr)
    }
    if testConfig.General.ListenPort != "168" {
        t.Errorf("Listen port is not set correctly: %s", testConfig.General.ListenPort)
    }

    if len(testConfig.General.IgnoreVersions) != 2 {
        t.Errorf("Ignore versions is not set correctly: %s", testConfig.General.IgnoreVersions)
    }

    if testConfig.General.PrometheusIp != "127.10.0.1" {
        t.Errorf("Prometheus host is not set correctly: %s", testConfig.General.PrometheusIp)
    }
    if testConfig.General.PrometheusPort != "8080" {
        t.Errorf("Prometheus port is not set correctly: %s", testConfig.General.PrometheusPort)
    }
    if testConfig.General.PrometheusEndpoint != "statistics" {
        t.Errorf("Prometheus endpoint is not set correctly: %s", testConfig.General.PrometheusEndpoint)
    }
}

func TestIgnoreVersions(t *testing.T) {
        var testConfig trapexConfig
        var err error

    loadConfig( "tests/config/ignore_versions_bad.yml", &testConfig)
    err = validateIgnoreVersions(&testConfig)
    if err == nil {
        t.Errorf("general:ignore_versions did not detect invalid version: %s", testConfig.General.IgnoreVersions)
    }

    loadConfig( "tests/config/ignore_versions_multiple.yml", &testConfig)
    err = validateIgnoreVersions(&testConfig)
    if len(testConfig.General.ignoreVersions) != 2 {
        t.Errorf("general:ignore_versions unable to deduplicate versions: %s", testConfig.General.IgnoreVersions)
    }

    loadConfig( "tests/config/ignore_versions_all.yml", &testConfig)
    err = validateIgnoreVersions(&testConfig)
    if err == nil {
        t.Errorf("general:ignore_versions did not detect all versions: %s", testConfig.General.IgnoreVersions)
    }
}

func TestLogging(t *testing.T) {
        var testConfig trapexConfig
    loadConfig( "tests/config/logging.yml" , &testConfig)

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
    loadConfig( "tests/config/snmpv3.yml" , &testConfig)

    if testConfig.V3Params.MsgFlags != "AuthPriv" {
        t.Errorf("username is not set correctly: %s", testConfig.V3Params.MsgFlags)
    }
    if testConfig.V3Params.Username != "myuser" {
        t.Errorf("username is not set correctly: %s", testConfig.V3Params.Username)
    }
    if testConfig.V3Params.AuthProto != "SHA" {
        t.Errorf("auth proto is not set correctly: %s", testConfig.V3Params.AuthProto)
    }
    if testConfig.V3Params.AuthPassword != "v3authPass" {
        t.Errorf("auth password is not set correctly: %s", testConfig.V3Params.AuthPassword)
    }
    if testConfig.V3Params.PrivacyProto != "AES" {
        t.Errorf("Privacy proto is not set correctly: %s", testConfig.V3Params.PrivacyProto)
    }
    if testConfig.V3Params.PrivacyPassword != "v3privPW" {
        t.Errorf("Privacy password is not set correctly: %s", testConfig.V3Params.PrivacyPassword)
    }
}


func TestIpSets(t *testing.T) {
    var testConfig trapexConfig
    loadConfig( "tests/config/ipsets.yml" , &testConfig)

    var numsets = len(testConfig.IpSets)
    if numsets !=  3 {
        t.Errorf("Have different number of expected ip sets (expected 3, got %d)", numsets)
    }

    var err error
    if err = processIpSets(&testConfig); err != nil {
        t.Errorf("%s", err)
    }
    
}


func TestFiltersGood(t *testing.T) {
    var testConfig trapexConfig
    loadConfig( "tests/config/filters.yml" , &testConfig)

    var numfilters = len(testConfig.RawFilters)
    if numfilters !=  11 {
        t.Errorf("filters are missing entries (expected 11): %s", testConfig.RawFilters)
    }

    var err error
    if err = processFilters(&testConfig); err != nil {
        t.Errorf("%s", err)
    }
    numfilters = len(testConfig.filters)
    if numfilters !=  11 {
        t.Errorf("processed filters are missing entries (expected 11): %d", numfilters)
    }
}

func TestFiltersMissingLogDir(t *testing.T) {
    var testConfig trapexConfig
    loadConfig( "tests/config/filters_bad_logfile.yml" , &testConfig)

    var err error
    if err = processFilters(&testConfig); err == nil {
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
