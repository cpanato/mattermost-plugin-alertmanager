package alertmanager

import (
	"encoding/json"
	"net/http"
	"time"
)

// StatusResponse is the data returned by Alertmanager about its current status.
type StatusResponse struct {
	Uptime      time.Time `json:"uptime"`
	VersionInfo struct {
		Branch    string `json:"branch"`
		BuildDate string `json:"buildDate"`
		BuildUser string `json:"buildUser"`
		GoVersion string `json:"goVersion"`
		Revision  string `json:"revision"`
		Version   string `json:"version"`
	} `json:"versionInfo"`
}

// Status returns a StatusResponse or an error.
func Status(alertmanagerURL string) (StatusResponse, error) {
	var statusResponse StatusResponse

	resp, err := httpRetry(http.MethodGet, alertmanagerURL+"/api/v2/status")
	if err != nil {
		return statusResponse, err
	}

	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	if err := dec.Decode(&statusResponse); err != nil {
		return statusResponse, err
	}

	return statusResponse, nil
}
