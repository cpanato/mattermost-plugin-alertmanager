package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cpanato/mattermost-plugin-alertmanager/internal/notification"
	"github.com/mattermost/mattermost/server/public/model"
	amtemplate "github.com/prometheus/alertmanager/template"
	"github.com/stretchr/testify/require"
)

const webhook = `{"receiver":"platform","status":"firing","alerts":[{"status":"firing","labels":{"alertname":"InstanceDown","instance":"host-1","severity":"warning"}},{"status":"resolved","labels":{"alertname":"InstanceDown","instance":"host-2","severity":"critical"}}]}`

func TestPreviewUsesPluginRenderer(t *testing.T) {
	dir := t.TempDir()
	body := `alertname={{ .Labels.alertname }}, instance={{ .Labels.instance }}, severity={{ .Labels.severity }}`
	path := filepath.Join(dir, "body.tmpl")
	require.NoError(t, os.WriteFile(path, []byte(body), 0600))
	var out, stderr bytes.Buffer
	require.NoError(t, run([]string{"-message-template", path, "-config-id", "east"}, strings.NewReader(webhook), &out, &stderr))
	var got []*model.MessageAttachment
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Len(t, got, 2)
	var data amtemplate.Data
	require.NoError(t, json.Unmarshal([]byte(webhook), &data))
	tmpl, err := notification.Parse("", body)
	require.NoError(t, err)
	for i, alert := range data.Alerts {
		expected, err := tmpl.Render(notification.AlertData(data, alert, "east"))
		require.NoError(t, err)
		require.Equal(t, expected, got[i])
	}
	require.Equal(t, "#FFCC00", got[0].Color)
	require.Equal(t, "#008000", got[1].Color)
	require.Empty(t, stderr.String())
}

func TestPreviewRejectsInvalidInputWithoutPartialOutput(t *testing.T) {
	for _, tc := range []struct{ name, input, source string }{
		{"bad JSON", "{", ""},
		{"no alerts", "{}", ""},
		{"multiple JSON values", webhook + " {}", ""},
		{"bad syntax", webhook, "{{"},
		{"runtime error", webhook, `{{ .UnknownField }}`},
		{"second alert fails", webhook, `{{ if eq .Status "resolved" }}{{ .UnknownField }}{{ else }}valid{{ end }}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "body.tmpl")
			require.NoError(t, os.WriteFile(path, []byte(tc.source), 0600))
			var out, stderr bytes.Buffer
			require.Error(t, run([]string{"-message-template", path}, strings.NewReader(tc.input), &out, &stderr))
			require.Empty(t, out.String())
		})
	}
}
