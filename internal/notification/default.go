package notification

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/mattermost/mattermost/server/public/model"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func addFields(fields []*model.MessageAttachmentField, title, msg string, short bool) []*model.MessageAttachmentField {
	return append(fields, &model.MessageAttachmentField{Title: title, Value: msg, Short: model.SlackCompatibleBool(short)})
}

// Default renders the built-in detailed alert format.
func Default(data Data) *model.MessageAttachment {
	alert := data.Alert
	configID, externalURL, receiver := data.ConfigID, data.ExternalURL, data.Receiver
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
	footer := "Config: " + configID
	if receiver != "" {
		footer += " · Receiver: " + receiver
	}
	return &model.MessageAttachment{
		Fallback: strings.TrimSpace(title + "\n" + strings.Join(text, "\n\n")),
		Title:    title, Text: strings.Join(text, "\n\n"), Fields: fields,
		Color: Color(alert.Status, alert.Labels["severity"]), Footer: footer,
	}
}
