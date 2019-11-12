package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	g "github.com/damienstuart/gosnmp"
)

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
	trap := sgt.data

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
	b.WriteString(fmt.Sprintf("\tTrap Type: %v\n", trap.GenericTrap))
	b.WriteString(fmt.Sprintf("\tSpecific Type: %v\n", trap.SpecificTrap))
	b.WriteString(fmt.Sprintf("\tEnterprise: %s\n", strings.Trim(trap.Enterprise, ".")))
	b.WriteString(fmt.Sprintf("\tTimestamp: %v\n", trap.Timestamp))

	// Process the Varbinds.
	for _, v := range trap.Variables {
		vbName := strings.Trim(v.Name, ".")
		switch v.Type {
		case g.OctetString:
			val := v.Value.([]byte)
			// Non-printable/Non-ascii strings will be dumped as a hex string.
			if len(val) > 0 && (val[0] < 32 || val[0] > 127) {
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
