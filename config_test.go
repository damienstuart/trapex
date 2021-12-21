// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main


import (
    "testing"
)

// TestGoodConf
func TestGoodConf(t *testing.T) {
    teCmdLine.configFile = "tests/config/good_ehealth.conf"

    err := getConfig()
    if err != nil {
        t.Errorf("Ran into issues running known good conf: %s", err)
    }

    if teConfig.listenAddr != "0.0.0.0" {
        t.Errorf("Default host is not set correctly: %s", teConfig.listenAddr)
    }
    if teConfig.listenPort != "162" {
        t.Errorf("Default port is not set correctly: %s", teConfig.listenPort)
    }
}
