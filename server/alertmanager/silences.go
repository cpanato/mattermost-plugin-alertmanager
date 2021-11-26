package alertmanager

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"time"

	"github.com/prometheus/alertmanager/types"
)

// ListSilences returns a slice of Silence and an error.
func ListSilences(alertmanagerURL string) ([]types.Silence, error) {
	resp, err := httpRetry(http.MethodGet, alertmanagerURL+"/api/v2/silences")
	if err != nil {
		return nil, err
	}

	var silencesResponse []types.Silence
	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	if errDec := dec.Decode(&silencesResponse); errDec != nil {
		return nil, errDec
	}

	silences := silencesResponse
	sort.Slice(silences, func(i, j int) bool {
		return silences[i].EndsAt.After(silences[j].EndsAt)
	})

	return silences, err
}

// DeleteSilence delete a silence by ID.
func ExpireSilence(silenceID, alertmanagerURL string) error {
	if silenceID == "" {
		return fmt.Errorf("silence ID cannot be empty")
	}

	expireSilence := fmt.Sprintf("%s/api/v2/silence/%s", alertmanagerURL, silenceID)
	resp, err := httpRetry(http.MethodDelete, expireSilence)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf(string(body))
	}

	return nil
}

// Resolved returns if a silence is reolved by EndsAt
func Resolved(s types.Silence) bool {
	if s.EndsAt.IsZero() {
		return false
	}
	return !s.EndsAt.After(time.Now())
}
