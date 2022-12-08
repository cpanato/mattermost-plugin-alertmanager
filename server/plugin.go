package main

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

type Plugin struct {
	plugin.MattermostPlugin
	client *pluginapi.Client

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	// key - channel name from config, value - existing or created channel id received from api
	ChannelIds map[string]string
	BotUserID  string

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex
}

func (p *Plugin) OnDeactivate() error {
	return nil
}

func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)
	botID, err := p.client.Bot.EnsureBot(&model.Bot{
		Username:    "alertmanagerbot",
		DisplayName: "AlertManager Bot",
		Description: "Created by the AlertManager plugin.",
	}, pluginapi.ProfileImagePath(filepath.Join("assets", "alertmanager-logo.png")))
	if err != nil {
		return errors.Wrap(err, "failed to ensure bot account")
	}
	p.BotUserID = botID

	configuration := p.getConfiguration()
	p.ChannelIds = make(map[string]string)
	for k, alertConfig := range configuration.AlertConfigs {
		var channelID string
		channelID, err = p.ensureAlertChannelExists(alertConfig)
		if err != nil {
			p.API.LogWarn(fmt.Sprintf("Failed to ensure alert channel %v", k), "error", err.Error())
		} else {
			p.ChannelIds[alertConfig.Channel] = channelID
		}
	}

	_ = p.API.RegisterCommand(getCommand())

	return nil
}

func (p *Plugin) ensureAlertChannelExists(alertConfig alertConfig) (string, error) {
	if err := alertConfig.IsValid(); err != nil {
		return "", errors.Wrap(err, "Alert Configuration is invalid")
	}

	team, appErr := p.API.GetTeamByName(alertConfig.Team)
	if appErr != nil {
		return "", errors.Wrap(appErr, "Failed to get team")
	}

	channel, appErr := p.API.GetChannelByName(team.Id, alertConfig.Channel, false)
	if appErr != nil {
		if appErr.StatusCode == http.StatusNotFound {
			channelToCreate := &model.Channel{
				Name:        alertConfig.Channel,
				DisplayName: alertConfig.Channel,
				Type:        model.ChannelTypeOpen,
				TeamId:      team.Id,
				CreatorId:   p.BotUserID,
			}

			newChannel, errChannel := p.API.CreateChannel(channelToCreate)
			if errChannel != nil {
				return "", errors.Wrap(appErr, "Failed to create alert channel")
			}

			return newChannel.Id, nil
		}
		return "", errors.Wrap(appErr, "Failed to get existing alert channel")
	}

	return channel.Id, nil
}

func (p *Plugin) ServeHTTP(_ *plugin.Context, w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Mattermost AlertManager Plugin"))
		return
	}

	invalidOrMissingTokenErr := "Invalid or missing token"
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, invalidOrMissingTokenErr, http.StatusBadRequest)
		return
	}
	for _, alertConfig := range p.configuration.AlertConfigs {
		if subtle.ConstantTimeCompare([]byte(token), []byte(alertConfig.Token)) == 0 {
			switch r.URL.Path {
			case "/api/webhook":
				p.handleWebhook(w, r, alertConfig)
			case "/api/expire":
				p.handleExpireAction(w, r, alertConfig)
			default:
				http.NotFound(w, r)
			}
			return
		}
	}

	http.Error(w, invalidOrMissingTokenErr, http.StatusBadRequest)
}
