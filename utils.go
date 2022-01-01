// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/damienstuart/trapex/actions"
	g "github.com/gosnmp/gosnmp"
	"github.com/natefinch/lumberjack"
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

// network stuct holds the data parsed from a CIDR representation of a
// subnet.
//
type network struct {
	ip  net.IP
	net *net.IPNet
}

// newNetwork initializes the network stuct based on the given CIDR
// formatted subnet.
//
func newNetwork(cidr string) (*network, error) {
	nm := network{}
	i, n, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	nm.ip = i
	nm.net = n
	return &nm, nil
}

// Returns true if the given IP falls within the subnet contained
// in the network object.
//
func (n *network) contains(ip net.IP) bool {
	return n.net.Contains(ip)
}

// logTrap takes care of logging the given trap to the given trapLogger
// destination.
//
func logTrap(sgt *plugin_interface.Trap, l *log.Logger) {
	l.Printf(makeTrapLogEntry(sgt))
}

// logCsvTrap takes care of logging the given trap to the given trapCsvLogger
// destination.
//
func logCsvTrap(sgt *plugin_interface.Trap, l *log.Logger) {
	l.Printf(makeTrapLogCsvEntry(sgt))
}

// panicOnError check an error pointer and panics if it is not nil.
//
/*
 */
func panicOnError(e error) {
	if e != nil {
		panic(e)
	}
}

// makeLogger initializes and returns a lumberjack.Logger (logger with
// built-in log rotation management).
//
func makeLogger(logfile string, teConf *trapexConfig) *lumberjack.Logger {
	l := lumberjack.Logger{
		Filename:   logfile,
		MaxSize:    teConf.Logging.LogMaxSize,
		MaxBackups: teConf.Logging.LogMaxBackups,
		Compress:   teConf.Logging.LogCompress,
	}
	return &l
}

// makeCsvLogger initializes and returns a lumberjack.Logger (logger with
// built-in log rotation management).
//
func makeCsvLogger(logfile string, teConf *trapexConfig) *lumberjack.Logger {
	l := lumberjack.Logger{
		Filename: logfile,
	}
	return &l
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
		genTrapType = trapType[trap.GenericTrap]
	} else {
		genTrapType = strconv.Itoa(trap.GenericTrap)
	}
	b.WriteString(fmt.Sprintf("\nTrap: %v", stats.TrapCount))
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

// makeTrapLogEntry creates a log entry string for the given trap data.
// Note that this particulare implementation expects to be dealing with
// only v1 traps.
//
func makeTrapLogCsvEntry(sgt *plugin_interface.Trap) string {
	var csv [11]string
	trap := sgt.Data

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
	csv[2] = fmt.Sprintf("\"%v\"", teConfig.General.Hostname)
	csv[3] = fmt.Sprintf("%v", stats.TrapCount)
	csv[4] = fmt.Sprintf("\"%v\"", sgt.SrcIP)
	csv[5] = fmt.Sprintf("\"%v\"", trap.AgentAddress)
	csv[6] = fmt.Sprintf("%v", trap.GenericTrap)
	csv[7] = fmt.Sprintf("%v", trap.SpecificTrap)
	csv[8] = fmt.Sprintf("\"%v\"", strings.Trim(trap.Enterprise, "."))

	var vbObj []string
	var vbVal []string

	// For escaping quotes and backslashes and replace newlines with a space
	replacer := strings.NewReplacer("\"", "\"\"", "'", "''", "\\", "\\\\", "\n", " - ", "%", "%%")

	// Process the Varbinds for this trap.
	// Varbinds are split to separate arrays - one for the ObjectIDs,
	// and the other for Values
	for _, v := range trap.Variables {
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

// secondsToDuration converts the given number of seconds into a more
// human-readable formatted string.
//
func secondsToDuration(s uint) string {
	var d uint
	var h uint
	var m uint
	if s >= 86400 {
		d = s / 86400
		s %= 86400
	}
	if s >= 3600 {
		h = s / 3600
		s %= 3600
	}
	if s >= 60 {
		m = s / 60
		s %= 60
	}
	return fmt.Sprintf("%vd-%vh-%vm-%vs", d, h, m, s)
}

// isIgnoredVersion returns a boolean indicating whether or not the given
// SnmpVersion value is being ignored.
//
func isIgnoredVersion(ver g.SnmpVersion) bool {
	for _, v := range teConfig.General.IgnoreVersions {
		if ver == v {
			return true
		}
	}
	return false
}
