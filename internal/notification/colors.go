package notification

import "strings"

const (
	ColorFiring   = "#FF0000" // red
	ColorWarning  = "#FFCC00" // yellow
	ColorInfo     = "#3498DB" // blue
	ColorResolved = "#008000" // green
	ColorExpired  = "#F0F8FF" // aliceBlue
)

func Color(status, severity string) string {
	switch status {
	case "resolved":
		return ColorResolved
	case "firing":
		switch strings.ToLower(strings.TrimSpace(severity)) {
		case "info":
			return ColorInfo
		case "warning":
			return ColorWarning
		default: // critical, high, unknown or absent
			return ColorFiring
		}
	default:
		return ColorExpired
	}
}
