package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/prometheus/alertmanager/types"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"

	"github.com/cpanato/mattermost-plugin-alertmanager/server/alertmanager"
)

const (
	ActionHelp = "help"

	helpMsg = `run:
	/alertmanager alerts - to list the existing alerts
	/alertmanager silences - to list the existing silences
	/alertmanager expire_silence <ALERT_MANAGER_PLUGIN_ID> <SILENCE_ID> - to expire the specified silence
	/alertmanager status - to list the version and uptime of the Alertmanager instance
	/alertmanager help - to get this help
	`
)

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "alertmanager",
		DisplayName:      "Alert Manager",
		Description:      "Alert Manager Bot",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: status, alerts, silences, expire_silence, help",
		AutoCompleteHint: "[command]",
	}
}

func (p *Plugin) ExecuteCommand(_ *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	split := strings.Fields(args.Command)
	command := split[0]
	action := ""
	if len(split) > 1 {
		action = strings.TrimSpace(split[1])
	}

	if command != "/alertmanager" {
		return &model.CommandResponse{}, nil
	}

	if action == "" {
		return getCommandResponse(model.CommandResponseTypeEphemeral, "Missing command, please run `/alertmanager help` to check all commands available."), nil
	}

	if action == ActionHelp {
		return getCommandResponse(model.CommandResponseTypeEphemeral, helpMsg), nil
	}

	switch action {
	case "alerts":
		resp, err := p.handleAlert(args)
		return resp, err
	case "status":
		resp, err := p.handleStatus(args)
		return resp, err
	case "silences":
		resp, err := p.handleListSilences(args)
		return resp, err
	case "expire_silence":
		resp, err := p.handleExpireSilence(args)
		return resp, err
	case ActionHelp:
	default:
		return getCommandResponse(model.CommandResponseTypeEphemeral, helpMsg), nil
	}

	return &model.CommandResponse{}, nil
}

func getCommandResponse(responseType, text string) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: responseType,
		Text:         text,
		Type:         model.PostTypeDefault,
	}
}

func (p *Plugin) handleAlert(_ *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	var alertsCount = 0
	var errors []string

	for _, alertConfig := range p.configuration.AlertConfigs {
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
		}

		model.ParseSlackAttachment(post, attachments)
		if _, appErr := p.API.CreatePost(post); appErr != nil {
			errors = append(errors, fmt.Sprintf("Channel %q: Error creating the Alert post", alertConfig.Channel))
			continue
		}
	}

	if len(errors) > 0 {
		msg := strings.Join(errors, "\n")
		return getCommandResponse(model.CommandResponseTypeInChannel, msg), nil
	}

	if alertsCount == 0 {
		return getCommandResponse(model.CommandResponseTypeInChannel, "No alerts right now! :tada:"), nil
	}

	return &model.CommandResponse{}, nil
}

func (p *Plugin) handleStatus(_ *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	var errors []string
	for _, alertConfig := range p.configuration.AlertConfigs {
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
		}

		model.ParseSlackAttachment(post, []*model.SlackAttachment{attachment})
		if _, appErr := p.API.CreatePost(post); appErr != nil {
			errors = append(errors, fmt.Sprintf("Channel %q: Error creating the Status post", alertConfig.Channel))
			continue
		}
	}

	if len(errors) > 0 {
		msg := strings.Join(errors, "\n")
		return getCommandResponse(model.CommandResponseTypeInChannel, msg), nil
	}

	return &model.CommandResponse{}, nil
}

func (p *Plugin) handleListSilences(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	var errors []string
	var silencesCount = 0
	var pendingSilencesCount = 0

	config := p.API.GetConfig()
	siteURLPort := *config.ServiceSettings.ListenAddress

	for _, alertConfig := range p.configuration.AlertConfigs {
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
		}

		model.ParseSlackAttachment(post, attachments)
		if _, appErr := p.API.CreatePost(post); appErr != nil {
			errors = append(errors, fmt.Sprintf("Channel %q: Error creating the Alert post", alertConfig.Channel))
			continue
		}
	}

	if silencesCount == 0 {
		return getCommandResponse(model.CommandResponseTypeInChannel, "No silences right now."), nil
	}

	if pendingSilencesCount == 0 {
		return getCommandResponse(model.CommandResponseTypeInChannel, "No active or pending silences right now."), nil
	}

	if len(errors) > 0 {
		msg := strings.Join(errors, "\n")
		return getCommandResponse(model.CommandResponseTypeInChannel, msg), nil
	}

	return &model.CommandResponse{}, nil
}

func (p *Plugin) handleExpireSilence(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	split := strings.Fields(args.Command)
	var parameters []string
	if len(split) > 2 {
		parameters = split[2:]
	}

	if len(parameters) != 2 {
		return getCommandResponse(model.CommandResponseTypeEphemeral, "Command requires 2 parameters: ALERT_MANAGER_PLUGIN_ID and SILENCE_ID"), nil
	}

	if config, ok := p.configuration.AlertConfigs[parameters[0]]; ok {
		err := alertmanager.ExpireSilence(parameters[1], config.AlertManagerURL)
		if err != nil {
			msg := fmt.Sprintf("failed to expire the silence: %v", err)
			return getCommandResponse(model.CommandResponseTypeInChannel, msg), nil
		}
	} else {
		msg := fmt.Sprintf("Missing such ALERT_MANAGER_PLUGIN_ID (%s)", parameters[0])
		return getCommandResponse(model.CommandResponseTypeInChannel, msg), nil
	}

	silenceDeleted := fmt.Sprintf("Silence %s expired.", parameters[1])
	return getCommandResponse(model.CommandResponseTypeInChannel, silenceDeleted), nil
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
