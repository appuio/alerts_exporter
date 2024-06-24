package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"

	alertscollector "github.com/appuio/alerts_exporter/internal/alerts_collector"
	"github.com/appuio/alerts_exporter/internal/healthcheck"
	"github.com/appuio/alerts_exporter/internal/saauth"
	openapiclient "github.com/go-openapi/runtime/client"
	alertmanagerclient "github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var listenAddr, healthListenAddr string

var host string
var withInhibited, withSilenced, withUnprocessed, withActive bool
var filters stringSliceFlag

var tlsCert, tlsCertKey, tlsCaCert, tlsServerName string
var tlsInsecure bool
var useTLS bool
var bearerToken string
var k8sBearerTokenAuth bool

func main() {
	flag.StringVar(&listenAddr, "listen-addr", ":8080", "The addr to listen on")
	flag.StringVar(&healthListenAddr, "health-listen-addr", ":8081", "The addr to listen on for the health check endpoint.")

	flag.StringVar(&host, "host", "localhost:9093", "The host of the Alertmanager")

	flag.BoolVar(&useTLS, "tls", false, "Use TLS when connecting to Alertmanager")
	flag.StringVar(&tlsCert, "tls-cert", "", "Path to client certificate for TLS authentication")
	flag.StringVar(&tlsCertKey, "tls-cert-key", "", "Path to client certificate key for TLS authentication")
	flag.StringVar(&tlsCaCert, "tls-ca-cert", "", "Path to CA certificate. System certificates are used if not provided.")
	flag.StringVar(&tlsServerName, "tls-server-name", "", "Server name to verify the hostname on the returned certificates. It must be a substring of either the Common Name or a Subject Alternative Name in the certificate. If empty, the hostname given in the address parameter is used.")
	flag.BoolVar(&tlsInsecure, "insecure", false, "Disable TLS host verification")

	flag.StringVar(&bearerToken, "bearer-token", "", "Bearer token to use for authentication")
	flag.BoolVar(&k8sBearerTokenAuth, "k8s-bearer-token-auth", false, "Use Kubernetes service account bearer token for authentication")

	flag.BoolVar(&withActive, "with-active", true, "Query for active alerts")
	flag.BoolVar(&withInhibited, "with-inhibited", true, "Query for inhibited alerts")
	flag.BoolVar(&withSilenced, "with-silenced", true, "Query for silenced alerts")
	flag.BoolVar(&withUnprocessed, "with-unprocessed", true, "Query for unprocessed alerts")
	flag.Var(&filters, "filter", "A list of Alertmanager matchers to filter alerts by. Multiple matchers are ANDed.\nUsage example: '--filter slo=\"true\" --filter severity=\"critical\"'")

	flag.Parse()

	opts := openapiclient.TLSClientOptions{
		Certificate: tlsCert,
		Key:         tlsCertKey,
		CA:          tlsCaCert,
		ServerName:  tlsServerName,
	}
	if tlsInsecure {
		opts.InsecureSkipVerify = true
		opts.ServerName = ""
	}
	var schemes []string
	if useTLS {
		schemes = []string{"https"}
	}

	hc, err := openapiclient.TLSClient(opts)
	if err != nil {
		log.Fatal(err)
	}

	rt := openapiclient.NewWithClient(host, alertmanagerclient.DefaultBasePath, schemes, hc)

	if bearerToken != "" {
		rt.DefaultAuthentication = openapiclient.BearerToken(bearerToken)
	}
	if k8sBearerTokenAuth {
		sa, err := saauth.NewServiceAccountAuthInfoWriter("", 0)
		if err != nil {
			log.Fatal(err)
		}
		defer sa.Stop()
		rt.DefaultAuthentication = sa
	}

	ac := alertmanagerclient.New(rt, nil)

	reg := prometheus.NewRegistry()

	reg.MustRegister(&alertscollector.AlertsCollector{
		AlertService: ac.Alert,

		WithActive:      &withActive,
		WithSilenced:    &withSilenced,
		WithInhibited:   &withInhibited,
		WithUnprocessed: &withUnprocessed,
		Filters:         filters,
	})

	msm := http.NewServeMux()
	msm.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))

	hsm := http.NewServeMux()
	hsm.HandleFunc("/healthz", healthcheck.HealthCheck{GeneralService: ac.General}.HandleHealthz)

	ms := &http.Server{
		Addr:    listenAddr,
		Handler: msm,
	}

	hs := &http.Server{
		Addr:    healthListenAddr,
		Handler: hsm,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	go func() {
		defer cancel()
		log.Printf("Metrics: Listening on `%s`", listenAddr)
		log.Println("Metrics:", ms.ListenAndServe())
	}()
	go func() {
		defer cancel()
		log.Printf("Healthz: Listening on `%s`", healthListenAddr)
		log.Println("Healthz:", hs.ListenAndServe())
	}()

	var waitShutdown sync.WaitGroup
	waitShutdown.Add(2)
	go func() {
		defer waitShutdown.Done()
		<-ctx.Done()
		ms.Shutdown(context.Background())
	}()
	go func() {
		defer waitShutdown.Done()
		<-ctx.Done()
		hs.Shutdown(context.Background())
	}()

	waitShutdown.Wait()
}

type stringSliceFlag []string

func (f stringSliceFlag) String() string {
	return fmt.Sprint([]string(f))
}

func (f *stringSliceFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}
