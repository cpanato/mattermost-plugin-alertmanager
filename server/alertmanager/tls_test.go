package alertmanager

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func useHTTPClient(t *testing.T, client *http.Client) {
	t.Helper()
	original := http.DefaultClient
	http.DefaultClient = client
	t.Cleanup(func() { http.DefaultClient = original })
}

func TestHTTPSAndRedirects(t *testing.T) {
	upstream := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v2/status":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = io.WriteString(w, `{"versionInfo":{"version":"test-https"}}`)
		case "/api/v2/alerts", "/api/v2/silences":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = io.WriteString(w, `[]`)
		case "/api/v2/silence/test":
			require.Equal(t, http.MethodDelete, r.Method)
			_, _ = io.WriteString(w, `{}`)
		default:
			http.NotFound(w, r)
		}
	}))
	upstream.StartTLS()
	defer upstream.Close()
	useHTTPClient(t, upstream.Client())
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, upstream.URL+r.URL.RequestURI(), http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	for _, tc := range []struct{ name, url string }{{"direct", upstream.URL}, {"HTTP to HTTPS", redirect.URL}} {
		t.Run(tc.name, func(t *testing.T) {
			status, err := Status(tc.url)
			require.NoError(t, err)
			require.Equal(t, "test-https", status.VersionInfo.Version)
			_, err = ListAlerts(tc.url)
			require.NoError(t, err)
			_, err = ListSilences(tc.url)
			require.NoError(t, err)
			require.NoError(t, ExpireSilence("test", tc.url))
		})
	}
}

func TestCertificateErrorsArePreserved(t *testing.T) {
	upstream := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("invalid certificate must prevent HTTP requests")
	}))
	upstream.Config.ErrorLog = log.New(io.Discard, "", 0)
	upstream.StartTLS()
	defer upstream.Close()
	for _, tc := range []struct {
		name   string
		client *http.Client
	}{{"untrusted CA", &http.Client{Transport: http.DefaultTransport.(*http.Transport).Clone()}}, {"hostname mismatch", upstream.Client()}} {
		t.Run(tc.name, func(t *testing.T) {
			transport := tc.client.Transport.(*http.Transport).Clone()
			if tc.name == "hostname mismatch" {
				transport.TLSClientConfig = transport.TLSClientConfig.Clone()
				transport.TLSClientConfig.ServerName = "wrong.invalid"
			}
			attempts := 0
			tc.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) { attempts++; return transport.RoundTrip(r) })
			defer transport.CloseIdleConnections()
			useHTTPClient(t, tc.client)
			_, err := Status(upstream.URL)
			var verification *tls.CertificateVerificationError
			require.ErrorAs(t, err, &verification, "certificate failures must not become a generic context deadline")
			if tc.name == "untrusted CA" {
				var cause x509.UnknownAuthorityError
				require.ErrorAs(t, err, &cause)
			} else {
				var cause x509.HostnameError
				require.ErrorAs(t, err, &cause)
			}
			require.Equal(t, 1, attempts, "retrying cannot repair a certificate validation failure")
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
