// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"encoding/json"
	"net"

	pluginMeta "github.com/damienstuart/trapex/txPlugins"
	g "github.com/gosnmp/gosnmp"
)

// network stuct holds the data parsed from a CIDR representation of a
// subnet.
//
type network struct {
	ip  net.IP
	net *net.IPNet
}

// newNetwork initializes the network stuct based on the given CIDR
// formatted subnet.
//
func newNetwork(cidr string) (*network, error) {
	nm := network{}
	i, n, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	nm.ip = i
	nm.net = n
	return &nm, nil
}

// Returns true if the given IP falls within the subnet contained
// in the network object.
//
func (n *network) contains(ip net.IP) bool {
	return n.net.Contains(ip)
}

// makeTrapLogEntry creates a log entry string for the given trap data.
// Note that this particulare implementation expects to be dealing with
// only v1 traps.
//
func makeTrapLogEntry(trap *pluginMeta.Trap) string {
	trapMap := trap.Trap2Map()
	jsonBytes, _ := json.Marshal(trapMap)
	return string(jsonBytes[:])
}

// isIgnoredVersion returns a boolean indicating whether or not the given
// SnmpVersion value is being ignored.
//
func isIgnoredVersion(ver g.SnmpVersion) bool {
	for _, v := range teConfig.TrapReceiverSettings.IgnoreVersions {
		if ver == v {
			return true
		}
	}
	return false
}
