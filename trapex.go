package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	g "github.com/damienstuart/gosnmp"
)

const myVersion string = "0.9.0-beta"

//var runLogger *log.Logger

// teStats is a structure for holding trapex stats.
type teStats struct {
	startTime     uint32
	trapCount     uint64
	filteredTraps uint64
	fromV2c       uint64
	fromV3        uint64
}

var stats teStats

// sgTrap holds a pointer to a trap and the source IP of
// the incoming trap.
//
type sgTrap struct {
	data       g.SnmpTrap
	trapVer    g.SnmpVersion
	srcIP      net.IP
	translated bool
	dropped    bool
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

	tl := g.NewTrapListener()

	tl.OnNewTrap = trapHandler
	tl.Params = g.Default
	tl.Params.Community = ""

	// Uncomment for debugging gosnmp
	if teConfig.debug == true {
		fmt.Println("*DEBUG MODE ENABLED*")
		tl.Params.Logger = log.New(os.Stdout, "", 0)
	}

	// SNMP v3 stuff
	tl.Params.SecurityModel = g.UserSecurityModel
	tl.Params.MsgFlags = teConfig.v3Params.msgFlags
	tl.Params.Version = g.Version3
	tl.Params.SecurityParameters = &g.UsmSecurityParameters{
		UserName:					teConfig.v3Params.username,
		AuthenticationProtocol:		teConfig.v3Params.authProto,
		AuthenticationPassphrase:	teConfig.v3Params.authPassword,
		PrivacyProtocol:   			teConfig.v3Params.privacyProto,
		PrivacyPassphrase:			teConfig.v3Params.privacyPassword,
	}

	fmt.Printf("SNMPv3 security Params: %v\n", teConfig.v3Params)

	listenAddr := fmt.Sprintf("%s:%s", teConfig.listenAddr, teConfig.listenPort)
	fmt.Printf("Start trapex listener on %s\n", listenAddr)
	err := tl.Listen(listenAddr)
	if err != nil {
		log.Panicf("error in listen on %s: %s", listenAddr, err)
	}
}

func trapHandler(p *g.SnmpPacket, addr *net.UDPAddr) {
	stats.trapCount++

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

	if teConfig.debug {
		fmt.Printf(makeTrapLogEntry(&trap).String())
	}
	processTrap(&trap)
}

// processTrap is the entry point to code that checks the incoming trap
// against the filter list and processes the trap accordingly.
//
func processTrap(sgt *sgTrap) {
	for _, f := range teConfig.filters {
		// If this trap is tagged to drop and the action is not log - or
		// if logging dropped traps is not enabled, then continue.
		if sgt.dropped && f.actionType != actionLog {
			continue
		}
		// If matchAll is true, just process the action.
		if f.matchAll == true {
			// We don't expect to see this here (set a wide open filter for
			// drop).... (but...)
			if f.actionType == actionDrop {
				sgt.dropped = true
				continue
				//break
			} 
			processAction(sgt, &f)
		} else {
			// Determine if this trap matches this filter
			if isFilterMatch(sgt, &f) == true {
				if f.actionType == actionDrop {
					sgt.dropped = true
					continue
					//break
				}
				processAction(sgt, &f)
			}
		}

	}
}

func processAction(sgt *sgTrap, f *trapexFilter) {
	switch f.actionType {
	case actionDrop:
		sgt.dropped = true 
		return
	case actionNat:
		if f.actionArg == "$SRC_IP" {
			sgt.data.AgentAddress = sgt.srcIP.String()
		} else {
			sgt.data.AgentAddress = f.actionArg
		}
	case actionForward:
		f.action.(*trapForwarder).processTrap(sgt)
	case actionLog:
		if !sgt.dropped || teConfig.logDropped {
			f.action.(*trapLogger).processTrap(sgt)
		}
	}
}

func isFilterMatch(sgt *sgTrap, f *trapexFilter) bool {
	// Assume true - until one of the filter items does not match
	trap := &(sgt.data)
	for _, fo := range f.filterItems {
		fval := fo.filterValue
		switch fo.filterItem {
		case srcIP:
			if fo.filterType == parseTypeString && fval.(string) != sgt.srcIP.String() {
				return false
			} else if fo.filterType == parseTypeCIDR && !fval.(*network).contains(sgt.srcIP) {
				return false
			}
		case agentAddr:
			if fo.filterType == parseTypeString && fval.(string) != trap.AgentAddress {
				return false
			}
			if fo.filterType == parseTypeCIDR && !fval.(*network).contains(net.ParseIP(trap.AgentAddress)) {
				return false
			}
		case enterprise:
			if fo.filterType == parseTypeRegex && !fval.(*regexp.Regexp).MatchString(strings.TrimLeft(trap.Enterprise,".")) {
				return false
			} 
			if fo.filterType == parseTypeString && fval.(string) != strings.TrimLeft(trap.Enterprise,".") {
				return false
			} 
		case genericType:
			if fo.filterType == parseTypeInt && fval.(int) != trap.GenericTrap {
				return false
			}
		case specificType:
			if fo.filterType == parseTypeInt && fval.(int) != trap.SpecificTrap {
				return false
			}
		}
	}
	return true
}
