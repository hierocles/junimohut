package smapi

import (
	"html"
	"regexp"
	"strings"
)

var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

// StripCompatibilityHTML converts SMAPI compatibility summary HTML to plain text.
func StripCompatibilityHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = htmlTagPattern.ReplaceAllString(raw, "")
	raw = html.UnescapeString(raw)
	return strings.TrimSpace(raw)
}

// MapCompatibilityStatus maps wiki compatibility status to mod update state.
// Returns empty state when no override is needed.
func MapCompatibilityStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "Broken", "Obsolete", "Abandoned":
		return "incompatible"
	default:
		return ""
	}
}
