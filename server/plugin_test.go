package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/prometheus/alertmanager/template"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestConfigurationClone(t *testing.T) {
	original := &configuration{AlertConfigs: map[string]alertConfig{"0": {Token: "original"}}}
	clone := original.Clone()
	clone.AlertConfigs["0"] = alertConfig{Token: "changed"}
	require.Equal(t, "original", original.AlertConfigs["0"].Token)
}

func TestConfigurationBeforeActivation(t *testing.T) {
	api := &plugintest.API{}
	api.On("LoadPluginConfiguration", mock.Anything).Return(nil)
	p := &Plugin{}
	p.SetAPI(api)
	require.NoError(t, p.OnConfigurationChange())
	require.Nil(t, p.client)
	api.AssertExpectations(t)
}

func TestWebhook(t *testing.T) {
	for _, tc := range []struct {
		name, token, body string
		postErr           *model.AppError
		status            int
	}{
		{name: "invalid token", token: "wrong", body: `{}`, status: http.StatusBadRequest},
		{name: "invalid JSON", token: "secret", body: `{`, status: http.StatusBadRequest},
		{name: "empty payload", token: "secret", body: `{}`, status: http.StatusBadRequest},
		{name: "post failed", token: "secret", body: `{"version":"4","status":"firing","alerts":[{"status":"firing"}]}`, postErr: &model.AppError{Message: "database unavailable"}, status: http.StatusInternalServerError},
		{name: "delivered", token: "secret", body: `{"version":"4","status":"firing","alerts":[{"status":"firing"}]}`, status: http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := &plugintest.API{}
			api.On("LogInfo", mock.Anything).Maybe()
			api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Maybe()
			if tc.name == "post failed" || tc.name == "delivered" {
				api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
					return post.ChannelId == "channel" && post.UserId == "bot" && len(post.Attachments()) == 1
				})).Return(&model.Post{}, tc.postErr).Once()
			}
			p := &Plugin{BotUserID: "bot", AlertConfigIDChannelID: map[string]string{"0": "channel"}}
			p.SetAPI(api)
			p.setConfiguration(&configuration{AlertConfigs: map[string]alertConfig{"0": {ID: "0", Token: "secret"}}})
			w := httptest.NewRecorder()
			p.ServeHTTP(nil, w, httptest.NewRequest(http.MethodPost, "/api/webhook?token="+tc.token, strings.NewReader(tc.body)))
			require.Equal(t, tc.status, w.Code)
			api.AssertExpectations(t)
		})
	}
}

func TestConvertAlertDoesNotMutateLabels(t *testing.T) {
	alert := template.Alert{Labels: template.KV{"alertname": "test"}}
	require.NotEmpty(t, ConvertAlertToFields(alertConfig{ID: "0"}, alert, "", ""))
	require.Equal(t, template.KV{"alertname": "test"}, alert.Labels)
	require.NotPanics(t, func() { ConvertAlertToFields(alertConfig{ID: "0"}, template.Alert{}, "", "") })
}
