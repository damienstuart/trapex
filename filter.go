// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"net"
	"regexp"
	"strings"

	pluginMeta "github.com/damienstuart/trapex/txPlugins"
	pluginLoader "github.com/damienstuart/trapex/txPlugins/interfaces"
)

// Filter types
const (
	parseTypeAny      int = iota // Match anything (wildcard)
	parseTypeString              // Direct String comparison
	parseTypeInt                 // Direct Integer comparison
	parseTypeRegex               // Regular Expression
	parseTypeCIDR                // CIDR IP/Netmask
	parseTypeIPSet               // A set of IP addresses
	parseTypeIntRange            // Integer range x:y or x,y,z
)

// Filter object items
const (
	filterByVersion int = iota
	filterBySrcIP
	filterByAgentAddr
	filterByGenericType
	filterBySpecificType
	filterByOid
)

// Supported action types
const (
	actionBreak int = iota
	actionNat
	actionPlugin
)

// isFilterMatch checks trap data against a trapexFilter and returns a boolean
// to indicate whether or not the trap data matches the filter criteria.
//
func (f *trapexFilter) isFilterMatch(sgt *pluginMeta.Trap) bool {
	if len(f.matchers) == 0 {
		return true
	}
	// Assume true - until one of the filter items does not match
	trap := &(sgt.Data)
	for _, fo := range f.matchers {
		fval := fo.filterValue
		switch fo.filterItem {
		case filterByVersion:
			if fval != sgt.SnmpVersion {
				return false
			}
		case filterBySrcIP:
			if fo.filterType == parseTypeString && fval.(string) != sgt.SrcIP.String() {
				return false
			} else if fo.filterType == parseTypeCIDR && !fval.(*network).contains(sgt.SrcIP) {
				return false
			} else if fo.filterType == parseTypeRegex && !fval.(*regexp.Regexp).MatchString(sgt.SrcIP.String()) {
				return false
			} else if fo.filterType == parseTypeIPSet {
				_, ok := teConfig.IpSets[fval.(string)][sgt.SrcIP.String()]
				if ok != true {
					return false
				}
			}
		case filterByAgentAddr:
			if fo.filterType == parseTypeString && fval.(string) != trap.AgentAddress {
				return false
			} else if fo.filterType == parseTypeCIDR && !fval.(*network).contains(net.ParseIP(trap.AgentAddress)) {
				return false
			} else if fo.filterType == parseTypeRegex && !fval.(*regexp.Regexp).MatchString(trap.AgentAddress) {
				return false
			} else if fo.filterType == parseTypeIPSet {
				_, ok := teConfig.IpSets[fval.(string)][trap.AgentAddress]
				if ok != true {
					return false
				}
			}
		case filterByOid:
			if fo.filterType == parseTypeRegex && !fval.(*regexp.Regexp).MatchString(strings.TrimLeft(trap.Enterprise, ".")) {
				return false
			} else if fo.filterType == parseTypeString && fval.(string) != strings.TrimLeft(trap.Enterprise, ".") {
				return false
			}
		case filterByGenericType:
			if fo.filterType == parseTypeInt && fval.(int) != trap.GenericTrap {
				return false
			}
		case filterBySpecificType:
			if fo.filterType == parseTypeInt && fval.(int) != trap.SpecificTrap {
				return false
			}
		}
	}
	return true
}

// processAction handles the execution of the action for the
// trapexFilter instance on the the given trap data.
//
func (f *trapexFilter) processAction(trap *pluginMeta.Trap) error {
	var err error

	switch f.actionType {
	case actionBreak:
		trap.Dropped = true
	case actionNat:
		if f.ActionArg == "$SRC_IP" {
			trap.Data.AgentAddress = trap.SrcIP.String()
		} else {
			trap.Data.AgentAddress = f.ActionArg
		}
	case actionPlugin:
		err = f.plugin.(pluginLoader.ActionPlugin).ProcessTrap(trap)
		if err != nil {
			trapexLog.Err(err).Str("plugin", f.ActionName).Msg("Issue in processing trap by plugin")
		}
	default:
		trapexLog.Warn().Int("action_type", f.actionType).Msg("Unkown action type given to processAction")
	}
	if f.BreakAfter {
		trap.Dropped = true
	}
	return err
}
