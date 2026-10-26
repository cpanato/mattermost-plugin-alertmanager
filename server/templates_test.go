package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestConfiguredWebhookTemplates(t *testing.T) {
	for _, tc := range []struct {
		name, title, message string
		warn, compact        bool
	}{
		{name: "default"},
		{name: "custom", title: `{{ .Status | toUpper }} {{ .Labels.alertname }}`, message: `{{ .Annotations.summary }} ({{ .Labels.instance }})`, compact: true},
		{name: "invalid syntax", message: `{{`, warn: true},
		{name: "runtime error", message: `{{ .UnknownField }}`, warn: true},
		{name: "empty output", title: `{{ "" }}`, message: `{{ "" }}`, warn: true},
		{name: "oversized output", message: `{{ .Annotations.large }}`, warn: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := &plugintest.API{}
			api.On("LoadPluginConfiguration", mock.Anything).Run(func(args mock.Arguments) {
				config := args.Get(0).(*configuration)
				config.AlertConfigs = map[string]alertConfig{"east": {Token: "east-token", AlertManagerURL: "https://alertmanager.example.com/", TitleTemplate: tc.title, MessageTemplate: tc.message}}
			}).Return(nil).Once()
			api.On("LogInfo", mock.Anything).Return()
			if tc.warn {
				api.On("LogWarn", mock.Anything, "config_id", "east", "error", mock.Anything).Return().Once()
			}
			api.On("CreatePost", mock.Anything).Run(func(args mock.Arguments) {
				post := args.Get(0).(*model.Post)
				require.Equal(t, "east-channel", post.ChannelId)
				attachments := post.Attachments()
				require.Len(t, attachments, 1)
				a := attachments[0]
				require.Equal(t, "#FFCC00", a.Color)
				require.Contains(t, a.Fallback, "InstanceDown")
				if tc.compact {
					require.Equal(t, "FIRING InstanceDown", a.Title)
					require.Equal(t, "Host unreachable (host-1)", a.Text)
					require.Empty(t, a.Fields)
					require.Empty(t, a.Footer)
				} else {
					require.Equal(t, "FIRING: InstanceDown", a.Title)
					require.NotEmpty(t, a.Fields)
					require.Contains(t, a.Text, "**Host unreachable**")
				}
			}).Return(&model.Post{}, (*model.AppError)(nil)).Once()
			p := &Plugin{BotUserID: "bot", AlertConfigIDChannelID: map[string]string{"east": "east-channel"}}
			p.SetAPI(api)
			require.NoError(t, p.OnConfigurationChange())
			config := p.getConfiguration().AlertConfigs["east"]
			require.Equal(t, "https://alertmanager.example.com", config.AlertManagerURL)
			payload := map[string]any{"status": "firing", "alerts": []map[string]any{{"status": "firing", "labels": map[string]string{"alertname": "InstanceDown", "instance": "host-1", "severity": "warning"}, "annotations": map[string]string{"summary": "Host unreachable"}}}}
			if tc.name == "oversized output" {
				payload["alerts"].([]map[string]any)[0]["annotations"].(map[string]string)["large"] = strings.Repeat("x", 16385)
			}
			data, err := json.Marshal(payload)
			require.NoError(t, err)
			w := httptest.NewRecorder()
			p.ServeHTTP(nil, w, httptest.NewRequest(http.MethodPost, "/api/webhook?token=east-token", strings.NewReader(string(data))))
			require.Equal(t, http.StatusOK, w.Code)
			api.AssertExpectations(t)
		})
	}
}
