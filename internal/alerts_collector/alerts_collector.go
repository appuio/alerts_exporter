package alertscollector

import (
	"context"
	"log"
	"strings"

	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/slices"
)

func newDesc(labels []string) *prometheus.Desc {
	return prometheus.NewDesc(
		"alerts_exporter_alerts",
		"Alertmanager alerts",
		labels,
		nil,
	)
}

type AlertsCollector struct {
	API *client.AlertmanagerAPI

	WithInhibited, WithSilenced, WithUnprocessed, WithActive *bool

	Filters []string
}

var _ prometheus.Collector = &AlertsCollector{}

// Describe implements prometheus.Collector.
// Does not send any description and thus makes the collector unchecked.
func (o *AlertsCollector) Describe(_ chan<- *prometheus.Desc) {}

func (o *AlertsCollector) Collect(ch chan<- prometheus.Metric) {
	p := alert.NewGetAlertsParamsWithContext(context.Background()).
		WithActive(o.WithActive).
		WithSilenced(o.WithSilenced).
		WithUnprocessed(o.WithUnprocessed).
		WithInhibited(o.WithInhibited).
		WithFilter(o.Filters)

	as, err := o.API.Alert.GetAlerts(p)

	if err != nil {
		ch <- prometheus.NewInvalidMetric(newDesc([]string{}), err)
		log.Print("Error querying Alertmanager", err)
		return
	}

	for _, a := range as.Payload {
		if a.Status.State != nil {
			a.Labels["_alerts_exporter_alert_status"] = *a.Status.State
		}
		if len(a.Status.InhibitedBy) > 0 {
			a.Labels["_alerts_exporter_alert_inhibited_by"] = strings.Join(a.Status.InhibitedBy, ",")
		}
		if len(a.Status.SilencedBy) > 0 {
			a.Labels["_alerts_exporter_alert_silenced_by"] = strings.Join(a.Status.SilencedBy, ",")
		}

		k, v := pairs(a.Labels)

		ch <- prometheus.MustNewConstMetric(
			newDesc(k),
			prometheus.GaugeValue,
			1,
			v...,
		)
	}
}

func pairs(m map[string]string) (keys []string, values []string) {
	p := make([][2]string, 0, len(m))

	for k, v := range m {
		p = append(p, [2]string{k, v})
	}

	slices.SortFunc(p, func(a, b [2]string) int {
		return 10*strings.Compare(a[0], b[0]) + strings.Compare(a[1], b[1])
	})

	k := make([]string, len(p))
	v := make([]string, len(p))

	for i := range p {
		k[i] = p[i][0]
		v[i] = p[i][1]
	}

	return k, v
}
