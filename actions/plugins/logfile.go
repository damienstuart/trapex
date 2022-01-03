// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

/*
This plugin logs SNMP trap data to a log file
*/

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	plugin_data "github.com/damienstuart/trapex/actions"
	g "github.com/gosnmp/gosnmp"
	"github.com/natefinch/lumberjack"
	"github.com/rs/zerolog"
)

// trapType is an array of trap Generic Type human-friendly names
// ordered by the type value.
//
var trapType = [...]string{
	"Cold Start",
	"Warm Start",
	"Link Down",
	"Link Up",
	"Authentication Failure",
	"EGP Neighbor Loss",
	"Vendor Specific",
}

type trapLogger struct {
	logFile   string
	fd        *os.File
	logHandle *log.Logger
	isBroken  bool

	trapexLog *zerolog.Logger
}

const plugin_name = "trap logger"

func makeLogger(logfile string, actionArgs map[string]string) *lumberjack.Logger {
	l := lumberjack.Logger{
		Filename: logfile,
		//MaxSize:    actionArgs["size_mb"],
		//MaxBackups: actionArgs["backups_max"],
		//Compress:   actionArgs["compress_after_rotate"],
	}
	return &l
}

func (a *trapLogger) Configure(trapexLog *zerolog.Logger, actionArgs map[string]string) error {
	a.trapexLog = trapexLog
	a.trapexLog.Info().Str("plugin", plugin_name).Msg("Initialization of plugin")

	a.trapexLog.Debug().Str("plugin", plugin_name).Msg("Showing plugin variables")

	a.logFile = actionArgs["filename"]
	fd, err := os.OpenFile(a.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	a.fd = fd
	a.logHandle = log.New(fd, "", 0)
	a.logHandle.SetOutput(makeLogger(a.logFile, actionArgs))
	a.trapexLog.Info().Str("logfile", a.logFile).Msg("Added log destination")
	return nil
}

func (a trapLogger) ProcessTrap(trap *plugin_data.Trap) error {
	a.logHandle.Printf(makeTrapLogEntry(trap))
	return nil
}

func (a trapLogger) SigUsr1() error {
	return nil
}

func (a trapLogger) SigUsr2() error {
	return nil
}

func (a trapLogger) Close() error {
	a.fd.Close()
	return nil
}

// makeTrapLogEntry creates a log entry string for the given trap data.
// Note that this particulare implementation expects to be dealing with
// only v1 traps.
//
func makeTrapLogEntry(sgt *plugin_data.Trap) string {
	var b strings.Builder
	var genTrapType string
	trap := sgt.Data

	if trap.GenericTrap >= 0 && trap.GenericTrap <= 6 {
		genTrapType = trapType[trap.GenericTrap]
	} else {
		genTrapType = strconv.Itoa(trap.GenericTrap)
	}
	b.WriteString(fmt.Sprintf("\nTrap: %v", sgt.TrapNumber))
	if sgt.Translated == true {
		b.WriteString(fmt.Sprintf(" (translated from v%s)", sgt.TrapVer.String()))
	}
	b.WriteString(fmt.Sprintf("\n\t%s\n", time.Now().Format(time.ANSIC)))
	b.WriteString(fmt.Sprintf("\tSrc IP: %s\n", sgt.SrcIP))
	b.WriteString(fmt.Sprintf("\tAgent: %s\n", trap.AgentAddress))
	b.WriteString(fmt.Sprintf("\tTrap Type: %s\n", genTrapType))
	b.WriteString(fmt.Sprintf("\tSpecific Type: %v\n", trap.SpecificTrap))
	b.WriteString(fmt.Sprintf("\tEnterprise: %s\n", strings.Trim(trap.Enterprise, ".")))
	b.WriteString(fmt.Sprintf("\tTimestamp: %v\n", trap.Timestamp))

	replacer := strings.NewReplacer("\n", " - ", "%", "%%")

	// Process the Varbinds for this trap.
	for _, v := range trap.Variables {
		vbName := strings.Trim(v.Name, ".")
		switch v.Type {
		case g.OctetString:
			var nonASCII bool
			val := v.Value.([]byte)
			if len(val) > 0 {
				for i := 0; i < len(val); i++ {
					if (val[i] < 32 || val[i] > 127) && val[i] != 9 && val[i] != 10 {
						nonASCII = true
						break
					}
				}
			}
			// Strings with non-printable/non-ascii characters will be dumped
			// as a hex string. Otherwise, just as a plain string.
			if nonASCII {
				b.WriteString(fmt.Sprintf("\tObject:%s Value:%v\n", vbName, replacer.Replace(hex.EncodeToString(val))))
			} else {
				b.WriteString(fmt.Sprintf("\tObject:%s Value:%s\n", vbName, replacer.Replace(string(val))))
			}
		default:
			b.WriteString(fmt.Sprintf("\tObject:%s Value:%v\n", vbName, v.Value))
		}
	}
	return b.String()
}

// Exported symbol which supports filter.go's FilterAction type
var FilterPlugin trapLogger
