// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	g "github.com/gosnmp/gosnmp"
	"github.com/natefinch/lumberjack"
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
	action      interface{}
	actionType  int
	actionArg   string
}

// trapForwarder is an instance of a forward destination.
//
type trapForwarder struct {
	destination *g.GoSNMP
}

// trapLogger is an instace of a trap logfile destination.
//
type trapLogger struct {
	logFile   string
	fd        *os.File
	logHandle *log.Logger
	isBroken  bool
}

// trapCsvLogger is an instace of a trap CSV logfile destination.
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
	fmt.Printf(" -Added trap destination: %s, port %s\n", s[0], s[1])
	return nil
}

// Hook for sending a trap to the destination defined for this trapForwarder
// instance.
//
func (a trapForwarder) processTrap(trap *sgTrap) error {
	_, err := a.destination.SendTrap(trap.data)
	return err
}

// Close the trapForwarder connection
//
func (a trapForwarder) close() {
	a.destination.Conn.Close()
}

// Initialize a trapLogger instance.
//
func (a *trapLogger) initAction(logfile string, teConf *trapexConfig) error {
	fd, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	a.fd = fd
	a.logFile = logfile
	a.logHandle = log.New(fd, "", 0)
	a.logHandle.SetOutput(makeLogger(logfile, teConf))
	fmt.Printf(" -Added log destination: %s\n", logfile)
	return nil
}

// Hook for logging a trap for this instance of a log action.
//
func (a *trapLogger) processTrap(trap *sgTrap) {
	logTrap(trap, a.logHandle)
}

// Close a trap logger handle
//
func (a *trapLogger) close() {
	a.fd.Close()
}

// Initialize a trapCsvLogger instance.
//
func (a *trapCsvLogger) initAction(logfile string, teConf *trapexConfig) error {
	fd, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	a.fd = fd
	a.logFile = logfile
	a.logHandle = log.New(fd, "", 0)
	a.logger = makeCsvLogger(logfile, teConf)
	a.logHandle.SetOutput(a.logger)
	fmt.Printf(" -Added CSV log destination: %s\n", logfile)
	return nil
}

// Hook for logging a trap for this instance of a log action.
//
func (a *trapCsvLogger) processTrap(trap *sgTrap) {
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
func (f *trapexFilter) isFilterMatch(sgt *sgTrap) bool {
	// Assume true - until one of the filter items does not match
	trap := &(sgt.data)
	for _, fo := range f.filterItems {
		fval := fo.filterValue
		switch fo.filterItem {
		case version:
			if fval != sgt.trapVer {
				return false
			}
		case srcIP:
			if fo.filterType == parseTypeString && fval.(string) != sgt.srcIP.String() {
				return false
			} else if fo.filterType == parseTypeCIDR && !fval.(*network).contains(sgt.srcIP) {
				return false
			} else if fo.filterType == parseTypeRegex && !fval.(*regexp.Regexp).MatchString(sgt.srcIP.String()) {
				return false
			} else if fo.filterType == parseTypeIPSet {
				_, ok := teConfig.ipSets[fval.(string)][sgt.srcIP.String()]
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
				_, ok := teConfig.ipSets[fval.(string)][trap.AgentAddress]
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
func (f *trapexFilter) processAction(sgt *sgTrap) {
	switch f.actionType {
	case actionBreak:
		sgt.dropped = true
		return
	case actionNat:
		if f.actionArg == "$SRC_IP" {
			sgt.data.AgentAddress = sgt.srcIP.String()
		} else {
			sgt.data.AgentAddress = f.actionArg
		}
	case actionForward:
		f.action.(*trapForwarder).processTrap(sgt)
	case actionForwardBreak:
		f.action.(*trapForwarder).processTrap(sgt)
		sgt.dropped = true
		return
	case actionLog:
		if !sgt.dropped {
			f.action.(*trapLogger).processTrap(sgt)
		}
	case actionLogBreak:
		if !sgt.dropped {
			f.action.(*trapLogger).processTrap(sgt)
		}
		sgt.dropped = true
		return
	case actionCsv:
		if !sgt.dropped {
			f.action.(*trapCsvLogger).processTrap(sgt)
		}
	case actionCsvBreak:
		if !sgt.dropped {
			f.action.(*trapCsvLogger).processTrap(sgt)
		}
		sgt.dropped = true
		return
	}
}
