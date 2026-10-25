package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cpanato/mattermost-plugin-alertmanager/server/alertmanager"
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

func TestWebhookRoutesByToken(t *testing.T) {
	for _, config := range []alertConfig{
		{ID: "east", Token: "east-token", Channel: "east-channel"},
		{ID: "west", Token: "west-token", Channel: "west-channel"},
	} {
		t.Run(config.ID, func(t *testing.T) {
			api := &plugintest.API{}
			api.On("LogInfo", mock.Anything).Return()
			api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
				return post.ChannelId == config.Channel
			})).Return(&model.Post{}, (*model.AppError)(nil)).Times(20)
			p := &Plugin{BotUserID: "bot", AlertConfigIDChannelID: map[string]string{"east": "east-channel", "west": "west-channel"}}
			p.SetAPI(api)
			p.setConfiguration(&configuration{AlertConfigs: map[string]alertConfig{
				"east": {ID: "east", Token: "east-token"}, "west": {ID: "west", Token: "west-token"},
			}})
			for i := 0; i < 20; i++ {
				w := httptest.NewRecorder()
				p.ServeHTTP(nil, w, httptest.NewRequest(http.MethodPost, "/api/webhook?token="+config.Token, strings.NewReader(`{"status":"firing","alerts":[{"status":"firing"}]}`)))
				require.Equal(t, http.StatusOK, w.Code)
			}
			api.AssertExpectations(t)
		})
	}
}

func TestSilenceDuration(t *testing.T) {
	for _, state := range []string{"active", "pending"} {
		t.Run(state, func(t *testing.T) {
			silence := alertmanager.Silence{StartsAt: time.Now().Add(-time.Hour), EndsAt: time.Now().Add(time.Hour)}
			silence.Status.State = state
			if state == "pending" {
				silence.StartsAt = time.Now().Add(10 * time.Minute)
			}
			attachment := ConvertSilenceToMessageAttachment(silence, alertConfig{}, "", ":8065")
			var duration string
			for _, field := range attachment.Fields {
				if value, ok := field.Value.(string); ok && strings.Contains(value, "**Ends:") {
					duration = value
				}
			}
			require.Contains(t, duration, "**Ends:** in ")
			require.NotContains(t, duration, "-59")
			if state == "pending" {
				require.Contains(t, duration, "**Starts:** in ")
			} else {
				require.Contains(t, duration, " ago")
			}
		})
	}
}

func TestExpireActionRejectsMissingContext(t *testing.T) {
	for _, body := range []string{`null`, `{}`, `{"context":null}`, `{"context":`} {
		t.Run(body, func(t *testing.T) {
			api := &plugintest.API{}
			api.On("LogInfo", mock.Anything).Return()
			p := &Plugin{}
			p.SetAPI(api)
			w := httptest.NewRecorder()
			require.NotPanics(t, func() {
				p.handleExpireAction(w, httptest.NewRequest(http.MethodPost, "/api/expire", strings.NewReader(body)), alertConfig{})
			})
			require.Contains(t, w.Body.String(), "We could not decode the action")
			api.AssertExpectations(t)
		})
	}
}
