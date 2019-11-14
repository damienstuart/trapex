package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	g "github.com/damienstuart/gosnmp"
)

/*
type trapAction interface {
	initAction(data string)
	processTrap(*sgTrap)
}
*/

type trapForwarder struct {
	destination	*g.GoSNMP
}

type trapLogger struct {
	logFile			string
	//logHandle		*bufio.Writer
	logHandle		*log.Logger
	isBroken		bool
}

// Filter types
const (
	parseTypeAny int = iota	// Match anything (wildcard)
	parseTypeString			// Direct String comparison
	parseTypeInt			// Direct Integer comparison
	parseTypeRegex			// Regular Expression
	parseTypeCIDR			// CIDR IP/Netmask 
	parseTypeRange			// Integer range x:y
)

// Filter items
const (
	srcIP int = iota
	agentAddr
	genericType
	specificType
	enterprise
)

const (
	actionDrop int = iota
	actionNat
	actionForward
	actionLog
)

type filterObj struct {
	filterItem	int
	filterType	int
	filterValue	interface{}		// string, *regex.Regexp, *network, int
}

type trapexFilter struct {
	filterItems	[]filterObj
	matchAll	bool
	action		interface{}
	actionType	int
	actionArg	string
}

func (a *trapForwarder) initAction(dest string) {
	s := strings.Split(dest, ":")
	port, err := strconv.Atoi(s[1])
	if err != nil {
		panic("Invalid destination port: " + s[1])
	}
	a.destination = &g.GoSNMP{
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
	err = a.destination.Connect()
	if err != nil {
		panic(err)
	}
	fmt.Printf(" -Added trap destination: %s, port %s\n", s[0], s[1])
}

func (a trapForwarder) processTrap(trap *sgTrap) error {
	_, err := a.destination.SendTrap(trap.data)
	return err
}

func (a *trapLogger) initAction(logfile string) {
	fd, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	checkErr(err)
	a.logFile = logfile
	a.logHandle = log.New(fd, "", 0)
	a.logHandle.SetOutput(makeLogger(logfile))
	fmt.Printf(" -Added log destination: %s\n", logfile)
}

func (a *trapLogger) processTrap(trap *sgTrap) {
	logTrap(trap, a.logHandle)
}
