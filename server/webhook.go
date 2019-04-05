package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/mattermost/mattermost-server/model"
	"github.com/prometheus/alertmanager/notify"
)

const (
	alertManagerIconURL  = "https://upload.wikimedia.org/wikipedia/en/3/38/Prometheus_software_logo.svg"
	alertManagerUsername = "AlertManager Bot"
)

func (p *Plugin) handleWebhook(w http.ResponseWriter, r *http.Request) {
	p.API.LogInfo("Received alertmanager notification")

	var webhook notify.WebhookMessage
	err := json.NewDecoder(r.Body).Decode(&webhook)
	if err != nil {
		p.API.LogError("failed to decode webhook message", "err", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var fields []*model.SlackAttachmentField

	for _, alert := range webhook.Alerts {
		statusMsg := strings.ToUpper(alert.Status)
		if alert.Status == "firing" {
			statusMsg = fmt.Sprintf(":fire: %s :fire:", strings.ToUpper(alert.Status))
		}
		fields = addFields(fields, "Status", statusMsg, false)
		for k, v := range alert.Annotations {
			fields = addFields(fields, k, v, true)
		}
		for k, v := range alert.Labels {
			fields = addFields(fields, k, v, true)
		}

		fields = addFields(fields, "Start At", durafmt.Parse(time.Since(alert.StartsAt)).String(), false)
		fields = addFields(fields, "Ended At", durafmt.Parse(time.Since(alert.EndsAt)).String(), false)
	}

	title := fmt.Sprintf("[%s](%s)", webhook.Receiver, webhook.ExternalURL)
	attachment := &model.SlackAttachment{
		Title:  title,
		Fields: fields,
		Color:  setColor(webhook.Status),
	}

	post := &model.Post{
		ChannelId: p.ChannelID,
		UserId:    p.BotUserID,
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": alertManagerUsername,
			"override_icon_url": alertManagerIconURL,
		},
	}

	model.ParseSlackAttachment(post, []*model.SlackAttachment{attachment})
	if _, appErr := p.API.CreatePost(post); appErr != nil {
		return
	}
}

func addFields(fields []*model.SlackAttachmentField, title, msg string, short bool) []*model.SlackAttachmentField {
	return append(fields, &model.SlackAttachmentField{
		Title: title,
		Value: msg,
		Short: short,
	})
}

func setColor(impact string) string {
	mapImpactColor := map[string]string{
		"firing":   "#FF0000",
		"resolved": "#008000",
	}

	if val, ok := mapImpactColor[impact]; ok {
		return val
	}

	return "#F0F8FF"
}
