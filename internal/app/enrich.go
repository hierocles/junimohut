package app

import (
	"junimohut/internal/mods"
	"junimohut/internal/modupdates"
	"junimohut/internal/nexus"
)

func enrichDownloadRecordFromArchive(rec *nexus.DownloadRecord) {
	if rec == nil || rec.ArchivePath == "" {
		return
	}
	manifests, err := mods.ManifestsFromArchive(rec.ArchivePath)
	if err != nil || len(manifests) == 0 {
		return
	}
	manifest := manifests[0]
	if rec.UniqueID == "" && manifest.UniqueID != "" {
		rec.UniqueID = manifest.UniqueID
	}
	if rec.ModName == "" && manifest.Name != "" {
		rec.ModName = manifest.Name
	}
	if rec.NexusModID == 0 {
		if id := nexus.ModIDFromUpdateKeys(manifest.UpdateKeys); id > 0 {
			rec.NexusModID = id
		}
	}
}

func cachedUpdatesFromService(svc *modupdates.Service) map[string]mods.CachedUpdate {
	if svc == nil {
		return nil
	}
	raw := svc.All()
	out := make(map[string]mods.CachedUpdate, len(raw))
	for id, e := range raw {
		out[id] = mods.CachedUpdate{
			ManifestVersion:      e.ManifestVersion,
			State:                e.State,
			LatestVersion:        e.LatestVersion,
			ModPageURL:           e.ModPageURL,
			Message:              e.Message,
			CompatibilityStatus:  e.CompatibilityStatus,
			CompatibilitySummary: e.CompatibilitySummary,
		}
	}
	return out
}
