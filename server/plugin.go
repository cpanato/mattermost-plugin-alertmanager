package main

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	pluginapi "github.com/mattermost/mattermost/server/public/pluginapi"

	root "github.com/cpanato/mattermost-plugin-alertmanager"
)

var (
	Manifest model.Manifest = root.Manifest
)

type Plugin struct {
	plugin.MattermostPlugin
	client *pluginapi.Client

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	// key - alert config id, value - existing or created channel id received from api
	AlertConfigIDChannelID map[string]string
	channelLock            sync.RWMutex
	BotUserID              string

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
		return fmt.Errorf("failed to ensure bot account: %w", err)
	}
	p.BotUserID = botID

	if err := p.refreshAlertChannels(); err != nil {
		return err
	}

	command, err := p.getCommand()
	if err != nil {
		return fmt.Errorf("failed to get command: %w", err)
	}

	err = p.API.RegisterCommand(command)
	if err != nil {
		return fmt.Errorf("failed to register command: %w", err)
	}

	return nil
}

func (p *Plugin) refreshAlertChannels() error {
	channels := make(map[string]string)
	for id, config := range p.getConfiguration().AlertConfigs {
		channelID, err := p.ensureAlertChannelExists(config)
		if err != nil {
			p.API.LogWarn("Failed to ensure alert channel", "id", id, "error", err.Error())
			continue
		}
		channels[id] = channelID
	}
	p.channelLock.Lock()
	p.AlertConfigIDChannelID = channels
	p.channelLock.Unlock()
	return nil
}

func (p *Plugin) ensureAlertChannelExists(alertConfig alertConfig) (string, error) {
	if err := alertConfig.IsValid(); err != nil {
		return "", fmt.Errorf("alert Configuration is invalid: %w", err)
	}

	team, appErr := p.API.GetTeamByName(alertConfig.Team)
	if appErr != nil {
		return "", fmt.Errorf("failed to get team: %w", appErr)
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
				return "", fmt.Errorf("failed to create alert channel: %w", errChannel)
			}

			return newChannel.Id, nil
		}
		return "", fmt.Errorf("failed to get existing alert channel: %w", appErr)
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
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, invalidOrMissingTokenErr, http.StatusBadRequest)
		return
	}

	configuration := p.getConfiguration()
	for _, alertConfig := range configuration.AlertConfigs {
		if subtle.ConstantTimeCompare([]byte(token), []byte(alertConfig.Token)) == 1 {
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

func (p *Plugin) alertChannelID(id string) string {
	p.channelLock.RLock()
	defer p.channelLock.RUnlock()
	return p.AlertConfigIDChannelID[id]
}
