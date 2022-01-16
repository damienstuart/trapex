package main

/*
 * Capture Prometheus metrics
 */

import (
	"fmt"
	"net/http"

	pluginMeta "github.com/damienstuart/trapex/txPlugins"

	"github.com/rs/zerolog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type prometheusStats struct {
	trapex_log *zerolog.Logger

	listenAddress string
	endpoint      string

	counters []prometheus.Counter
}

func (p *prometheusStats) Configure(trapexLog *zerolog.Logger, args map[string]string, metric_definitions []pluginMeta.MetricDef) error {
	p.trapex_log = trapexLog
	listenIP := args["listen_ip"]
	listenPort := args["listen_port"]
	p.listenAddress = listenIP + ":" + listenPort
	p.endpoint = args["endpoint"]

	for i, definition := range metric_definitions {
		p.counters[i] = promauto.NewCounter(prometheus.CounterOpts{
			Name: definition.Name,
			Help: definition.Help,
		})
	}

	exporter := fmt.Sprintf("http://%s/%s", p.listenAddress, p.endpoint)
	p.trapex_log.Info().Str("endpoint", exporter).Msg("Prometheus metrics exporter")

	go exposeMetrics(p.endpoint, p.listenAddress)

	return nil
}

func (p prometheusStats) Inc(metricIndex int) {

	p.counters[metricIndex].Inc()

}

func (p prometheusStats) Report() (string, error) {
	return "", nil
}

// exposeMetrics
// Allow Prometheus to gather current performance metrics via /metrics URL
func exposeMetrics(endpoint string, listenAddress string) {
	server := http.NewServeMux()
	server.Handle(endpoint, promhttp.Handler())
	http.ListenAndServe(listenAddress, server)
}

var MetricPlugin prometheusStats
