// Command alertmanager-template previews the same notifications as the plugin.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/cpanato/mattermost-plugin-alertmanager/internal/notification"
	"github.com/mattermost/mattermost/server/public/model"
	amtemplate "github.com/prometheus/alertmanager/template"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("alertmanager-template", flag.ContinueOnError)
	flags.SetOutput(stderr)
	input := flags.String("input", "-", "webhook JSON file, or - for stdin")
	titleFile := flags.String("title-template", "", "optional title template file")
	messageFile := flags.String("message-template", "", "optional message template file")
	configID := flags.String("config-id", "preview", "configuration ID exposed to templates")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	load := func(path string) (string, error) {
		if path == "" {
			return "", nil
		}
		f, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer func() { _ = f.Close() }()
		b, err := io.ReadAll(io.LimitReader(f, notification.MaxTemplateBytes+1))
		if err != nil {
			return "", err
		}
		if len(b) > notification.MaxTemplateBytes {
			return "", fmt.Errorf("%s exceeds %d template bytes", path, notification.MaxTemplateBytes)
		}
		return string(b), nil
	}
	title, err := load(*titleFile)
	if err != nil {
		return err
	}
	message, err := load(*messageFile)
	if err != nil {
		return err
	}
	templates, err := notification.Parse(title, message)
	if err != nil {
		return err
	}
	reader := stdin
	if *input != "-" {
		f, err := os.Open(*input)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		reader = f
	}
	var data amtemplate.Data
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("webhook JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("webhook input must contain exactly one JSON value")
	}
	if len(data.Alerts) == 0 {
		return fmt.Errorf("webhook contains no alerts")
	}
	attachments := make([]*model.MessageAttachment, 0, len(data.Alerts))
	for i, alert := range data.Alerts {
		attachment, err := templates.Render(notification.AlertData(data, alert, *configID))
		if err != nil {
			return fmt.Errorf("alert %d: %w", i, err)
		}
		attachments = append(attachments, attachment)
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(attachments)
}
