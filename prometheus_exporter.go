// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
    "fmt"
    "net/http"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)


// exposeMetrics
// Allow Prometheus to gather current performance metrics via /metrics URL
func exposeMetrics() {
    server := http.NewServeMux()
    server.Handle("/metrics", promhttp.Handler())
    http.ListenAndServe(teConfig.promServerPort, server)
    fmt.Printf("Prometheus metrics exported on http://%s/metrics\n", teConfig.promServerPort)
}

