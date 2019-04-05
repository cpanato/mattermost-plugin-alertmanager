package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
	BotUserID     string
	ChannelID     string
}

func (p *Plugin) OnDeactivate() error {
	command := getCommand()
	return p.API.UnregisterCommand("", command.Trigger)
}

func (p *Plugin) OnActivate() error {
	configuration := p.getConfiguration()

	if err := p.IsValid(configuration); err != nil {
		return err
	}

	team, err := p.API.GetTeamByName(p.configuration.Team)
	if err != nil {
		return err
	}

	user, err := p.API.GetUserByUsername(p.configuration.Username)
	if err != nil {
		p.API.LogError(err.Error())
		return fmt.Errorf("Unable to find user with configured username: %v", p.configuration.Username)
	}
	p.BotUserID = user.Id

	channel, err := p.API.GetChannelByName(team.Id, p.configuration.Channel, false)
	if err != nil && err.StatusCode == http.StatusNotFound {
		channelToCreate := &model.Channel{
			Name:        p.configuration.Channel,
			DisplayName: p.configuration.Channel,
			Type:        model.CHANNEL_OPEN,
			TeamId:      team.Id,
			CreatorId:   user.Id,
		}

		newChannel, errChannel := p.API.CreateChannel(channelToCreate)
		if err != nil {
			return errChannel
		}
		p.ChannelID = newChannel.Id
	} else if err != nil {
		return err
	} else {
		p.ChannelID = channel.Id
	}

	p.API.RegisterCommand(getCommand())

	return nil
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" || strings.Compare(token, p.configuration.Token) != 0 {
		errorMessage := "Invalid or missing token"
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}
	switch r.URL.Path {
	case "/api/webhook":
		p.handleWebhook(w, r)
	case "/api/expire":
		p.handleExpireAction(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (p *Plugin) IsValid(configuration *configuration) error {
	if configuration.Team == "" {
		return fmt.Errorf("Must set a Team")
	}

	if configuration.Channel == "" {
		return fmt.Errorf("Must set a Channel")
	}

	if configuration.Username == "" {
		return fmt.Errorf("Must set a User")
	}

	if configuration.Token == "" {
		return fmt.Errorf("Must set a Token")
	}

	if configuration.AlertManagerURL == "" {
		return fmt.Errorf("Must set the AlertManager URL")
	}

	return nil
}
