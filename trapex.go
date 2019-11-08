package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	g "github.com/damienstuart/gosnmp"
)

var trapDestList = []string{
	"192.168.7.12:162",
}

// teStats is a structure for holding trapex stats.
type teStats struct {
	startTime     uint32
	trapCount     uint64
	filteredTraps uint64
	fromV2c       uint64
	fromV3        uint64
}

var stats teStats

// Trap destinations
var trapDests []*g.GoSNMP

// sgTrap holds a pointer to a trap and the source IP of
// the incoming trap.
//
type sgTrap struct {
	data       g.SnmpTrap
	trapVer    g.SnmpVersion
	srcIP      net.IP
	translated bool
}

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage:\n")
		fmt.Printf("   %s\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	// Get the configuration
	//
	getConfig()

	fmt.Printf("Filters: %v\n", teConfig.filters)

	tl := g.NewTrapListener()

	tl.OnNewTrap = trapHandler
	tl.Params = g.Default
	tl.Params.Community = ""

	// Uncomment for debugging gosnmp
	if teConfig.debug == true {
		fmt.Println("DEBUG MODE ENABLED")
		tl.Params.Logger = log.New(os.Stdout, "", 0)
	}

	// SNMP v3 stuff
	tl.Params.SecurityModel = g.UserSecurityModel
	tl.Params.MsgFlags = g.AuthPriv
	tl.Params.Version = g.Version3
	tl.Params.SecurityParameters = &g.UsmSecurityParameters{
		UserName:					teConfig.v3Params.username,
		AuthenticationProtocol:		teConfig.v3Params.authProto,
		AuthenticationPassphrase:	teConfig.v3Params.authPassword,
		PrivacyProtocol:   			teConfig.v3Params.privacyProto,
		PrivacyPassphrase:			teConfig.v3Params.privacyPassword,
	}

	makeTrapDests()

	listenAddr := fmt.Sprintf("%s:%s", teConfig.listenAddr, teConfig.listenPort)
	fmt.Println("Start trapex listener on " + listenAddr)
	err := tl.Listen(listenAddr)
	if err != nil {
		log.Panicf("error in listen on %s: %s", listenAddr, err)
	}
}

func trapHandler(p *g.SnmpPacket, addr *net.UDPAddr) {
	stats.trapCount++
	/*
		if filteredTrap(p, addr.IP) {
			//fmt.Printf("*** Filtered trap %s from %s\n", p.Enterprise, addr.IP)
			trapsFiltered++
			return
		}
	*/

	// Make the trap
	trap := sgTrap{
		data: g.SnmpTrap{
			Variables:    p.Variables,
			Enterprise:   p.Enterprise,
			AgentAddress: p.AgentAddress,
			GenericTrap:  p.GenericTrap,
			SpecificTrap: p.SpecificTrap,
			Timestamp:    p.Timestamp,
		},
		srcIP:   addr.IP,
		trapVer: p.Version,
	}

	// Translate to v1 if needed
	/*
	 */
	if p.Version > g.Version1 {
		err := translateToV1(&trap)
		if err != nil {
			fmt.Printf("Error translating to v1: %v\n", err)
		}
	}

	logTrap(&trap)

	go forwardTrap(&trap)
}

// forwardTrap sends the incoming trap to the configured trap destinations.
//
func forwardTrap(trap *sgTrap) {
	for _, td := range trapDests {
		_, err := td.SendTrap(trap.data)
		if err != nil {
			fmt.Printf("SendTrap() error sending to %s: trap# %v: %v\n", td.Target, stats.trapCount, err)
		}
	}
}

// logTrap (for now) prints to stdout - a format that mimics the current
// SG trapexploder log file format.
//
func logTrap(t *sgTrap) {
	trap := &t.data

	fmt.Printf("\nTrap: %v", stats.trapCount)
	if t.translated == true {
		fmt.Printf(" (translated from v%s)", t.trapVer.String())
	}
	fmt.Printf("\n\t%s\n", time.Now().Format(time.ANSIC))
	fmt.Printf("\tSrc IP: %s\n", t.srcIP)
	fmt.Printf("\tAgent: %s\n", trap.AgentAddress)
	//fmt.Printf("\tVersion: %s\n", t.trapVer.String())
	fmt.Printf("\tTrap Type: %v\n", trap.GenericTrap)
	fmt.Printf("\tSpecific Type: %v\n", trap.SpecificTrap)
	fmt.Printf("\tEnterprise: %s\n", strings.Trim(trap.Enterprise, "."))
	fmt.Printf("\tTimestamp: %v\n", trap.Timestamp)

	// Process the Varbinds.
	for _, v := range trap.Variables {
		vbName := strings.Trim(v.Name, ".")
		switch v.Type {
		case g.OctetString:
			//b := v.Value.([]byte)
			//fmt.Printf("\tObject:%s Value:%s\n", vbName, cleanOctets(b))
			fmt.Printf("\tObject:%s Value:%s\n", vbName, string(v.Value.([]byte)))
		default:
			fmt.Printf("\tObject:%s Value:%v\n", vbName, v.Value)
		}
	}
}

// makeTrapDests populates the trapDest array with the list
// of trap destinations and ports.
//
func makeTrapDests() {
	tDests := make([]*g.GoSNMP, len(trapDestList))
	for i, d := range trapDestList {
		s := strings.Split(d, ":")
		port, err := strconv.Atoi(s[1])
		if err != nil {
			panic("Invalid destination port: " + d)
		}
		td := &g.GoSNMP{
			Target:             s[0],
			Port:               uint16(port),
			Transport:          "udp",
			Community:          "",
			Version:            g.Version1,
			Timeout:            time.Duration(2) * time.Second,
			Retries:            3,
			ExponentialTimeout: true,
			MaxOids:            g.MaxOids,
		}
		err = td.Connect()
		if err != nil {
			panic(err)
		}
		tDests[i] = td
		fmt.Printf("--Added trap destination: %s, port %s\n", s[0], s[1])
	}
	trapDests = tDests
}

// -DSS Temp - filteredTrap checks the incoming trap against various filters and will
// return true if the trap should be filtered (ignored)
func filteredTrap(p *g.SnmpPacket, ip net.IP) bool {
	if len(p.Enterprise) > 1 && strings.Trim(p.Enterprise, ".") == "1.3.6.1.4.1.546.1.1" {
		return true
	}
	return false
}

// cleanOctets takes an array of bytes and removes non-ascii (or rather
// printable) characters. It will allow for tab and newline characters
// however. The result is returned as a string.
//
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
