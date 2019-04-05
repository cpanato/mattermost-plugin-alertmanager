package alertmanager

import (
	"encoding/json"
	"net/http"

	"github.com/prometheus/alertmanager/types"
)

// ListAlerts returns a slice of Alert and an error.
func ListAlerts(alertmanagerURL string) ([]*types.Alert, error) {
	resp, err := httpRetry(http.MethodGet, alertmanagerURL+"/api/v2/alerts")
	if err != nil {
		return nil, err
	}

	var alertResponse []*types.Alert
	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	if errDec := dec.Decode(&alertResponse); errDec != nil {
		return nil, errDec
	}

	return alertResponse, err
}
