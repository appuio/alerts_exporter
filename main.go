package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	alertscollector "github.com/bastjan/alerts_exporter/internal/alerts_collector"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var host string
var withInhibited, withSilenced, withUnprocessed, withActive bool
var filters stringSliceFlag

func main() {
	flag.StringVar(&host, "host", "localhost:9093", "The host of the Alertmanager")

	flag.BoolVar(&withActive, "with-active", true, "Query for active alerts")
	flag.BoolVar(&withInhibited, "with-inhibited", true, "Query for inhibited alerts")
	flag.BoolVar(&withSilenced, "with-silenced", true, "Query for silenced alerts")
	flag.BoolVar(&withUnprocessed, "with-unprocessed", true, "Query for unprocessed alerts")
	flag.Var(&filters, "filter", "A list of Alertmanager matchers to filter alerts by. Multiple matchers are ANDed.\nUsage example: '--filter slo=\"true\" --filter severity=\"critical\"'")

	flag.Parse()

	ac := client.NewHTTPClientWithConfig(nil, client.DefaultTransportConfig().WithHost(host))

	reg := prometheus.NewRegistry()

	reg.MustRegister(&alertscollector.AlertsCollector{
		API: ac,

		WithActive:      &withActive,
		WithSilenced:    &withSilenced,
		WithInhibited:   &withInhibited,
		WithUnprocessed: &withUnprocessed,
		Filters:         filters,
	})

	// Expose metrics and custom registry via an HTTP server
	// using the HandleFor function. "/metrics" is the usual endpoint for that.
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	log.Println("Listening on `:8080/metrics`")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type stringSliceFlag []string

func (f stringSliceFlag) String() string {
	return fmt.Sprint([]string(f))
}

func (f *stringSliceFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}
