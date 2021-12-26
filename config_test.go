// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main


import (
    "testing"
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

