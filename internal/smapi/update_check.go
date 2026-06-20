package smapi

import "time"

const UpdateCheckCooldown = time.Hour

// UpdateCheckRateLimited reports whether a mod update check ran within the cooldown window.
func UpdateCheckRateLimited(lastCheck int64, now time.Time) bool {
	if lastCheck == 0 {
		return false
	}
	return now.Sub(time.Unix(lastCheck, 0)) < UpdateCheckCooldown
}
