// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

// exposeMetrics
// Allow Prometheus to gather current performance metrics via /metrics URL
func exposeMetrics() {
	server := http.NewServeMux()
	server.Handle("/"+teConfig.General.PrometheusEndpoint, promhttp.Handler())
	var listenAddress = teConfig.General.PrometheusIp + ":" + teConfig.General.PrometheusPort
	http.ListenAndServe(listenAddress, server)
}
