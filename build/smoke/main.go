// Command smoke verifies Alertmanager -> webhook -> Mattermost delivery locally.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	client := model.NewAPIv4Client("http://localhost:8065")
	client.HTTPClient.Timeout = 15 * time.Second
	user, _, err := client.Login(ctx, "alertmanager-dev", "Local-alertmanager-123!")
	if err != nil {
		return err
	}
	channel, _, err := client.GetChannelByNameForTeamName(ctx, "alerts", "alerts-dev", "")
	if err != nil {
		return err
	}
	if _, _, err = client.AddChannelMember(ctx, channel.Id, user.Id); err != nil {
		return err
	}
	bot, _, err := client.GetUserByUsername(ctx, "alertmanagerbot", "")
	if err != nil {
		return err
	}
	if _, _, err = client.AddTeamMember(ctx, channel.TeamId, bot.Id); err != nil {
		return err
	}
	if _, _, err = client.AddChannelMember(ctx, channel.Id, bot.Id); err != nil {
		return err
	}
	name := fmt.Sprintf("SmokeTest%d", time.Now().UnixNano())
	startsAt := time.Now().UTC().Add(-10 * time.Second)
	payload := []map[string]any{{
		"labels":      map[string]string{"alertname": name, "service": "local-smoke", "severity": "warning"},
		"annotations": map[string]string{"summary": "Automated integration test"},
		"startsAt":    startsAt, "endsAt": time.Now().UTC().Add(5 * time.Minute),
	}}
	send := func() error {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost:9093/api/v2/alerts", bytes.NewReader(data))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("alertmanager HTTP %d: %s", resp.StatusCode, body)
		}
		return nil
	}
	waitForPost := func(status, color string) error {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			posts, _, err := client.GetPostsForChannel(ctx, channel.Id, 0, 100, "", false, false)
			if err != nil {
				return err
			}
			for _, post := range posts.Posts {
				data, err := json.Marshal(post.Attachments())
				if err != nil {
					return err
				}
				if post.UserId == bot.Id && strings.Contains(string(data), name) && strings.Contains(string(data), status) {
					for _, attachment := range post.Attachments() {
						if attachment.Color != color {
							return fmt.Errorf("%s notification color: got %s, want %s", status, attachment.Color, color)
						}
					}
					fmt.Printf("PASS: %s notification posted by bot\n", status)
					return nil
				}
			}
			select {
			case <-ctx.Done():
				return fmt.Errorf("waiting for %s: %w", status, ctx.Err())
			case <-ticker.C:
			}
		}
	}
	if err := send(); err != nil {
		return err
	}
	if err := waitForPost("FIRING", "#FFCC00"); err != nil {
		return err
	}
	for _, command := range []string{"status", "alerts", "silences"} {
		if _, _, err := client.ExecuteCommand(ctx, channel.Id, "/alertmanager "+command); err != nil {
			return err
		}
		fmt.Printf("PASS: /alertmanager %s accepted\n", command)
	}
	posts, _, err := client.GetPostsForChannel(ctx, channel.Id, 0, 100, "", false, false)
	if err != nil {
		return err
	}
	listed := false
	for _, post := range posts.Posts {
		for _, attachment := range post.Attachments() {
			if post.UserId == bot.Id && attachment.Title == "Alert Name: "+name {
				if attachment.Color != "#FFCC00" {
					return fmt.Errorf("listed warning color: got %s, want #FFCC00", attachment.Color)
				}
				listed = true
			}
		}
	}
	if !listed {
		return fmt.Errorf("/alertmanager alerts did not list the warning alert")
	}
	fmt.Println("PASS: /alertmanager alerts uses warning severity color")
	payload[0]["endsAt"] = time.Now().UTC().Add(-time.Second)
	if err := send(); err != nil {
		return err
	}
	return waitForPost("RESOLVED", "#008000")
}
