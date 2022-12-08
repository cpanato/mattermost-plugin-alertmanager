package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/pkg/errors"
	"github.com/prometheus/alertmanager/types"

	"github.com/mattermost/mattermost-plugin-api/experimental/command"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"

	"github.com/cpanato/mattermost-plugin-alertmanager/server/alertmanager"
)

const (
	actionHelp  = "help"
	actionAbout = "about"

	helpMsg = `run:
	/alertmanager alerts - to list the existing alerts
	/alertmanager silences - to list the existing silences
	/alertmanager expire_silence - to expire a silence
	/alertmanager status - to list the version and uptime of the Alertmanager instance
	/alertmanager help - display Slash Command help text"
	/alertmanager about - display build information
	`
)

func (p *Plugin) getCommand() (*model.Command, error) {
	iconData, err := command.GetIconData(p.API, "assets/alertmanager-logo.svg")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get icon data")
	}

	return &model.Command{
		Trigger:              "alertmanager",
		AutoComplete:         true,
		AutoCompleteDesc:     fmt.Sprintf("Available commands: status, alerts, silences, expire_silence, %s, %s", actionHelp, actionAbout),
		AutoCompleteHint:     "[command]",
		AutocompleteData:     getAutocompleteData(),
		AutocompleteIconData: iconData,
	}, nil
}

func getAutocompleteData() *model.AutocompleteData {
	root := model.NewAutocompleteData("alertmanager", "[command]", fmt.Sprintf("Available commands: status, alerts, silences, expire_silence, %s, %s", actionHelp, actionAbout))

	alerts := model.NewAutocompleteData("alerts", "", "List the existing alerts")
	root.AddCommand(alerts)

	silences := model.NewAutocompleteData("silences", "", "List the existing silences")
	root.AddCommand(silences)

	expireSilence := model.NewAutocompleteData("expire_silence", "[configuration number] [silence ID]", "TODList the version and uptime of the Alertmanager instance")
	expireSilence.AddTextArgument("The number of the alert configuration", "[configuration number]", "")
	expireSilence.AddTextArgument("The ID of the silence to expire", "[silence ID]", "")
	root.AddCommand(expireSilence)

	status := model.NewAutocompleteData("status", "", "List the version and uptime of the Alertmanager instance")
	root.AddCommand(status)

	help := model.NewAutocompleteData(actionHelp, "", "Display Slash Command help text")
	root.AddCommand(help)

	info := command.BuildInfoAutocomplete(actionAbout)
	root.AddCommand(info)

	return root
}

func (p *Plugin) postCommandResponse(args *model.CommandArgs, text string) {
	post := &model.Post{
		UserId:    p.BotUserID,
		ChannelId: args.ChannelId,
		RootId:    args.RootId,
		Message:   text,
	}
	_ = p.API.SendEphemeralPost(args.UserId, post)
}

func (p *Plugin) ExecuteCommand(_ *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	msg := p.executeCommand(args)
	if msg != "" {
		p.postCommandResponse(args, msg)
	}

	return &model.CommandResponse{}, nil
}

func (p *Plugin) executeCommand(args *model.CommandArgs) string {
	split := strings.Fields(args.Command)
	cmd := split[0]
	action := ""
	if len(split) > 1 {
		action = strings.TrimSpace(split[1])
	}

	if cmd != "/alertmanager" {
		return ""
	}

	if action == "" {
		return "Missing command, please run `/alertmanager help` to check all commands available."
	}

	var msg string
	var err error
	switch action {
	case "alerts":
		msg, err = p.handleAlert(args)
	case "status":
		msg, err = p.handleStatus(args)
	case "silences":
		msg, err = p.handleListSilences(args)
	case "expire_silence":
		msg, err = p.handleExpireSilence(args)
	case actionAbout:
		msg, err = command.BuildInfo(Manifest)
	case actionHelp:
		msg = helpMsg
	default:
		msg = helpMsg
	}

	if err != nil {
		return err.Error()
	}

	return msg
}

func (p *Plugin) handleAlert(args *model.CommandArgs) (string, error) {
	configuration := p.getConfiguration()
	var alertsCount = 0
	var errors []string

	for _, alertConfig := range configuration.AlertConfigs {
		alerts, err := alertmanager.ListAlerts(alertConfig.AlertManagerURL)
		if err != nil {
			errors = append(errors, fmt.Sprintf("AlertManagerURL %q: failed to list alerts... %v", alertConfig.AlertManagerURL, err))
			continue
		}
		if len(alerts) == 0 {
			continue
		}
		alertsCount += len(alerts)

		attachments := make([]*model.SlackAttachment, 0)
		for _, alert := range alerts {
			var fields []*model.SlackAttachmentField
			fields = addFields(fields, "Status", string(alert.Status()), false)
			for k, v := range alert.Annotations {
				fields = addFields(fields, string(k), string(v), true)
			}
			for k, v := range alert.Labels {
				fields = addFields(fields, string(k), string(v), true)
			}
			fields = addFields(fields, "Resolved", strconv.FormatBool(alert.Resolved()), false)
			fields = addFields(fields, "Start At", alert.StartsAt.String(), false)
			fields = addFields(fields, "Ended At", alert.EndsAt.String(), false)
			fields = addFields(fields, "AlertManagerPluginId", alertConfig.ID, false)
			attachment := &model.SlackAttachment{
				Title:  alert.Name(),
				Fields: fields,
				Color:  setColor(string(alert.Status())),
			}
			attachments = append(attachments, attachment)
		}

		post := &model.Post{
			ChannelId: p.ChannelIds[alertConfig.Channel],
			UserId:    p.BotUserID,
			RootId:    args.RootId,
		}

		model.ParseSlackAttachment(post, attachments)
		if _, appErr := p.API.CreatePost(post); appErr != nil {
			errors = append(errors, fmt.Sprintf("Channel %q: Error creating the Alert post", alertConfig.Channel))
			continue
		}
	}

	if len(errors) > 0 {
		return strings.Join(errors, "\n"), nil
	}

	if alertsCount == 0 {
		return "No alerts right now! :tada:", nil
	}

	return "", nil
}

func (p *Plugin) handleStatus(args *model.CommandArgs) (string, error) {
	configuration := p.getConfiguration()

	var errors []string
	for _, alertConfig := range configuration.AlertConfigs {
		status, err := alertmanager.Status(alertConfig.AlertManagerURL)
		if err != nil {
			errors = append(errors, fmt.Sprintf("AlertManagerURL %q: failed to get status... %v", alertConfig.AlertManagerURL, err))
			continue
		}

		uptime := durafmt.Parse(time.Since(status.Uptime)).String()
		var fields []*model.SlackAttachmentField
		fields = addFields(fields, "AlertManager Version ", status.VersionInfo.Version, false)
		fields = addFields(fields, "AlertManager Uptime", uptime, false)

		attachment := &model.SlackAttachment{
			Fields: fields,
		}

		post := &model.Post{
			ChannelId: p.ChannelIds[alertConfig.Channel],
			UserId:    p.BotUserID,
			RootId:    args.RootId,
		}

		model.ParseSlackAttachment(post, []*model.SlackAttachment{attachment})
		if _, appErr := p.API.CreatePost(post); appErr != nil {
			errors = append(errors, fmt.Sprintf("Channel %q: Error creating the Status post", alertConfig.Channel))
			continue
		}
	}

	if len(errors) > 0 {
		return strings.Join(errors, "\n"), nil
	}

	if len(configuration.AlertConfigs) == 0 {
		return "No alert managers are configured!", nil
	}

	return "", nil
}

func (p *Plugin) handleListSilences(args *model.CommandArgs) (string, error) {
	configuration := p.getConfiguration()
	var errors []string
	var silencesCount = 0
	var pendingSilencesCount = 0

	config := p.API.GetConfig()
	siteURLPort := *config.ServiceSettings.ListenAddress

	for _, alertConfig := range configuration.AlertConfigs {
		silences, err := alertmanager.ListSilences(alertConfig.AlertManagerURL)
		if err != nil {
			errors = append(errors, fmt.Sprintf("AlertManagerURL %q: failed to get silences... %v", alertConfig.AlertManagerURL, err))
			continue
		}
		if len(silences) == 0 {
			continue
		}
		silencesCount += len(silences)

		attachments := make([]*model.SlackAttachment, 0)
		for _, silence := range silences {
			attachment := ConvertSilenceToSlackAttachment(silence, alertConfig, args.UserId, siteURLPort)
			if attachment != nil {
				attachments = append(attachments, attachment)
			}
		}

		if len(attachments) == 0 {
			continue
		}
		pendingSilencesCount += len(attachments)

		post := &model.Post{
			ChannelId: p.ChannelIds[alertConfig.Channel],
			UserId:    p.BotUserID,
			RootId:    args.RootId,
		}

		model.ParseSlackAttachment(post, attachments)
		if _, appErr := p.API.CreatePost(post); appErr != nil {
			errors = append(errors, fmt.Sprintf("Channel %q: Error creating the Alert post", alertConfig.Channel))
			continue
		}
	}

	if silencesCount == 0 {
		return "No silences right now.", nil
	}

	if pendingSilencesCount == 0 {
		return "No active or pending silences right now.", nil
	}

	if len(errors) > 0 {
		return strings.Join(errors, "\n"), nil
	}

	return "", nil
}

func (p *Plugin) handleExpireSilence(args *model.CommandArgs) (string, error) {
	split := strings.Fields(args.Command)
	var parameters []string
	if len(split) > 2 {
		parameters = split[2:]
	}

	if len(parameters) != 2 {
		return "Command requires 2 parameters: alert configuration number and silence ID", nil
	}

	configuration := p.getConfiguration()

	if config, ok := configuration.AlertConfigs[parameters[0]]; ok {
		err := alertmanager.ExpireSilence(parameters[1], config.AlertManagerURL)
		if err != nil {
			return "", errors.Wrap(err, "failed to expire the silence")
		}
	} else {
		return fmt.Sprintf("Alert configuration %s not found", parameters[0]), nil
	}

	return fmt.Sprintf("Silence %s expired.", parameters[1]), nil
}

func ConvertSilenceToSlackAttachment(silence types.Silence, config alertConfig, userID, siteURLPort string) *model.SlackAttachment {
	if string(silence.Status.State) == "expired" {
		return nil
	}
	var fields []*model.SlackAttachmentField
	var emoji, matchers, duration string
	for _, m := range silence.Matchers {
		if m.Name == "alertname" {
			fields = addFields(fields, "Alert Name", m.Value, false)
		} else {
			matchers += fmt.Sprintf(`%s="%s"`, m.Name, m.Value)
		}
	}
	fields = addFields(fields, "State", string(silence.Status.State), false)
	fields = addFields(fields, "Matchers", matchers, false)
	resolved := alertmanager.Resolved(silence)
	if !resolved {
		emoji = "🔕"
		duration = fmt.Sprintf(
			"**Started**: %s ago\n**Ends:** %s\n",
			durafmt.Parse(time.Since(silence.StartsAt)),
			durafmt.Parse(time.Since(silence.EndsAt)),
		)
		fields = addFields(fields, emoji, duration, false)
	} else {
		duration = fmt.Sprintf(
			"**Ended**: %s ago\n**Duration**: %s",
			durafmt.Parse(time.Since(silence.EndsAt)),
			durafmt.Parse(silence.EndsAt.Sub(silence.StartsAt)),
		)
		fields = addFields(fields, "", duration, false)
	}
	fields = addFields(fields, "Comments", silence.Comment, false)
	fields = addFields(fields, "Created by", silence.CreatedBy, false)
	fields = addFields(fields, "AlertManagerPluginId", config.ID, false)

	color := colorResolved
	if string(silence.Status.State) == "active" {
		color = colorFiring
	}

	expireSilenceAction := &model.PostAction{
		Name: "Expire Silence",
		Type: model.PostActionTypeButton,
		Integration: &model.PostActionIntegration{
			Context: map[string]interface{}{
				"action":     "expire",
				"silence_id": silence.ID,
				"user_id":    userID,
			},
			URL: fmt.Sprintf("http://localhost%v/plugins/%v/api/expire?token=%s", siteURLPort, manifest.ID, config.Token),
		},
	}
	attachment := &model.SlackAttachment{
		Title:  silence.ID,
		Fields: fields,
		Color:  color,
		Actions: []*model.PostAction{
			expireSilenceAction,
		},
	}

	return attachment
}
