package healthcheck_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/appuio/alerts_exporter/internal/healthcheck"
	"github.com/go-openapi/runtime"
	"github.com/prometheus/alertmanager/api/v2/client/general"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/stretchr/testify/require"
)

func TestOk(t *testing.T) {
	t.Parallel()

	hc := &healthcheck.HealthCheck{
		GeneralService: &mockClientService{
			OkResponse: &general.GetStatusOK{
				Payload: &models.AlertmanagerStatus{
					VersionInfo: &models.VersionInfo{
						Version: ptr("v0.22.2"),
					},
				},
			},
		},
	}

	req := httptest.NewRecorder()
	hc.HandleHealthz(req, httptest.NewRequest("GET", "/healthz", nil))
	res := req.Result()
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Contains(t, req.Body.String(), `"status":"connected"`)
	require.Contains(t, req.Body.String(), `v0.22.2`)
}

func TestErrResponse(t *testing.T) {
	t.Parallel()

	hc := &healthcheck.HealthCheck{
		GeneralService: &mockClientService{
			Err: errors.New("some error"),
		},
	}

	req := httptest.NewRecorder()
	hc.HandleHealthz(req, httptest.NewRequest("GET", "/healthz", nil))
	res := req.Result()
	require.Equal(t, http.StatusInternalServerError, res.StatusCode)
	require.Contains(t, req.Body.String(), "some error")
}

func TestNilResponse(t *testing.T) {
	t.Parallel()

	hc := &healthcheck.HealthCheck{
		GeneralService: &mockClientService{},
	}

	req := httptest.NewRecorder()
	hc.HandleHealthz(req, httptest.NewRequest("GET", "/healthz", nil))
	res := req.Result()
	require.Equal(t, http.StatusInternalServerError, res.StatusCode)
	require.Contains(t, req.Body.String(), "Nil response")
}

type mockClientService struct {
	OkResponse *general.GetStatusOK
	Err        error
}

var _ general.ClientService = (*mockClientService)(nil)

func (m *mockClientService) GetStatus(*general.GetStatusParams, ...general.ClientOption) (*general.GetStatusOK, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.OkResponse, nil
}

func (m *mockClientService) SetTransport(runtime.ClientTransport) {}

func ptr[T any](t T) *T { return &t }
