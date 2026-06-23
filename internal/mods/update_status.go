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
	case "update_ignored":
		return "update_ignored"
	default:
		return smapiStatus
	}
}

// PreserveUpdateStatus copies update status from a prior mod list when the installed
// manifest version is unchanged, so folder rescans do not wipe SMAPI check results.
func PreserveUpdateStatus(list []Mod, previous []Mod) {
	if len(previous) == 0 {
		return
	}
	type prior struct {
		version string
		status  UpdateStatus
	}
	byID := make(map[string]prior, len(previous))
	for _, m := range previous {
		byID[m.ID] = prior{version: m.Manifest.Version, status: m.UpdateStatus}
	}

	changedNexus := map[int]bool{}
	for _, m := range list {
		p, ok := byID[m.ID]
		if !ok || p.version == m.Manifest.Version {
			continue
		}
		if id := NexusModIDFromUpdateKeys(m.Manifest.UpdateKeys); id > 0 {
			changedNexus[id] = true
		}
	}

	for i := range list {
		p, ok := byID[list[i].ID]
		if !ok || p.version != list[i].Manifest.Version {
			continue
		}
		if id := NexusModIDFromUpdateKeys(list[i].Manifest.UpdateKeys); id > 0 && changedNexus[id] && p.status.State == "update_available" {
			continue
		}
		list[i].UpdateStatus = p.status
	}
}

// CachedUpdate is a persisted update check result for one mod.
type CachedUpdate struct {
	ManifestVersion string
	State           string
	LatestVersion   string
	ModPageURL      string
	Message         string
}

// ApplyCachedUpdateStatus restores persisted update results after a rescan when the
// installed manifest version is unchanged. Skips mods that already have a non-default status.
func ApplyCachedUpdateStatus(list []Mod, cached map[string]CachedUpdate) {
	if len(cached) == 0 {
		return
	}
	for i := range list {
		entry, ok := cached[list[i].ID]
		if !ok || entry.ManifestVersion != list[i].Manifest.Version {
			continue
		}
		st := list[i].UpdateStatus
		if st.State != "" && st.State != "current" {
			continue
		}
		list[i].UpdateStatus = UpdateStatus{
			State:         entry.State,
			LatestVersion: entry.LatestVersion,
			ModPageURL:    entry.ModPageURL,
			Message:       entry.Message,
		}
	}
	PropagateNexusUpdateStatus(list)
}
