package mods

// NormalizeUpdateState maps SMAPI update API statuses to canonical mod update states.
func NormalizeUpdateState(smapiStatus string) string {
	switch smapiStatus {
	case "update":
		return "update_available"
	case "ok":
		return "current"
	case "incompatible":
		return "incompatible"
	default:
		return smapiStatus
	}
}
