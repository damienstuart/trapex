package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"strings"
	"strconv"
	"time"

	g "github.com/damienstuart/gosnmp"
)

var trapType = [...]string {
	"Cold Start",
	"Warm Start",
	"Link Down",
	"Link Up",
	"Authentication Failure",
	"EGP Neighbor Loss",
	"Vendor Specific",
}

type network struct {
	ip	net.IP
	net	*net.IPNet
}

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

func (n *network) contains (ip net.IP) bool {
	return n.net.Contains(ip)
}

func logTrap(sgt *sgTrap, l *log.Logger) {
	l.Printf(makeTrapLogEntry(sgt).String())
}

func makeTrapLogEntry(sgt *sgTrap) *(strings.Builder) {
	var b strings.Builder
	var genTrapType string
	trap := sgt.data

	if trap.GenericTrap >= 0 && trap.GenericTrap <= 6 {
		genTrapType = trapType[trap.GenericTrap]
	} else {
		genTrapType = strconv.Itoa(trap.GenericTrap)
	}
	b.WriteString(fmt.Sprintf("\nTrap: %v", stats.trapCount))
	if sgt.translated == true {
		b.WriteString(fmt.Sprintf(" (translated from v%s)", sgt.trapVer.String()))
	}
	if sgt.dropped == true {
		b.WriteString(fmt.Sprintf(" (DROPPED)"))
	}
	b.WriteString(fmt.Sprintf("\n\t%s\n", time.Now().Format(time.ANSIC)))
	b.WriteString(fmt.Sprintf("\tSrc IP: %s\n", sgt.srcIP))
	b.WriteString(fmt.Sprintf("\tAgent: %s\n", trap.AgentAddress))
	b.WriteString(fmt.Sprintf("\tTrap Type: %s\n", genTrapType))
	b.WriteString(fmt.Sprintf("\tSpecific Type: %v\n", trap.SpecificTrap))
	b.WriteString(fmt.Sprintf("\tEnterprise: %s\n", strings.Trim(trap.Enterprise, ".")))
	b.WriteString(fmt.Sprintf("\tTimestamp: %v\n", trap.Timestamp))

	// Process the Varbinds.
	for _, v := range trap.Variables {
		vbName := strings.Trim(v.Name, ".")
		switch v.Type {
		case g.OctetString:
			var nonAscii bool
			val := v.Value.([]byte)
			if len(val) > 0 {
				for i:=0; i<len(val); i++ {
					if (val[i] < 32 || val[i] > 127) && val[i] != 9 && val[i] != 10 {
						nonAscii = true
						break
					}
				}
			}
			// Non-printable/Non-ascii strings will be dumped as a hex string.
			if nonAscii {
				b.WriteString(fmt.Sprintf("\tObject:%s Value:%v\n", vbName, hex.EncodeToString(val)))
			} else {
				b.WriteString(fmt.Sprintf("\tObject:%s Value:%s\n", vbName, string(val)))
			}
		default:
			b.WriteString(fmt.Sprintf("\tObject:%s Value:%v\n", vbName, v.Value))
		}
	}
	return &b
}
