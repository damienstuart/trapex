// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

/*
This plugin performs no useful action, but can be used for unit testing or performance
testing purposes.
 */

import (
       "net"
        g "github.com/gosnmp/gosnmp"

        "github.com/rs/zerolog"
)

// sgTrap holds a pointer to a trap and the source IP of
// the incoming trap.
//
type sgTrap struct {
        trapNumber uint64
        data       g.SnmpTrap
        trapVer    g.SnmpVersion
        srcIP      net.IP
        translated bool
        dropped    bool
}


type noopFilter string
const plugin_name = "no op"


func (p noopFilter) Init(logger zerolog.Logger) error {
        logger.Info().Str("plugin", plugin_name).Msg("Initialization of plugin")

   return nil
}

/*
  .................
Plugin issue: how to pass in data types to a plugin?
I think that Golang might be comparing a hash of some sort, rather than
a structure-based method, so that the locally declared sgTrap is different
than the trapex.go sgTrap type

Ugh.  This sucks

*/
func (p noopFilter) ProcessTrap() error {
//func (p noopFilter) ProcessTrap(trap *sgTrap) error {
        //logger.Info().Str("plugin", plugin_name).Msg("Noop processing trap")
   return nil
}

func (p noopFilter) SigUsr1() error {
   return nil
}

func (p noopFilter) SigUsr2() error {
   return nil
}


// Exported symbol which supports filter.go's FilterAction type
var FilterPlugin noopFilter

