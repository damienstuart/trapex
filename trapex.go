package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	g "github.com/gosnmp/gosnmp"
)

// sgTrap holds a pointer to a trap and the source IP of
// the incoming trap.
//
type sgTrap struct {
	trapNumber uint64
	data       g.SnmpTrap
	trapVer    g.SnmpVersion
	srcIP      net.IP
	translated bool
	dropped    bool
}

var trapRateTracker = newTrapRateTracker()

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage:\n")
		fmt.Printf("   %s\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	// Process the command-line and get the configuration.
	//
	processCommandLine()

	if err := getConfig(); err != nil {
		panic(err)
	}

	initSigHandlers()

	stats.StartTime = time.Now()

	go trapRateTracker.start()

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
		UserName:                 teConfig.v3Params.username,
		AuthenticationProtocol:   teConfig.v3Params.authProto,
		AuthenticationPassphrase: teConfig.v3Params.authPassword,
		PrivacyProtocol:          teConfig.v3Params.privacyProto,
		PrivacyPassphrase:        teConfig.v3Params.privacyPassword,
	}

	listenAddr := fmt.Sprintf("%s:%s", teConfig.listenAddr, teConfig.listenPort)
	fmt.Printf("Start trapex listener on %s\n", listenAddr)
	err := tl.Listen(listenAddr)
	if err != nil {
		log.Panicf("error in listen on %s: %s", listenAddr, err)
	}
}

// trapHandler is the callback for handling traps received by the listener.
//
func trapHandler(p *g.SnmpPacket, addr *net.UDPAddr) {
	// Count every trap received
	stats.TrapCount++

	// First thing to do is check for ignored versions
	if isIgnoredVersion(p.Version) {
		stats.IgnoredTraps++
		return
	}

	// Also keep track of traps we handle
	stats.HandledTraps++

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
			fmt.Printf(makeTrapLogEntry(&trap))
		}
	}

	if teConfig.debug {
		fmt.Printf(makeTrapLogEntry(&trap))
	}

	processTrap(&trap)
}

// processTrap is the entry point to code that checks the incoming trap
// against the filter list and processes the trap accordingly.
//
func processTrap(sgt *sgTrap) {
	for _, f := range teConfig.filters {
		// If this trap is tagged to drop, then continue.
		if sgt.dropped {
			continue
		}
		// If matchAll is true, just process the action.
		if f.matchAll == true {
			// We don't expect to see this here (set a wide open filter for
			// drop).... (but...)
			if f.actionType == actionBreak {
				sgt.dropped = true
				stats.DroppedTraps++
				continue
			}
			f.processAction(sgt)
			if f.actionType == actionForwardBreak || f.actionType == actionLogBreak || f.actionType == actionCsvBreak {
				sgt.dropped = true
				stats.DroppedTraps++
				continue
			}
		} else {
			// Determine if this trap matches this filter
			if f.isFilterMatch(sgt) {
				if f.actionType == actionBreak {
					sgt.dropped = true
					stats.DroppedTraps++
					continue
				}
				f.processAction(sgt)
				if f.actionType == actionForwardBreak || f.actionType == actionLogBreak || f.actionType == actionCsvBreak {
					sgt.dropped = true
					stats.DroppedTraps++
					continue
				}
			}
		}
	}
}
