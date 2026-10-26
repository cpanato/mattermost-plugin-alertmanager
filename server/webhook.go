package main

import (
	"encoding/json"
	"net/http"

	"github.com/cpanato/mattermost-plugin-alertmanager/internal/notification"
	"github.com/prometheus/alertmanager/template"

	"github.com/mattermost/mattermost/server/public/model"
)

func (p *Plugin) handleWebhook(w http.ResponseWriter, r *http.Request, alertConfig alertConfig) {
	p.API.LogInfo("Received alertmanager notification")

	// Decode the public webhook data without importing Alertmanager's notifier
	// implementation and its transport/exporter dependencies.
	var message template.Data
	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		p.API.LogError("failed to decode webhook message", "err", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(message.Alerts) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	attachments := make([]*model.MessageAttachment, 0, len(message.Alerts))
	for _, alert := range message.Alerts {
		data := notification.AlertData(message, alert, alertConfig.ID)
		attachment, err := alertConfig.templates.Render(data)
		if err != nil {
			p.API.LogWarn("Failed to render notification template; using default alert format", "config_id", alertConfig.ID, "error", err.Error())
			attachment = convertAlertToAttachment(alertConfig, alert, message.ExternalURL, message.Receiver)
		}
		attachments = append(attachments, attachment)
	}

	p.channelLock.RLock()
	channelID := p.AlertConfigIDChannelID[alertConfig.ID]
	p.channelLock.RUnlock()
	if channelID == "" {
		http.Error(w, "alert channel is unavailable", http.StatusServiceUnavailable)
		return
	}
	post := &model.Post{
		ChannelId: channelID,
		UserId:    p.BotUserID,
	}

	model.ParseMessageAttachment(post, attachments)
	if _, appErr := p.API.CreatePost(post); appErr != nil {
		p.API.LogError("failed to create alert post", "err", appErr.Error())
		http.Error(w, "failed to create alert post", http.StatusInternalServerError)
		return
	}
}

func addFields(fields []*model.MessageAttachmentField, title, msg string, short bool) []*model.MessageAttachmentField {
	return append(fields, &model.MessageAttachmentField{
		Title: title,
		Value: msg,
		Short: model.SlackCompatibleBool(short),
	})
}

func convertAlertToAttachment(config alertConfig, alert template.Alert, externalURL, receiver string) *model.MessageAttachment {
	return notification.Default(notification.Data{Alert: alert, ExternalURL: externalURL, Receiver: receiver, ConfigID: config.ID})
}
