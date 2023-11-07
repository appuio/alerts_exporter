package alertscollector_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	alertscollector "github.com/appuio/alerts_exporter/internal/alerts_collector"
	"github.com/appuio/alerts_exporter/internal/alerts_collector/mock"
)

//go:generate go run github.com/golang/mock/mockgen -destination=./mock/alert_service.go -package mock github.com/prometheus/alertmanager/api/v2/client/alert ClientService

func TestAlertsCollector(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockAlertService := mock.NewMockClientService(ctrl)

	mockAlertService.
		EXPECT().
		GetAlerts(
			gomock.Eq(alert.NewGetAlertsParamsWithContext(context.Background()).WithActive(ptr(true))),
			gomock.Any(),
		).
		Return(&alert.GetAlertsOK{
			Payload: []*models.GettableAlert{
				{
					Alert: models.Alert{
						Labels: map[string]string{
							"alertname": "ImportantAlert",
							"severity":  "critical",
						},
					},
					Status: &models.AlertStatus{
						State:       ptr("active"),
						InhibitedBy: []string{"22a8bdd0-9b1e-4855-b8fb-8c1e18fa434f"},
						SilencedBy:  []string{"d505b8d4-c5ce-466f-abd7-c704864299f5"},
					},
				},
				{
					Alert: models.Alert{
						Labels: map[string]string{
							"alertname": "WhateverHappensHappens",
							"severity":  "low",
						},
					},
				},
			},
		}, nil)

	subject := &alertscollector.AlertsCollector{
		AlertService: mockAlertService,

		WithActive: ptr(true),
	}

	require.NoError(t,
		testutil.CollectAndCompare(subject, strings.NewReader(`
# HELP alerts_exporter_alerts Alerts queried from the Alertmanager API. Alert state can be found in the '_alerts_exporter_alert_state' label.
# TYPE alerts_exporter_alerts gauge
alerts_exporter_alerts{alertname="WhateverHappensHappens",severity="low"} 1
alerts_exporter_alerts{_alerts_exporter_alert_inhibited_by="22a8bdd0-9b1e-4855-b8fb-8c1e18fa434f",_alerts_exporter_alert_silenced_by="d505b8d4-c5ce-466f-abd7-c704864299f5",_alerts_exporter_alert_state="active",alertname="ImportantAlert",severity="critical"} 1
`),
		),
	)
}

func TestAlertsCollector_Err(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAlertService := mock.NewMockClientService(ctrl)

	mockAlertService.EXPECT().GetAlerts(gomock.Any(), gomock.Any()).Return(nil, errors.New("API error"))

	subject := &alertscollector.AlertsCollector{
		AlertService: mockAlertService,
	}

	require.ErrorContains(t,
		testutil.CollectAndCompare(subject, nil),
		"API error",
	)
}

func ptr[T any](t T) *T { return &t }
