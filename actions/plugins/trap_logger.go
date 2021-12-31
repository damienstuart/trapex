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
        logger.Info().Str("plugin", plugin_name).Msg("Initialization of plugin")

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

        logger.Info().Str("plugin", plugin_name).Msg("Processing trap")
   return nil
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


// Exported symbol which supports filter.go's FilterAction type
var FilterPlugin trapLogger

