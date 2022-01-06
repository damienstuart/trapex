// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"encoding/json"
	"fmt"
	"net"

	plugin_data "github.com/damienstuart/trapex/txPlugins"
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

// panicOnError check an error pointer and panics if it is not nil.
//
func panicOnError(e error) {
	if e != nil {
		panic(e)
	}
}

// makeTrapLogEntry creates a log entry string for the given trap data.
// Note that this particulare implementation expects to be dealing with
// only v1 traps.
//
func makeTrapLogEntry(trap *plugin_data.Trap) string {
	trapMap := trap.V1Trap2Map()
	jsonBytes, _ := json.Marshal(trapMap)
	return string(jsonBytes[:])
}

// secondsToDuration converts the given number of seconds into a more
// human-readable formatted string.
//
func secondsToDuration(s uint) string {
	var d uint
	var h uint
	var m uint
	if s >= 86400 {
		d = s / 86400
		s %= 86400
	}
	if s >= 3600 {
		h = s / 3600
		s %= 3600
	}
	if s >= 60 {
		m = s / 60
		s %= 60
	}
	return fmt.Sprintf("%vd-%vh-%vm-%vs", d, h, m, s)
}

// isIgnoredVersion returns a boolean indicating whether or not the given
// SnmpVersion value is being ignored.
//
func isIgnoredVersion(ver g.SnmpVersion) bool {
	for _, v := range teConfig.General.IgnoreVersions {
		if ver == v {
			return true
		}
	}
	return false
}
