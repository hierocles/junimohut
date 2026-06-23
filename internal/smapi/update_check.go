package smapi

import (
	"errors"
	"fmt"
	"time"
)

// UpdateCheckCooldown limits how often we call SMAPI's update API.
// SMAPI refreshes Nexus version data about once per day, so more frequent
// checks would repeat the same results while adding unnecessary load.
const UpdateCheckCooldown = 24 * time.Hour

var ErrUpdateCheckRateLimited = errors.New("mod update check was run recently")

// UpdateCheckRateLimited reports whether a mod update check ran within the cooldown window.
func UpdateCheckRateLimited(lastCheck int64, now time.Time) bool {
	if lastCheck == 0 {
		return false
	}
	return now.Sub(time.Unix(lastCheck, 0)) < UpdateCheckCooldown
}

// UpdateCheckRetryAfter returns how long to wait before another check is allowed.
func UpdateCheckRetryAfter(lastCheck int64, now time.Time) time.Duration {
	if lastCheck == 0 {
		return 0
	}
	elapsed := now.Sub(time.Unix(lastCheck, 0))
	if elapsed >= UpdateCheckCooldown {
		return 0
	}
	return UpdateCheckCooldown - elapsed
}

// FormatUpdateCheckRetryAfter returns a short human-readable wait time.
func FormatUpdateCheckRetryAfter(d time.Duration) string {
	if d <= 0 {
		return "soon"
	}
	if d >= time.Hour {
		hours := int(d.Hours()) + 1
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	minutes := int(d.Minutes()) + 1
	if minutes == 1 {
		return "1 minute"
	}
	return fmt.Sprintf("%d minutes", minutes)
}
