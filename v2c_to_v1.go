package main

import (
	"fmt"
	"strconv"
	"strings"

	g "github.com/gosnmp/gosnmp"
)

// OID constants we will need for v2c to v1 conversion.
const (
	snmpTraps          = ".1.3.6.1.6.3.1.1.5"
	snmpTrapOID        = ".1.3.6.1.6.3.1.1.4.1.0"
	snmpTrapEnterprise = ".1.3.6.1.6.3.1.1.4.3.0"
	snmpTrapAddress    = ".1.3.6.1.6.3.18.1.3.0"
	sysUpTime          = ".1.3.6.1.2.1.1.3.0"
)

// translateToV1 converts a trap from v2c/v3 to v1 per RFC-3584
//
func translateToV1(t *sgTrap) error {
	// If this is already a v1 trap, there is nothing to do.
	if t.trapVer == g.Version1 {
		return nil
	}

	trap := &t.data
	if len(trap.Variables) < 2 {
		return fmt.Errorf("got invalid v2 trap with less than 2 varbinds: %v", trap)
	}
	vb0 := trap.Variables[0]
	vb1 := trap.Variables[1]

	// We expect varbind 0 to be sysUptime (type: TimeTicks). If it isn't,
	// something is wrong and we bail.
	//
	if vb0.Name != sysUpTime || vb0.Type != g.TimeTicks {
		return fmt.Errorf("Invalid sysUptime (varbind0) for v2c trap: %v", vb0)
	}
	// We also expect varbind 1 to the snmpTrapOID.
	if vb1.Name != snmpTrapOID || vb1.Type != g.ObjectIdentifier {
		return fmt.Errorf("Invalid snmpTrapOID (varbind1) for v2c trap: %v", vb0)
	}

	// Use the sysUpTime varbind value for v1 timestamp
	trap.Timestamp = uint(vb0.Value.(uint32))

	trapOID := vb1.Value.(string)

	// Let's see if we are dealing with a standard trap
	var isStd bool
	if strings.HasPrefix(trapOID, snmpTraps) {
		isStd = true
		// Remove any trailing .0 from the trapOID value
		if strings.HasSuffix(trapOID, ".0") {
			trapOID = trapOID[:len(trapOID)-2]
		}
	}

	// Get the the position of the last "." in the trapOID so we can use that
	// as our first demarcation to parse out the last OID element.
	dmarc := strings.LastIndex(trapOID, ".")

	// Using the dmarc, we get the value of the last element of trapOID
	jval, err := strconv.Atoi(trapOID[dmarc+1:])
	if err != nil {
		return fmt.Errorf("Error parsing last element of trapOID: %s", trapOID)
	}

	// For standard traps, generic trap type will be set to the value
	// of the last element - 1 , and specific will be 0.
	if isStd == true {
		trap.GenericTrap = jval - 1
		trap.SpecificTrap = 0
		// Otherwise
	} else {
		trap.GenericTrap = 6
		trap.SpecificTrap = jval
		// Also if the next to last element in trapOID is "0", then we
		// move the demarc to the next previous "." so we can discard
		// the last to elements in case we need to use it for the Enterprise
		// value.
		ndx := strings.LastIndex(trapOID[:dmarc], ".")
		if trapOID[ndx+1:dmarc] == "0" {
			dmarc = ndx
		}
	}

	// Now process the varbinds: Skip the first 2 and remove any Counter64
	// types. We may be capturing the Enterprise and AgentAddress value
	// from the varbinds as well. If not, we have fallbacks for them later.
	var enterprise string
	var agentAddress string
	n := 0
	for i, v := range trap.Variables {
		// Skip the first 2 varbinds as we don't need them in the v1 trap.
		if i < 2 {
			continue
		}
		// Ignore/Skip any Counter64 types as they are not supported in v1.
		if v.Type != g.Counter64 {
			trap.Variables[n] = v
			n++
		}
		// If this is a standard trap and the snmpTrapEnterprise OID, set
		// the v1 Enterprise value accordingly
		if isStd && v.Name == snmpTrapEnterprise {
			enterprise = v.Value.(string)
			// Or if we have an snmpTrapAddress OID set agentAddress to its value
		} else if v.Name == snmpTrapAddress {
			agentAddress = v.Value.(string)
		}
	}
	trap.Variables = trap.Variables[:n]

	// Figure out what our Enterprise value should be.
	if len(enterprise) > 0 {
		trap.Enterprise = enterprise
	} else if isStd {
		trap.Enterprise = snmpTraps
	} else {
		trap.Enterprise = trapOID[:dmarc]
	}

	// Now set the Agent address. If it is not in the trap data, then use the
	// packet source IP.
	if len(agentAddress) > 0 {
		trap.AgentAddress = agentAddress
	} else {
		trap.AgentAddress = t.srcIP.String()
	}

	t.translated = true

	// Update the translate stats
	if t.trapVer == g.Version2c {
		stats.TranslatedFromV2c++
	} else if t.trapVer == g.Version3 {
		stats.TranslatedFromV3++
	}

	return nil
}
