package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/hako/durafmt"
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
		attachments = append(attachments, convertAlertToAttachment(alertConfig, alert, message.ExternalURL, message.Receiver))
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
	title := strings.ToUpper(alert.Status)
	if name := alert.Labels["alertname"]; name != "" {
		title += ": " + name
	}
	var text []string
	if summary := alert.Annotations["summary"]; summary != "" {
		text = append(text, "**"+summary+"**")
	}
	if description := alert.Annotations["description"]; description != "" {
		text = append(text, description)
	}
	annotations := make([]string, 0, len(alert.Annotations))
	for k := range alert.Annotations {
		if k != "summary" && k != "description" {
			annotations = append(annotations, k)
		}
	}
	sort.Strings(annotations)
	for _, k := range annotations {
		text = append(text, fmt.Sprintf("**%s:** %s", cases.Title(language.Und, cases.NoLower).String(k), alert.Annotations[k]))
	}
	var fields []*model.MessageAttachmentField
	var timing []string
	if !alert.StartsAt.IsZero() {
		timing = append(timing, fmt.Sprintf("**Started:** %s (%s ago)", alert.StartsAt.UTC().Format(time.RFC3339), durafmt.Parse(time.Since(alert.StartsAt).Truncate(time.Second)).LimitFirstN(2).String()))
	}
	if alert.Status == "resolved" && !alert.EndsAt.IsZero() {
		timing = append(timing, fmt.Sprintf("**Ended:** %s (%s ago)", alert.EndsAt.UTC().Format(time.RFC3339), durafmt.Parse(time.Since(alert.EndsAt).Truncate(time.Second)).LimitFirstN(2).String()))
	}
	if len(timing) > 0 {
		fields = addFields(fields, "Timing", strings.Join(timing, "\n"), true)
	}
	var labelLines []string
	labels := make([]string, 0, len(alert.Labels))
	for k := range alert.Labels {
		labels = append(labels, k)
	}
	sort.Strings(labels)
	for _, k := range labels {
		labelLines = append(labelLines, fmt.Sprintf("**%s:** %s", cases.Title(language.Und, cases.NoLower).String(k), alert.Labels[k]))
	}
	if len(labelLines) > 0 {
		fields = addFields(fields, "Labels", strings.Join(labelLines, "\n"), true)
	}
	var links []string
	if alert.GeneratorURL != "" {
		links = append(links, fmt.Sprintf("[Prometheus](%s)", alert.GeneratorURL))
	}
	if externalURL != "" {
		links = append(links, fmt.Sprintf("[Alertmanager](%s)", externalURL))
	}
	if len(links) > 0 {
		fields = addFields(fields, "", strings.Join(links, " · "), false)
	}
	footer := "Config: " + config.ID
	if receiver != "" {
		footer += " · Receiver: " + receiver
	}
	return &model.MessageAttachment{
		Title: title, Text: strings.Join(text, "\n\n"), Fields: fields,
		Color: setColor(alert.Status, alert.Labels["severity"]), Footer: footer,
	}
}
