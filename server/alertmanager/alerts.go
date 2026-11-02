package alertmanager

import (
	"encoding/json"
	"net/http"

	"github.com/prometheus/alertmanager/alert"
)

// ListAlerts returns a slice of Alert and an error.
func ListAlerts(alertmanagerURL string, auth BasicAuth) ([]*alert.Alert, error) {
	resp, err := httpRetry(http.MethodGet, alertmanagerURL+"/api/v2/alerts", auth)
	if err != nil {
		return nil, err
	}

	var alertResponse []*alert.Alert
	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	if errDec := dec.Decode(&alertResponse); errDec != nil {
		return nil, errDec
	}

	return alertResponse, err
}
