package main

import (
	"bufio"
	"fmt"
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

/*
func inetAtoN (ipStr string) uint32 {
	var ipInt uint32
	for i, b := range net.ParseIP(ipStr).To4() {
		ipInt |= uint32(b)
		if i < 3 {
			ipInt <<= 8
		}
	}
	return ipInt
}

func inetNtoA(ip uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d", byte(ip>>24), byte(ip>>16), byte(ip>>8),
		byte(ip))
}
*/

func logr(sgt *sgTrap, fd *bufio.Writer) error {
	le := makeTrapLogEntry(sgt)
	fd.WriteString(le.String())
	err := fd.Flush()
	if err != nil {
		return err
	}
	return nil
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
			//b := v.Value.([]byte)
			//fd.WriteString(fmt.Printf("\tObject:%s Value:%s\n", vbName, cleanOctets(b)))
			b.WriteString(fmt.Sprintf("\tObject:%s Value:%s\n", vbName, string(v.Value.([]byte))))
		default:
			b.WriteString(fmt.Sprintf("\tObject:%s Value:%v\n", vbName, v.Value))
		}
	}
	return &b
}

// cleanOctets takes an array of bytes and removes non-ascii (or rather
// printable) characters. It will allow for tab and newline characters
// however. The result is returned as a string.
//
/*
func cleanOctets(indat []byte) string {
	n := 0
	for _, c := range indat {
		if c >= 32 && c < 127 && c != 10 && c != 9 {
			indat[n] = c
			n++
		}
	}
	return string(indat[:n])
}
*/
