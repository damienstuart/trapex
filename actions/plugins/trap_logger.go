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
"os"
"log"
"time"
"strconv"
"strings"
        "encoding/hex"
        g "github.com/gosnmp/gosnmp"

"fmt"
        "github.com/rs/zerolog"
        "github.com/damienstuart/trapex/actions"

)

type trapLogger struct {
        logFile   string
        fd        *os.File
        logHandle *log.Logger
        isBroken  bool
}

const plugin_name = "trap logger"


func (a trapLogger) Init(logger zerolog.Logger) error {
        //logger.Info().Str("plugin", plugin_name).Msg("Initialization of plugin")

/*
        fd, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if err != nil {
                return err
        }
        a.fd = fd
        a.logFile = logfile
        a.logHandle = log.New(fd, "", 0)
        a.logHandle.SetOutput(makeLogger(logfile, teConf))
        logger.Info().Str("logfile", logfile).Msg("Added log destination")
        return nil
*/
   return nil
}

func (a trapLogger) ProcessTrap(trap *plugin_interface.Trap) error {
        logTrap(trap, a.logHandle)

        //logger.Info().Str("plugin", plugin_name).Msg("Processing trap")
   return nil
}

func logTrap(sgt *plugin_interface.Trap, l *log.Logger) {
        l.Printf(makeTrapLogEntry(sgt))
}

func (p trapLogger) SigUsr1() error {
   return nil
}

func (p trapLogger) SigUsr2() error {
   return nil
}

func (a *trapLogger) close() {
        a.fd.Close()
}

// makeTrapLogEntry creates a log entry string for the given trap data.
// Note that this particulare implementation expects to be dealing with
// only v1 traps.
//
func makeTrapLogEntry(sgt *plugin_interface.Trap) string {
        var b strings.Builder
        var genTrapType string
        trap := sgt.Data

        if trap.GenericTrap >= 0 && trap.GenericTrap <= 6 {
// FIXME: can't find trapType
                //genTrapType = trapType[trap.GenericTrap]
                genTrapType = strconv.Itoa(trap.GenericTrap)
        } else {
                genTrapType = strconv.Itoa(trap.GenericTrap)
        }
// FIXME: how to pass in stats
        //b.WriteString(fmt.Sprintf("\nTrap: %v", stats.TrapCount))
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

