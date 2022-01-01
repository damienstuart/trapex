// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"plugin"
	"regexp"
	"strconv"
	"strings"
	"time"

	g "github.com/gosnmp/gosnmp"
	"github.com/natefinch/lumberjack"
	"github.com/rs/zerolog"

	"github.com/damienstuart/trapex/actions"
)

// Filter action plugin interface
type FilterPlugin interface {
	Configure(logger zerolog.Logger, actionArg string, pluginConfig *plugin_interface.PluginsConfig) error
	ProcessTrap(trap *plugin_interface.Trap) error
	SigUsr1() error
	SigUsr2() error
}

//var plugins []FilterPlugin
func loadFilterActions(newConfig *TrapexConfig) error {
 var plugin_name string = "noop"
		trapex_logger.Info().Str("filter_plugin", plugin_name).Msg("Initializing plugin")
		filter_plugin, err := loadFilterPlugin(plugin_name)
		if err == nil {
			filter_plugin.Configure(trapex_logger, "", &newConfig.FilterPluginsConfig)
		}
	return nil
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

// trapForwarder is an instance of a forward destination.
//
type trapForwarder struct {
	destination *g.GoSNMP
}

// trapLogger is an instance of a trap logfile destination.
//
type trapLogger struct {
	logFile   string
	fd        *os.File
	logHandle *log.Logger
	isBroken  bool
}

// trapCsvLogger is an instance of a trap CSV logfile destination.
//
type trapCsvLogger struct {
	logFile   string
	fd        *os.File
	logger    *lumberjack.Logger
	logHandle *log.Logger
	isBroken  bool
}

// Initialize a trapForwarder instance.
//
func (a *trapForwarder) initAction(dest string) error {
	s := strings.Split(dest, ":")
	port, err := strconv.Atoi(s[1])
	if err != nil {
		panic("Invalid destination port: " + s[1])
	}
	a.destination = &g.GoSNMP{
		Target:             s[0],
		Port:               uint16(port),
		Transport:          "udp",
		Community:          "",
		Version:            g.Version1,
		Timeout:            time.Duration(2) * time.Second,
		Retries:            3,
		ExponentialTimeout: true,
		MaxOids:            g.MaxOids,
	}
	err = a.destination.Connect()
	if err != nil {
		return (err)
	}
	trapex_logger.Info().Str("target", s[0]).Str("port", s[1]).Msg("Added trap destination")
	return nil
}

// Hook for sending a trap to the destination defined for this trapForwarder
// instance.
//
func (a trapForwarder) processTrap(trap *plugin_interface.Trap) error {
	_, err := a.destination.SendTrap(trap.Data)
	return err
}

// Close the trapForwarder connection
//
func (a trapForwarder) close() {
	a.destination.Conn.Close()
}

// Initialize a trapLogger instance.
//
func (a *trapLogger) initAction(logfile string, teConf *TrapexConfig) error {
	fd, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	a.fd = fd
	a.logFile = logfile
	a.logHandle = log.New(fd, "", 0)
	a.logHandle.SetOutput(makeLogger(logfile, teConf))
	trapex_logger.Info().Str("logfile", logfile).Msg("Added log destination")
	return nil
}

// Hook for logging a trap for this instance of a log action.
//
func (a *trapLogger) processTrap(trap *plugin_interface.Trap) {
	logTrap(trap, a.logHandle)
}

// Close a trap logger handle
//
func (a *trapLogger) close() {
	a.fd.Close()
}

// Initialize a trapCsvLogger instance.
//
func (a *trapCsvLogger) initAction(logfile string, teConf *TrapexConfig) error {
	fd, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	a.fd = fd
	a.logFile = logfile
	a.logHandle = log.New(fd, "", 0)
	a.logger = makeCsvLogger(logfile, teConf)
	a.logHandle.SetOutput(a.logger)
	trapex_logger.Info().Str("logfile", logfile).Msg("Added CSV log destination")
	return nil
}

// Hook for logging a trap for this instance of a log action.
//
func (a *trapCsvLogger) processTrap(trap *plugin_interface.Trap) {
	logCsvTrap(trap, a.logHandle)
}

// Get this logger's file name
func (a *trapCsvLogger) logfileName() string {
	return a.logFile
}

// Force a CSV log rotation
func (a *trapCsvLogger) rotateLog() {
	a.logger.Rotate()
}

// Close a trap logger handle
//
func (a *trapCsvLogger) close() {
	a.fd.Close()
}

// isFilterMatch checks trap data against a trapexFilter and returns a boolean
// to indicate whether or not the trap data matches the filter criteria.
//
func (f *TrapexFilter) isFilterMatch(sgt *plugin_interface.Trap) bool {
	// Assume true - until one of the filter items does not match
	trap := &(sgt.Data)
	for _, fo := range f.FilterItems {
		fval := fo.FilterValue
		switch fo.FilterItem {
		case version:
			if fval != sgt.TrapVer {
				return false
			}
		case srcIP:
			if fo.FilterType == parseTypeString && fval.(string) != sgt.SrcIP.String() {
				return false
			} else if fo.FilterType == parseTypeCIDR && !fval.(*network).contains(sgt.SrcIP) {
				return false
			} else if fo.FilterType == parseTypeRegex && !fval.(*regexp.Regexp).MatchString(sgt.SrcIP.String()) {
				return false
			} else if fo.FilterType == parseTypeIPSet {
				_, ok := teConfig.IpSets[fval.(string)][sgt.SrcIP.String()]
				if ok != true {
					return false
				}
			}
		case agentAddr:
			if fo.FilterType == parseTypeString && fval.(string) != trap.AgentAddress {
				return false
			} else if fo.FilterType == parseTypeCIDR && !fval.(*network).contains(net.ParseIP(trap.AgentAddress)) {
				return false
			} else if fo.FilterType == parseTypeRegex && !fval.(*regexp.Regexp).MatchString(trap.AgentAddress) {
				return false
			} else if fo.FilterType == parseTypeIPSet {
				_, ok := teConfig.IpSets[fval.(string)][trap.AgentAddress]
				if ok != true {
					return false
				}
			}
		case enterprise:
			if fo.FilterType == parseTypeRegex && !fval.(*regexp.Regexp).MatchString(strings.TrimLeft(trap.Enterprise, ".")) {
				return false
			} else if fo.FilterType == parseTypeString && fval.(string) != strings.TrimLeft(trap.Enterprise, ".") {
				return false
			}
		case genericType:
			if fo.FilterType == parseTypeInt && fval.(int) != trap.GenericTrap {
				return false
			}
		case specificType:
			if fo.FilterType == parseTypeInt && fval.(int) != trap.SpecificTrap {
				return false
			}
		}
	}
	return true
}

// processAction handles the execution of the action for the
// trapexFilter instance on the the given trap data.
//
func (f *TrapexFilter) processAction(sgt *plugin_interface.Trap) {
	switch f.ActionType {
	case actionBreak:
		sgt.Dropped = true
		return
	case actionNat:
		if f.ActionArg == "$SRC_IP" {
			sgt.Data.AgentAddress = sgt.SrcIP.String()
		} else {
			sgt.Data.AgentAddress = f.ActionArg
		}
	case actionForward:
		f.Action.(*trapForwarder).processTrap(sgt)
	case actionForwardBreak:
		f.Action.(*trapForwarder).processTrap(sgt)
		sgt.Dropped = true
		return
	case actionLog:
		if !sgt.Dropped {
			f.Action.(*trapLogger).processTrap(sgt)
		}
	case actionLogBreak:
		if !sgt.Dropped {
			f.Action.(*trapLogger).processTrap(sgt)
		}
		sgt.Dropped = true
		return
	case actionCsv:
		if !sgt.Dropped {
			f.Action.(*trapCsvLogger).processTrap(sgt)
		}
	case actionCsvBreak:
		if !sgt.Dropped {
			f.Action.(*trapCsvLogger).processTrap(sgt)
		}
		sgt.Dropped = true
		return
	}
}

