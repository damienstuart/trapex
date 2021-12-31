// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package plugin_interface

/*
 * Externalize data types into a separate file in order to pass data from
 * the main program to plugins
 */

import (
 "net"
         g "github.com/gosnmp/gosnmp"

)

// Trap holds a pointer to a trap and the source IP of the incoming trap.
//
type Trap struct {
        TrapNumber uint64
        Data       g.SnmpTrap
        TrapVer    g.SnmpVersion
        SrcIP      net.IP
        Translated bool
        Dropped    bool
}

