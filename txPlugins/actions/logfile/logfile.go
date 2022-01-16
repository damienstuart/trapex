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

	pluginMeta "github.com/damienstuart/trapex/txPlugins"
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

const pluginName = "trap logger"

func makeLogger(logfile string, actionArgs map[string]string) *lumberjack.Logger {
	var ok bool

	l := lumberjack.Logger{Filename: logfile}
	var value string
	value, ok = actionArgs["size_mb"]
	if ok {
		converted, err := strconv.Atoi(value)
		if err != nil {
			l.MaxSize = converted
		}
	}
	value, ok = actionArgs["backups_max"]
	if ok {
		converted, err := strconv.Atoi(value)
		if err != nil {
			l.MaxBackups = converted
		}
	}
	value, ok = actionArgs["compress_after_rotate"]
	if ok {
		converted, err := strconv.ParseBool(value)
		if err != nil {
			l.Compress = converted
		}
	}
	return &l
}

func validateArguments(actionArgs map[string]string) error {
	validArgs := map[string]bool{"filename": true, "size_mb": true, "backups_max": true, "compress_after_rotate": true}

	for key, _ := range actionArgs {
		if _, ok := validArgs[key]; !ok {
			return fmt.Errorf("Unrecognized option to %s plugin: %s", pluginName, key)
		}
	}
	return nil
}

func (a *trapLogger) Configure(trapexLog *zerolog.Logger, actionArgs map[string]string) error {
	var ok bool
	a.trapexLog = trapexLog
	a.trapexLog.Info().Str("plugin", pluginName).Msg("Initialization of plugin")

	if err := validateArguments(actionArgs); err != nil {
		return err
	}

	a.logFile, ok = actionArgs["filename"]
	if !ok {
		return fmt.Errorf("Missing the required 'filename' argument to the logfile action")
	}
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

func (a trapLogger) ProcessTrap(trap *pluginMeta.Trap) error {
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
func makeTrapLogEntry(sgt *pluginMeta.Trap) string {
	var b strings.Builder
	var genTrapType string
	trap := sgt.Data

	if trap.GenericTrap >= 0 && trap.GenericTrap <= 6 {
		genTrapType = trapType[trap.GenericTrap]
	} else {
		genTrapType = strconv.Itoa(trap.GenericTrap)
	}
	b.WriteString(fmt.Sprintf("\nTrap: %v", sgt.TrapNumber))
	b.WriteString(fmt.Sprintf("\nSNMP Version%s", sgt.SnmpVersion.String()))
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
var ActionPlugin trapLogger
