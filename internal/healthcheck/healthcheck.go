package healthcheck

import (
	"encoding/json"
	"net/http"

	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/client/general"
	"github.com/prometheus/alertmanager/api/v2/models"
)

// HealthCheck is a health check handler for the Alertmanager API.
type HealthCheck struct {
	GeneralService general.ClientService
}

// HandleHealthz handles a health check request.
// It returns a JSON response with the status of the Alertmanager API or an error if the client returns an error or if receiving a nil response.
func (h HealthCheck) HandleHealthz(res http.ResponseWriter, req *http.Request) {
	ams, err := h.GeneralService.GetStatus(general.NewGetStatusParamsWithContext(req.Context()))
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if ams == nil || ams.Payload == nil {
		http.Error(res, "Nil response from Alertmanager", http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(res).Encode(response{
		Status:              "connected",
		AlertmanagerCluster: ams.Payload.Cluster,
		AlertmanagerVersion: ams.Payload.VersionInfo,
		AlertmanagerUptime:  ams.Payload.Uptime,
	}); err != nil {
		http.Error(res, "Encoding error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

type response struct {
	Status              string                `json:"status"`
	AlertmanagerCluster *models.ClusterStatus `json:"alertmanager_cluster"`
	AlertmanagerVersion *models.VersionInfo   `json:"alertmanager_version"`
	AlertmanagerUptime  *strfmt.DateTime      `json:"alertmanager_uptime"`
}
