package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cpanato/mattermost-plugin-alertmanager/server/alertmanager"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"

	"github.com/hako/durafmt"
)

const (
	helpMsg = `run:
	/alertmanager alerts - to list the existing alerts
	/alertmanager silences - to list the existing silences
	/alertmanager expire_silence <SILENCE_ID> - to expire the specified silence
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

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
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
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Missing command, please run `/alertmanager help` to check all commands available."), nil
	}

	if action == "help" {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, helpMsg), nil
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
	case "help":
	default:
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, helpMsg), nil
	}

	return &model.CommandResponse{}, nil
}

func getCommandResponse(responseType, text string) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: responseType,
		Text:         text,
		Username:     alertManagerUsername,
		IconURL:      alertManagerIconURL,
		Type:         model.POST_DEFAULT,
	}
}

func (p *Plugin) sendEphemeralMessage(msg, channelId, userId string) {
	ephemeralPost := &model.Post{
		Message:   msg,
		ChannelId: channelId,
		UserId:    userId,
		Props: model.StringInterface{
			"override_username": alertManagerUsername,
			"override_icon_url": alertManagerIconURL,
			"from_webhook":      "true",
		},
	}

	p.API.LogDebug("Will send an ephemeralPost", "msg", msg)

	p.API.SendEphemeralPost(userId, ephemeralPost)
}

func (p *Plugin) handleAlert(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	alerts, err := alertmanager.ListAlerts(p.configuration.AlertManagerURL)
	if err != nil {
		msg := fmt.Sprintf("failed to list alerts... %v", err)
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, msg), nil
	}

	if len(alerts) == 0 {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, "No alerts right now! :tada:"), nil
	}

	for _, alert := range alerts {
		fmt.Println(alert.Alert)
	}

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
		attachment := &model.SlackAttachment{
			Title:  alert.Name(),
			Fields: fields,
			Color:  setColor(string(alert.Status())),
		}
		attachments = append(attachments, attachment)
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

	model.ParseSlackAttachment(post, attachments)
	if _, appErr := p.API.CreatePost(post); appErr != nil {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Error creating the Alert post"), nil
	}
	return &model.CommandResponse{}, nil
}

func (p *Plugin) handleStatus(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	status, err := alertmanager.Status(p.configuration.AlertManagerURL)
	if err != nil {
		msg := fmt.Sprintf("failed to get status... %v", err)
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, msg), nil
	}

	uptime := durafmt.Parse(time.Since(status.Uptime)).String()
	var fields []*model.SlackAttachmentField
	fields = addFields(fields, "AlertManager Version ", status.VersionInfo.Version, false)
	fields = addFields(fields, "AlertManager Uptime", uptime, false)

	attachment := &model.SlackAttachment{
		Fields: fields,
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
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Error creating the Status post"), nil
	}
	return &model.CommandResponse{}, nil
}

func (p *Plugin) handleListSilences(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	silences, err := alertmanager.ListSilences(p.configuration.AlertManagerURL)
	if err != nil {
		msg := fmt.Sprintf("failed to get silences... %v", err)
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, msg), nil
	}

	if len(silences) == 0 {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, "No silences right now."), nil
	}

	attachments := make([]*model.SlackAttachment, 0)
	for _, silence := range silences {
		if string(silence.Status.State) == "expired" {
			continue
		}
		var fields []*model.SlackAttachmentField
		var emoji, matchers, duration string
		for _, m := range silence.Matchers {
			if m.Name == "alertname" {
				fields = addFields(fields, "Alert Name", m.Value, false)
			} else {
				matchers = matchers + fmt.Sprintf(`%s="%s"`, m.Name, m.Value)
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

		color := "#808080" //gray
		if string(silence.Status.State) == "active" {
			color = "#008000" //green
		}

		config := p.API.GetConfig()
		siteURLPort := *config.ServiceSettings.ListenAddress
		expireSilenceAction := &model.PostAction{
			Name: "Expire Silence",
			Type: model.POST_ACTION_TYPE_BUTTON,
			Integration: &model.PostActionIntegration{
				Context: map[string]interface{}{
					"action":     "expire",
					"silence_id": silence.ID,
					"user_id":    args.UserId,
				},
				URL: fmt.Sprintf("http://localhost%v/plugins/%v/api/expire?token=%s", siteURLPort, manifest.Id, p.configuration.Token),
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
		attachments = append(attachments, attachment)
	}

	if len(attachments) == 0 {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, "No active or pending silences right now."), nil
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

	model.ParseSlackAttachment(post, attachments)
	if _, appErr := p.API.CreatePost(post); appErr != nil {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Error creating the Alert post"), nil
	}
	return &model.CommandResponse{}, nil
}

func (p *Plugin) handleExpireSilence(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	split := strings.Fields(args.Command)
	parameters := []string{}
	if len(split) > 2 {
		parameters = split[2:]
	}

	if len(parameters) != 1 {
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Missing silence ID"), nil
	}

	err := alertmanager.ExpireSilence(parameters[0], p.configuration.AlertManagerURL)
	if err != nil {
		msg := fmt.Sprintf("failed to expire the silence: %v", err)
		return getCommandResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, msg), nil
	}

	silenceDeleted := fmt.Sprintf("Silence %s expired.", parameters[0])
	return getCommandResponse(model.COMMAND_RESPONSE_TYPE_IN_CHANNEL, silenceDeleted), nil
}
