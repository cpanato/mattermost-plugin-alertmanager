package main

import "strings"

const (
	colorFiring   = "#FF0000" // red
	colorWarning  = "#FFCC00" // yellow
	colorInfo     = "#3498DB" // blue
	colorResolved = "#008000" // green
	colorExpired  = "#F0F8FF" // aliceBlue
)

func setColor(status, severity string) string {
	switch status {
	case "resolved":
		return colorResolved
	case "firing":
		switch strings.ToLower(strings.TrimSpace(severity)) {
		case "info":
			return colorInfo
		case "warning":
			return colorWarning
		default: // critical, high, unknown or absent
			return colorFiring
		}
	default:
		return colorExpired
	}
}
