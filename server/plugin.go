package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

const (
	BotUserKey = "AlertManagerBot"
)

type Plugin struct {
	plugin.MattermostPlugin

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
	BotUserID     string
	ChannelID     string

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex
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

	if err := p.ensureBotExists(); err != nil {
		return errors.Wrap(err, "failed to ensure bot user exists")
	}

	team, err := p.API.GetTeamByName(p.configuration.Team)
	if err != nil {
		return err
	}

	_ = p.API.RegisterCommand(getCommand())

	channel, err := p.API.GetChannelByName(team.Id, p.configuration.Channel, false)
	if err != nil && err.StatusCode == http.StatusNotFound {
		channelToCreate := &model.Channel{
			Name:        p.configuration.Channel,
			DisplayName: p.configuration.Channel,
			Type:        model.ChannelTypeOpen,
			TeamId:      team.Id,
			CreatorId:   p.BotUserID,
		}

		newChannel, errChannel := p.API.CreateChannel(channelToCreate)
		if errChannel != nil {
			return errChannel
		}

		p.ChannelID = newChannel.Id
		return nil
	} else if err != nil {
		return err
	}

	p.ChannelID = channel.Id
	return nil
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Mattermost AlertManager Plugin"))
		return
	}

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
		return fmt.Errorf("must set a Team")
	}

	if configuration.Channel == "" {
		return fmt.Errorf("must set a Channel")
	}

	if configuration.Token == "" {
		return fmt.Errorf("must set a Token")
	}

	if configuration.AlertManagerURL == "" {
		return fmt.Errorf("must set the AlertManager URL")
	}

	return nil
}

func (p *Plugin) ensureBotExists() error {
	// Attempt to find an existing bot
	botUserIDBytes, err := p.API.KVGet(BotUserKey)
	if err != nil {
		return err
	}

	if botUserIDBytes == nil {
		// Create a bot since one doesn't exist
		p.API.LogDebug("Creating bot for AlertManager plugin")

		bot, err := p.API.CreateBot(&model.Bot{
			Username:    "alertmanagerbot",
			DisplayName: "AlertManager Bot",
			Description: "Created by the AlertManager plugin.",
		})
		if err != nil {
			return err
		}

		// Give it a profile picture
		err = p.API.SetProfileImage(bot.UserId, profileImage)
		if err != nil {
			p.API.LogError("Failed to set profile image for bot", "err", err)
		}

		p.API.LogDebug("Bot created for AlertManager plugin")

		// Save the bot ID
		err = p.API.KVSet(BotUserKey, []byte(bot.UserId))
		if err != nil {
			return err
		}
		p.BotUserID = bot.UserId
	} else {
		p.BotUserID = string(botUserIDBytes)
	}

	return nil
}
