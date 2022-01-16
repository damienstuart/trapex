// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package pluginMeta

import (
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"time"

	g "github.com/gosnmp/gosnmp"
)

// Trap holds a pointer to the raw trap and other meta-data
//
type Trap struct {
	TrapNumber  uint
	Data        g.SnmpTrap
	SnmpVersion g.SnmpVersion
	SrcIP       net.IP
	Translated  bool
	Dropped     bool
	Hostname    string
}

func (trap *Trap) Trap2Map() map[string]string {
	trapMap := make(map[string]string)
	raw_trap := trap.Data

	// FIXME: should include a check to validate that we work with only SNMP v1 traps?
	var ts = time.Now().Format(time.RFC3339)
	trapMap["TrapDate"] = fmt.Sprintf("%v", ts[:10])
	trapMap["TrapTimestamp"] = fmt.Sprintf("%v %v", ts[:10], ts[11:19])

	trapMap["TrapHost"] = fmt.Sprintf("\"%v\"", trap.Hostname)

	//trapMap[3] = fmt.Sprintf("%v", stats.TrapCount)
	// FIXME: global stats object not visible in plugin space
	trapMap["TrapNumber"] = fmt.Sprintf("%v", 1)

	trapMap["TrapSourceIP"] = fmt.Sprintf("\"%v\"", trap.SrcIP)
	trapMap["TrapAgentAddress"] = fmt.Sprintf("\"%v\"", raw_trap.AgentAddress)
	trapMap["TrapGenericType"] = fmt.Sprintf("%v", raw_trap.GenericTrap)
	trapMap["TrapEnterpriseOID"] = fmt.Sprintf("%v", raw_trap.SpecificTrap)
	trapMap["TrapEnterpriseOID"] = fmt.Sprintf("\"%v\"", strings.Trim(raw_trap.Enterprise, "."))

	// For escaping quotes and backslashes and replace newlines with a space
	replacer := strings.NewReplacer("\"", "\"\"", "'", "''", "\\", "\\\\", "\n", " - ", "%", "%%")

	// Process the Varbinds for this raw_trap.
	var oidPath, oidValue string
	for _, v := range raw_trap.Variables {
		// Get the OID
		oidPath = strings.Trim(v.Name, ".")
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
				oidValue = fmt.Sprintf("%v", replacer.Replace(hex.EncodeToString(val)))
			} else {
				oidValue = replacer.Replace(fmt.Sprintf("%v", string(val)))
			}
		default:
			oidValue = replacer.Replace(fmt.Sprintf("%v", v.Value))
		}
		trapMap[oidPath] = oidValue
	}

	return trapMap
}
