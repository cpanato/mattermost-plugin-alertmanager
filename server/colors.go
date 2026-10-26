package main

import "github.com/cpanato/mattermost-plugin-alertmanager/internal/notification"

const (
	colorFiring   = notification.ColorFiring
	colorResolved = notification.ColorResolved
	colorExpired  = notification.ColorExpired
)

func setColor(status, severity string) string {
	return notification.Color(status, severity)
}
