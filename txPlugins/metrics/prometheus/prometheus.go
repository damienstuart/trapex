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

	trapsTotal   prometheus.Counter
	trapsHandled prometheus.Counter
	trapsDropped prometheus.Counter
	trapsIgnored prometheus.Counter
	trapsFromV2c prometheus.Counter
	trapsFromV3  prometheus.Counter
}


func (p *prometheusStats) Configure(trapexLog *zerolog.Logger, args map[string]string, metric_definitions []pluginMeta.MetricDef) error {
	p.trapex_log = trapexLog
	listenIP := args["listen_ip"]
	listenPort := args["listen_port"]
	p.listenAddress = listenIP + ":" + listenPort
	p.endpoint = args["endpoint"]

	p.trapsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "trapex_incoming_traps_total",
		Help: "The total number of incoming SNMP traps",
	})
	p.trapsHandled = promauto.NewCounter(prometheus.CounterOpts{
		Name: "trapex_handled_traps_total",
		Help: "The total number of handled SNMP traps",
	})
	p.trapsDropped = promauto.NewCounter(prometheus.CounterOpts{
		Name: "trapex_dropped_traps_total",
		Help: "The total number of dropped SNMP traps",
	})
	p.trapsIgnored = promauto.NewCounter(prometheus.CounterOpts{
		Name: "trapex_ignored_traps_total",
		Help: "The total number of ignored SNMP traps",
	})
	p.trapsFromV2c = promauto.NewCounter(prometheus.CounterOpts{
		Name: "trapex_v2c_traps_total",
		Help: "The total number of SNMPv2c traps translated",
	})
	p.trapsFromV3 = promauto.NewCounter(prometheus.CounterOpts{
		Name: "trapex_v3_traps_total",
		Help: "The total number of SNMPv3 traps translated",
	})

	exporter := fmt.Sprintf("http://%s/%s", p.listenAddress, p.endpoint)
	p.trapex_log.Info().Str("endpoint", exporter).Msg("Prometheus metrics exporter")

        go exposeMetrics(p.endpoint, p.listenAddress)

	return nil
}

func (p prometheusStats) Inc(metric pluginMeta.Metric) {

/*
	switch metric {
	case pluginMeta.MetricTotal:
		p.trapsTotal.Inc()
	case pluginMeta.MetricHandled:
		p.trapsHandled.Inc()
	case pluginMeta.MetricDropped:
		p.trapsDropped.Inc()
	case pluginMeta.MetricIgnored:
		p.trapsIgnored.Inc()
	case pluginMeta.MetricFromV2c:
		p.trapsFromV2c.Inc()
	case pluginMeta.MetricFromV3:
		p.trapsFromV3.Inc()

	}
*/

}

        func (p prometheusStats)Report() (string, error) {
return "", nil
}

// exposeMetrics
// Allow Prometheus to gather current performance metrics via /metrics URL
func exposeMetrics(endpoint string, listenAddress string) {
	server := http.NewServeMux()
	server.Handle(endpoint, promhttp.Handler())
	http.ListenAndServe(listenAddress, server)
}

var StatsPlugin prometheusStats
