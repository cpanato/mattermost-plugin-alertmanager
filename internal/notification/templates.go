// Package notification renders Mattermost alerts for the plugin and preview CLI.
package notification

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/mattermost/mattermost/server/public/model"
	amtemplate "github.com/prometheus/alertmanager/template"
)

// MaxTemplateBytes limits each template source and rendered custom field.
const MaxTemplateBytes = 16 * 1024

// Data describes one alert and its enclosing webhook. Credentials are excluded.
type Data struct {
	amtemplate.Alert
	Receiver          string
	ExternalURL       string
	GroupLabels       amtemplate.KV
	CommonLabels      amtemplate.KV
	CommonAnnotations amtemplate.KV
	RouteLabels       amtemplate.KV
	ConfigID          string
}

// AlertData keeps each alert's status separate from the group's firing status.
func AlertData(message amtemplate.Data, alert amtemplate.Alert, configID string) Data {
	return Data{Alert: alert, Receiver: message.Receiver, ExternalURL: message.ExternalURL,
		GroupLabels: message.GroupLabels, CommonLabels: message.CommonLabels,
		CommonAnnotations: message.CommonAnnotations, RouteLabels: message.RouteLabels, ConfigID: configID}
}

// Templates is immutable after parsing and safe for concurrent rendering.
// A nil *Templates renders the built-in format.
type Templates struct{ title, message *template.Template }

// Parse compiles optional per-alert title and body templates. Blank sources keep
// the built-in format. In custom mode, omitted fields retain their default text.
func Parse(title, message string) (*Templates, error) {
	funcs := template.FuncMap{}
	for name, fn := range amtemplate.DefaultFuncs {
		// Alertmanager binds this function dynamically to its group data. Our
		// per-alert context exposes received route labels directly as .RouteLabels.
		if name != "routeLabels" {
			funcs[name] = fn
		}
	}
	parse := func(name, source string) (*template.Template, error) {
		if len(source) > MaxTemplateBytes {
			return nil, fmt.Errorf("%s template exceeds %d bytes", name, MaxTemplateBytes)
		}
		if strings.TrimSpace(source) == "" {
			return nil, nil
		}
		t, err := template.New(name).Option("missingkey=zero").Funcs(funcs).Parse(source)
		if err != nil {
			return nil, fmt.Errorf("%s template: %w", name, err)
		}
		return t, nil
	}
	titleTemplate, err := parse("title", title)
	if err != nil {
		return nil, err
	}
	messageTemplate, err := parse("message", message)
	if err != nil {
		return nil, err
	}
	if titleTemplate == nil && messageTemplate == nil {
		return nil, nil
	}
	return &Templates{title: titleTemplate, message: messageTemplate}, nil
}

// Render returns an error for invalid execution or output. The plugin falls back
// to Default on error; the CLI reports it so administrators can fix the source.
func (t *Templates) Render(data Data) (*model.MessageAttachment, error) {
	attachment := Default(data)
	if t == nil {
		return attachment, nil
	}
	render := func(tmpl *template.Template, defaultText string) (string, error) {
		if tmpl == nil {
			return defaultText, nil
		}
		output := &limitedBuffer{}
		if err := tmpl.Execute(output, data); err != nil {
			return "", fmt.Errorf("%s template: %w", tmpl.Name(), err)
		}
		return strings.TrimSpace(output.String()), nil
	}
	title, err := render(t.title, attachment.Title)
	if err != nil {
		return nil, err
	}
	message, err := render(t.message, attachment.Text)
	if err != nil {
		return nil, err
	}
	if title == "" && message == "" {
		return nil, errors.New("templates rendered an empty title and message")
	}
	return &model.MessageAttachment{Title: title, Text: message, Color: attachment.Color, Fallback: strings.TrimSpace(title + "\n" + message)}, nil
}

type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if len(p) > MaxTemplateBytes-b.Len() {
		return 0, fmt.Errorf("rendered output exceeds %d bytes", MaxTemplateBytes)
	}
	return b.Buffer.Write(p)
}

func (b *limitedBuffer) WriteString(s string) (int, error) {
	if len(s) > MaxTemplateBytes-b.Len() {
		return 0, fmt.Errorf("rendered output exceeds %d bytes", MaxTemplateBytes)
	}
	return b.Buffer.WriteString(s)
}
