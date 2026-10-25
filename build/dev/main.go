// Command dev configures only the disposable localhost Docker environment.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

const siteURL = "http://localhost:8065"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: go run ./dev <plugin bundle>")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	client := model.NewAPIv4Client(siteURL)
	client.HTTPClient.Timeout = 30 * time.Second
	// On an empty server the first user becomes a system administrator. Login
	// below validates the credentials when this command is run again.
	_, _, _ = client.CreateUser(ctx, &model.User{
		Username: "alertmanager-dev", Email: "alertmanager-dev@example.com", Password: "Local-alertmanager-123!",
	})
	admin, _, err := client.Login(ctx, "alertmanager-dev", "Local-alertmanager-123!")
	if err != nil {
		return fmt.Errorf("login to disposable development server: %w", err)
	}
	team, resp, err := client.GetTeamByName(ctx, "alerts-dev", "")
	if err != nil {
		if resp == nil || resp.StatusCode != http.StatusNotFound {
			return err
		}
		team, _, err = client.CreateTeam(ctx, &model.Team{Name: "alerts-dev", DisplayName: "Plugin Development", Type: model.TeamOpen})
		if err != nil {
			return err
		}
	}
	config, _, err := client.GetConfig(ctx)
	if err != nil {
		return err
	}
	if config.PluginSettings.Plugins == nil {
		config.PluginSettings.Plugins = make(map[string]map[string]any)
	}
	config.PluginSettings.Plugins["alertmanager"] = map[string]any{
		"alertconfigs": map[string]any{"0": map[string]string{
			"team": team.Name, "channel": "alerts", "token": "local-development-token", "alertmanagerurl": "http://alertmanager:9093",
		}},
	}
	if _, _, err = client.UpdateConfig(ctx, config); err != nil {
		return fmt.Errorf("configure plugin: %w", err)
	}
	bundle, err := os.Open(os.Args[1])
	if err != nil {
		return err
	}
	defer func() { _ = bundle.Close() }()
	if _, _, err = client.UploadPluginForced(ctx, bundle); err != nil {
		return fmt.Errorf("upload plugin: %w", err)
	}
	if _, err = client.EnablePlugin(ctx, "alertmanager"); err != nil {
		return fmt.Errorf("enable plugin: %w", err)
	}
	channel, _, err := client.GetChannelByName(ctx, "alerts", team.Id, "")
	if err != nil {
		return fmt.Errorf("find configured channel: %w", err)
	}
	bot, _, err := client.GetUserByUsername(ctx, "alertmanagerbot", "")
	if err != nil {
		return err
	}
	if _, _, err := client.AddTeamMember(ctx, team.Id, bot.Id); err != nil {
		return err
	}
	for _, userID := range []string{admin.Id, bot.Id} {
		if _, _, err := client.AddChannelMember(ctx, channel.Id, userID); err != nil {
			return err
		}
	}
	fmt.Println("Ready: http://localhost:8065/alerts-dev/channels/alerts")
	fmt.Println("Login: alertmanager-dev / Local-alertmanager-123!")
	fmt.Println("Run make dev-alert to send and resolve a test alert.")
	return nil
}
