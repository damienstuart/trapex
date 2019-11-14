package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"strings"
	"strconv"
	"time"

	"github.com/damienstuart/lumberjack"
	g "github.com/damienstuart/gosnmp"
)

// trapType is an array of trap Generic Type human-friendly names
// ordered by the type value.
//
var trapType = [...]string {
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
	ip	net.IP
	net	*net.IPNet
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
func (n *network) contains (ip net.IP) bool {
	return n.net.Contains(ip)
}

// logTrap takes care of logging the given trap to the given trapLogger
// destination.
//
func logTrap(sgt *sgTrap, l *log.Logger) {
	l.Printf(makeTrapLogEntry(sgt))
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
		Filename:	logfile,
		MaxSize: 	teConf.logMaxSize,
		MaxBackups: teConf.logMaxBackups,
		Compress:	teConf.logCompress,
	}
	return &l
}

// makeTrapLogEntry creates a log entry string for the given trap data.
// Note that this particulare implementation expects to be dealing with
// only v1 traps.
//
func makeTrapLogEntry(sgt *sgTrap) string {
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

	// Process the Varbinds for this trap.
	for _, v := range trap.Variables {
		vbName := strings.Trim(v.Name, ".")
		switch v.Type {
		case g.OctetString:
			var nonASCII bool
			val := v.Value.([]byte)
			if len(val) > 0 {
				for i:=0; i<len(val); i++ {
					if (val[i] < 32 || val[i] > 127) && val[i] != 9 && val[i] != 10 {
						nonASCII = true
						break
					}
				}
			}
			// Strings with non-printable/non-ascii characters will be dumped
			// as a hex string. Otherwise, just as a plain string.
			if nonASCII {
				b.WriteString(fmt.Sprintf("\tObject:%s Value:%v\n", vbName, hex.EncodeToString(val)))
			} else {
				b.WriteString(fmt.Sprintf("\tObject:%s Value:%s\n", vbName, string(val)))
			}
		default:
			b.WriteString(fmt.Sprintf("\tObject:%s Value:%v\n", vbName, v.Value))
		}
	}
	return b.String()
}
