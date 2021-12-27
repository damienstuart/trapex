// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main


import (
    "testing"
        g "github.com/gosnmp/gosnmp"
)

func TestGenearl(t *testing.T) {
        var testConfig trapexConfig
    loadConfig( "tests/config/general.yml", &testConfig)

    if testConfig.General.ListenAddr != "127.0.0.1" {
        t.Errorf("Default host is not set correctly: %s", testConfig.General.ListenAddr)
    }
    if testConfig.General.ListenPort != "168" {
        t.Errorf("Default port is not set correctly: %s", testConfig.General.ListenPort)
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


func TestLogging(t *testing.T) {
        var testConfig trapexConfig
    loadConfig( "tests/config/logging.yml" , &testConfig)

    if testConfig.Logging.Level != "info" {
        t.Errorf("Default logging level is not set correctly: %s", testConfig.Logging.Level)
    }
    if testConfig.Logging.LogMaxSize != 4096 {
        t.Errorf("Default logging max file size is not set correctly: %d", testConfig.Logging.LogMaxSize)
    }
    if testConfig.Logging.LogMaxBackups != 10 {
        t.Errorf("Default logging max backups is not set correctly: %d", testConfig.Logging.LogMaxBackups)
    }
    if testConfig.Logging.LogCompress != true {
        t.Errorf("Default LogCompress is not set correctly: %t", testConfig.Logging.LogCompress)
    }
}

func TestSnmpv3(t *testing.T) {
        var testConfig trapexConfig
    loadConfig( "tests/config/snmpv3.yml" , &testConfig)

    if testConfig.V3Params.Username != "myuser" {
        t.Errorf("username is not set correctly: %s", testConfig.V3Params.Username)
    }
    if testConfig.V3Params.AuthProto != g.SHA {
        t.Errorf("auth proto is not set correctly: %s", testConfig.V3Params.AuthProto)
    }
    if testConfig.V3Params.AuthPassword != "v3authPass" {
        t.Errorf("auth password is not set correctly: %s", testConfig.V3Params.AuthPassword)
    }
    if testConfig.V3Params.PrivacyProto != g.AES {
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
    if numsets !=  1 {
        t.Errorf("ip sets are missing entries (expected 1): %s", testConfig.IpSets)
    }
}


func TestFilters(t *testing.T) {
    var testConfig trapexConfig
    loadConfig( "tests/config/filters.yml" , &testConfig)

    var numfilters = len(testConfig.RawFilters)
    if numfilters !=  3 {
        t.Errorf("filters are missing entries (expected 3): %s", testConfig.RawFilters)
    }
}

