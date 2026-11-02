package alertmanager

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStatusReadsBodyAfterHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v2/status", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		time.Sleep(10 * time.Millisecond)
		_, _ = fmt.Fprint(w, `{"uptime":"2026-01-01T00:00:00Z","versionInfo":{"version":"0.34.1"}}`)
	}))
	defer server.Close()
	status, err := Status(server.URL, BasicAuth{})
	require.NoError(t, err)
	require.Equal(t, "0.34.1", status.VersionInfo.Version)
}

func TestClientRetriesServerErrors(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprint(w, `[]`)
	}))
	defer server.Close()
	alerts, err := ListAlerts(server.URL, BasicAuth{})
	require.NoError(t, err)
	require.Empty(t, alerts)
	require.EqualValues(t, 2, calls.Load())
}

func TestClientRejectsPermanentErrors(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Error(w, "not found", http.StatusNotFound)
			}))
			defer server.Close()
			response, err := httpRetry(method, server.URL, BasicAuth{})
			require.ErrorContains(t, err, "404")
			require.Nil(t, response)
			require.EqualValues(t, 1, calls.Load())
		})
	}
}

func TestListSilencesV2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v2/silences", r.URL.Path)
		_, _ = fmt.Fprint(w, `[{"id":"test","matchers":[{"name":"service","value":"api","isRegex":false,"isEqual":true}],"startsAt":"2026-01-01T00:00:00Z","endsAt":"2026-01-02T00:00:00Z","status":{"state":"active"},"createdBy":"tester","comment":"maintenance"}]`)
	}))
	defer server.Close()
	silences, err := ListSilences(server.URL, BasicAuth{})
	require.NoError(t, err)
	require.Len(t, silences, 1)
	require.Equal(t, "test", silences[0].ID)
	require.Equal(t, "api", silences[0].Matchers[0].Value)
	require.Equal(t, "active", silences[0].Status.State)
}

func TestAlertmanagerRequestsUseBasicAuth(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != "alertmanager-user" || password != "alertmanager-password" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		calls.Add(1)
		switch r.URL.Path {
		case "/api/v2/status":
			_, _ = fmt.Fprint(w, `{"versionInfo":{"version":"test"}}`)
		case "/api/v2/alerts", "/api/v2/silences":
			_, _ = fmt.Fprint(w, `[]`)
		case "/api/v2/silence/test":
			require.Equal(t, http.MethodDelete, r.Method)
			_, _ = fmt.Fprint(w, `{}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	auth := BasicAuth{Username: "alertmanager-user", Password: "alertmanager-password"}
	_, err := Status(server.URL, auth)
	require.NoError(t, err)
	_, err = ListAlerts(server.URL, auth)
	require.NoError(t, err)
	_, err = ListSilences(server.URL, auth)
	require.NoError(t, err)
	require.NoError(t, ExpireSilence("test", server.URL, auth))
	require.EqualValues(t, 4, calls.Load())
}
