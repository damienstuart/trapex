package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type versionResponseMsg struct {
	TrapexVersion string
}

type filterLineInfo struct {
	LineNumber uint
	FilterLine string
}

type filterListResponseMesg struct {
	ConfigFile  string
	FilterCount uint
	Filters     []filterLineInfo
}

func sendResponse(w http.ResponseWriter, respData interface{}) {
	b, err := json.Marshal(respData)
	if err != nil {
		fmt.Printf("*Error processing web response: %v", err)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.Write(b)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	m := versionResponseMsg{myVersion}
	sendResponse(w, m)
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	// Compute uptime
	now := time.Now()
	stats.UptimeInt = now.Unix() - stats.StartTime.Unix()
	stats.Uptime = secondsToDuration(uint(stats.UptimeInt))
	sendResponse(w, stats)
}

func handleFilterList(w http.ResponseWriter, r *http.Request) {
	numFilters := uint(len(teConfig.filters))
	resp := filterListResponseMesg{}
	resp.ConfigFile = teCmdLine.configFile
	filterLines := make([]filterLineInfo, numFilters)
	resp.FilterCount = numFilters
	for i, f := range teConfig.filters {
		filterLines[i].LineNumber = f.lineNumber
		filterLines[i].FilterLine = f.filterLine
	}
	resp.Filters = filterLines
	sendResponse(w, resp)
}

func httpListener(port int) {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/stats", handleStats)
	http.HandleFunc("/filter_list", handleFilterList)
	http.ListenAndServe(":8008", nil)
}
