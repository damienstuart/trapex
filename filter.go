// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"errors"
	"fmt"
	"net"
	"plugin"
	"regexp"
	"strings"

	"github.com/rs/zerolog"

	"github.com/damienstuart/trapex/actions"
)

// Filter action plugin interface
type FilterPlugin interface {
	Configure(logger zerolog.Logger, actionArg string, pluginConfig *plugin_interface.PluginsConfig) error
	ProcessTrap(trap *plugin_interface.Trap) error
	SigUsr1() error
	SigUsr2() error
	Close() error
}

func loadFilterPlugin(plugin_name string) (FilterPlugin, error) {
	var plugin_filename = "actions/plugins/" + plugin_name + ".so"

	plug, err := plugin.Open(plugin_filename)
	if err != nil {
		return nil, err
	}

	// Load the class from the plugin
	symAction, err := plug.Lookup("FilterPlugin")
	if err != nil {
		return nil, err
	}

	var initializer FilterPlugin
	// Instantiate the class from the plugin
	initializer, ok := symAction.(FilterPlugin)
	if !ok {
		symbolType := fmt.Sprintf("%T", symAction)
		trapex_logger.Error().Str("filter_plugin", plugin_name).Str("data type", symbolType).Msg("Unable to load plugin")
		return nil, errors.New("Unexpected type from plugin")
	}

	return initializer, nil
}

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
	version int = iota
	srcIP
	agentAddr
	genericType
	specificType
	enterprise
)

// Supported action types
const (
	actionBreak int = iota
	actionNat
	actionForward
	actionForwardBreak
	actionLog
	actionLogBreak
	actionCsv
	actionCsvBreak
)

// filterObj represents one of the filterable items in a filter line from
// the config file (i.e. Src IP, AgentAddress, GenericType, SpecificType,
// and Enterprise OID).
//
type filterObj struct {
	filterItem  int
	filterType  int
	filterValue interface{} // string, *regex.Regexp, *network, int
}

// trapexFilter holds the filter data and action for a specfic
// filter line from the config file.
type trapexFilter struct {
	filterItems []filterObj
	matchAll    bool
	//action      interface{}
	action      FilterPlugin
	actionType  int
	actionArg   string
}

// isFilterMatch checks trap data against a trapexFilter and returns a boolean
// to indicate whether or not the trap data matches the filter criteria.
//
func (f *trapexFilter) isFilterMatch(sgt *plugin_interface.Trap) bool {
	// Assume true - until one of the filter items does not match
	trap := &(sgt.Data)
	for _, fo := range f.filterItems {
		fval := fo.filterValue
		switch fo.filterItem {
		case version:
			if fval != sgt.TrapVer {
				return false
			}
		case srcIP:
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
		case agentAddr:
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
		case enterprise:
			if fo.filterType == parseTypeRegex && !fval.(*regexp.Regexp).MatchString(strings.TrimLeft(trap.Enterprise, ".")) {
				return false
			} else if fo.filterType == parseTypeString && fval.(string) != strings.TrimLeft(trap.Enterprise, ".") {
				return false
			}
		case genericType:
			if fo.filterType == parseTypeInt && fval.(int) != trap.GenericTrap {
				return false
			}
		case specificType:
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
func (f *trapexFilter) processAction(sgt *plugin_interface.Trap) {
	switch f.actionType {
	case actionBreak:
		sgt.Dropped = true
		return
	case actionNat:
		if f.actionArg == "$SRC_IP" {
			sgt.Data.AgentAddress = sgt.SrcIP.String()
		} else {
			sgt.Data.AgentAddress = f.actionArg
		}
	case actionForward:
		f.action.(FilterPlugin).ProcessTrap(sgt)
	case actionForwardBreak:
		f.action.(FilterPlugin).ProcessTrap(sgt)
		sgt.Dropped = true
		return
	case actionLog:
		if !sgt.Dropped {
			f.action.(FilterPlugin).ProcessTrap(sgt)
		}
	case actionLogBreak:
		if !sgt.Dropped {
			f.action.(FilterPlugin).ProcessTrap(sgt)
		}
		sgt.Dropped = true
		return
	case actionCsv:
		if !sgt.Dropped {
			f.action.(FilterPlugin).ProcessTrap(sgt)
		}
	case actionCsvBreak:
		if !sgt.Dropped {
			f.action.(FilterPlugin).ProcessTrap(sgt)
		}
		sgt.Dropped = true
		return
	}
}
