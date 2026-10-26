// Command smoke verifies Alertmanager -> webhook -> Mattermost delivery locally.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

func main() {
	ha := flag.Bool("ha", false, "send to all three local Alertmanagers and check cluster deduplication")
	flag.Parse()
	if err := run(*ha); err != nil {
		log.Fatal(err)
	}
}

func run(ha bool) error {
	budget := 90 * time.Second
	urls := []string{"http://localhost:9093"}
	if ha {
		budget = 3 * time.Minute
		urls = append(urls, "http://localhost:9095", "http://localhost:9096")
	}
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	if ha {
		if err := waitForCluster(ctx, urls); err != nil {
			return err
		}
	}
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
		for _, url := range urls {
			if err := sendAlert(ctx, url, data); err != nil {
				return err
			}
		}
		return nil
	}
	waitForPost := func(status, color string) error {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		var firstSeen time.Time
		for {
			posts, _, err := client.GetPostsForChannel(ctx, channel.Id, 0, 100, "", false, false)
			if err != nil {
				return err
			}
			count := 0
			for _, post := range posts.Posts {
				if post.UserId != bot.Id {
					continue
				}
				for _, attachment := range post.Attachments() {
					if attachment.Title != status+": "+name {
						continue
					}
					count++
					if attachment.Color != color {
						return fmt.Errorf("%s notification color: got %s, want %s", status, attachment.Color, color)
					}
				}
			}
			if count > 1 {
				return fmt.Errorf("%s: got %d notifications for one alert", status, count)
			}
			if count == 1 {
				if firstSeen.IsZero() {
					firstSeen = time.Now()
				}
				// Default HA peer waits are 0/15/30s. Observe beyond the last
				// peer plus the local 5s group interval before declaring success.
				if !ha || time.Since(firstSeen) >= 40*time.Second {
					fmt.Printf("PASS: exactly one %s notification posted by bot\n", status)
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

func sendAlert(ctx context.Context, url string, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url+"/api/v2/alerts", bytes.NewReader(data))
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
		return fmt.Errorf("%s HTTP %d: %s", url, resp.StatusCode, body)
	}
	return nil
}

func waitForCluster(ctx context.Context, urls []string) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	client := &http.Client{Timeout: 5 * time.Second}
	for {
		var lastErr error
		for _, url := range urls {
			if err := checkCluster(ctx, client, url, len(urls)); err != nil {
				lastErr = err
				break
			}
		}
		if lastErr == nil {
			fmt.Printf("PASS: all %d Alertmanagers see a ready cluster with %d peers\n", len(urls), len(urls))
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for HA cluster: %w (last check: %v)", ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func checkCluster(ctx context.Context, client *http.Client, url string, peers int) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"/api/v2/status", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s status HTTP %d", url, resp.StatusCode)
	}
	var status struct {
		Cluster struct {
			Status string            `json:"status"`
			Peers  []json.RawMessage `json:"peers"`
		} `json:"cluster"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return err
	}
	if status.Cluster.Status != "ready" || len(status.Cluster.Peers) != peers {
		return fmt.Errorf("%s cluster is %q with %d peers, want ready with %d", url, status.Cluster.Status, len(status.Cluster.Peers), peers)
	}
	return nil
}
