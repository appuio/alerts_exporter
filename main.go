package main

import (
	"flag"
	"log"
	"net/http"

	alertscollector "github.com/bastjan/alerts_exporter/internal/alerts_collector"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var host string
var withInhibited, withSilenced, withUnprocessed, withActive bool

func main() {
	flag.StringVar(&host, "host", "localhost:9093", "The host of the alertmanager")
	flag.BoolVar(&withActive, "with-active", false, "Query for active alerts")
	flag.BoolVar(&withInhibited, "with-inhibited", false, "Query for inhibited alerts")
	flag.BoolVar(&withSilenced, "with-silenced", false, "Query for silenced alerts")
	flag.BoolVar(&withUnprocessed, "with-unprocessed", false, "Query for unprocessed alerts")

	flag.Parse()

	ac := client.NewHTTPClientWithConfig(nil, client.DefaultTransportConfig().WithHost("localhost:9093"))

	reg := prometheus.NewRegistry()

	reg.MustRegister(&alertscollector.AlertsCollector{
		API: ac,

		WithActive:      &withActive,
		WithSilenced:    &withSilenced,
		WithInhibited:   &withInhibited,
		WithUnprocessed: &withUnprocessed,
	})

	// Expose metrics and custom registry via an HTTP server
	// using the HandleFor function. "/metrics" is the usual endpoint for that.
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	log.Println("Listening on `:8080/metrics`")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
