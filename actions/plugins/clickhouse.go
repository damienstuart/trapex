// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

/*
 * Dump data out in Clickhouse CSV data format, for adding to a Clickhouse database
 */

import (
	"encoding/hex"
	"fmt"
	g "github.com/gosnmp/gosnmp"
	"log"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/damienstuart/trapex/actions"
	"github.com/natefinch/lumberjack"
)

const plugin_name = "Clickhouse"

type ClickhouseExport struct {
	logFile   string
	fd        *os.File
	logger    *lumberjack.Logger
	logHandle *log.Logger
	isBroken  bool

	trapex_log zerolog.Logger
}

// makeCsvLogger initializes and returns a lumberjack.Logger (logger with
// built-in log rotation management).
//
func makeCsvLogger(logfile string) *lumberjack.Logger {
	l := lumberjack.Logger{
		Filename: logfile,
	}
	return &l
}

func (a ClickhouseExport) Configure(logger zerolog.Logger, actionArg string, pluginConfig *plugin_interface.PluginsConfig) error {
	a.trapex_log = logger
	a.trapex_log.Info().Str("plugin", plugin_name).Msg("Added CSV log destination")

	a.logFile = actionArg
	fd, err := os.OpenFile(a.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	a.fd = fd
	a.logHandle = log.New(fd, "", 0)
	a.logger = makeCsvLogger(a.logFile)
	a.logHandle.SetOutput(a.logger)
	a.trapex_log.Info().Str("logfile", a.logFile).Msg("Added CSV log destination")

	return nil
}

func (a ClickhouseExport) ProcessTrap(trap *plugin_interface.Trap) error {
	logCsvTrap(trap, a.logHandle)
	return nil
}

func (a ClickhouseExport) SigUsr1() error {
	fmt.Println("SigUsr1")
	return nil
}

func (a ClickhouseExport) Close() error {
	fmt.Println("Close")
	a.fd.Close()
	return nil
}

func (a ClickhouseExport) SigUsr2() error {
	fmt.Println("SigUsr2")
	a.logger.Rotate()

	a.trapex_log.Info().Str("logfile", a.logFile).Msg("Rotated CSV file")
	return nil
}

// logCsvTrap takes care of logging the given trap to the given ClickhouseExport
// destination.
//
func logCsvTrap(trap *plugin_interface.Trap, l *log.Logger) {
	l.Printf(makeTrapLogCsvEntry(trap))
}

// makeTrapLogEntry creates a log entry string for the given trap data.
// Note that this particular implementation expects to be dealing with
// only v1 traps.
//
func makeTrapLogCsvEntry(trap *plugin_interface.Trap) string {
	var csv [11]string
	raw_trap := trap.Data

	/* Fields in order:
	   TrapDate,
	   TrapTimestamp,
	   TrapHost,
	   TrapNumber,
	   TrapSourceIP,
	   TrapAgentAddress,
	   TrapGenericType,
	   TrapSpecificType,
	   TrapEnterpriseOID,
	   TrapVarBinds.ObjID (array)
	   TrapVarBinds.Value (array)
	*/

	var ts = time.Now().Format(time.RFC3339)

	csv[0] = fmt.Sprintf("%v", ts[:10])
	csv[1] = fmt.Sprintf("%v %v", ts[:10], ts[11:19])
	// FIXME: global teConfig object not visible in plugin space
	//csv[2] = fmt.Sprintf("\"%v\"", teConfig.General.Hostname)
	csv[2] = fmt.Sprintf("\"%v\"", "hostname")
	//csv[3] = fmt.Sprintf("%v", stats.TrapCount)
	// FIXME: global stats object not visible in plugin space
	csv[3] = fmt.Sprintf("%v", 1)
	csv[4] = fmt.Sprintf("\"%v\"", trap.SrcIP)
	csv[5] = fmt.Sprintf("\"%v\"", raw_trap.AgentAddress)
	csv[6] = fmt.Sprintf("%v", raw_trap.GenericTrap)
	csv[7] = fmt.Sprintf("%v", raw_trap.SpecificTrap)
	csv[8] = fmt.Sprintf("\"%v\"", strings.Trim(raw_trap.Enterprise, "."))

	var vbObj []string
	var vbVal []string

	// For escaping quotes and backslashes and replace newlines with a space
	replacer := strings.NewReplacer("\"", "\"\"", "'", "''", "\\", "\\\\", "\n", " - ", "%", "%%")

	// Process the Varbinds for this raw_trap.
	// Varbinds are split to separate arrays - one for the ObjectIDs,
	// and the other for Values
	for _, v := range raw_trap.Variables {
		// Get the OID
		vbObj = append(vbObj, strings.Trim(v.Name, "."))
		// Parse the value
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
				vbVal = append(vbVal, fmt.Sprintf("%v", replacer.Replace(hex.EncodeToString(val))))
			} else {
				vbVal = append(vbVal, replacer.Replace(fmt.Sprintf("%v", string(val))))
			}
		default:
			vbVal = append(vbVal, replacer.Replace(fmt.Sprintf("%v", v.Value)))
		}
	}
	// Now we create the CS-escaped string representation of our varbind arrays
	// and add them to the CSV array.
	csv[9] = fmt.Sprintf("\"['%v']\"", strings.Join(vbObj, "','"))
	csv[10] = fmt.Sprintf("\"['%v']\"", strings.Join(vbVal, "','"))

	return strings.Join(csv[:], ",")
}

var FilterPlugin ClickhouseExport
