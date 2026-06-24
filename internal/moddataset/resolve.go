package moddataset

import (
	"fmt"

	"junimohut/internal/nexus"
)

// ResolveNexusModID picks the best Nexus mod ID for a mod.
// Priority: manifest UpdateKeys → download index ID → dataset index → bundle Nexus ID.
func ResolveNexusModID(uniqueID string, updateKeys []string, idx *Index, downloadNexusID, bundleNexusID int) int {
	if id := nexus.ModIDFromUpdateKeys(updateKeys); id > 0 {
		return id
	}
	if downloadNexusID > 0 {
		return downloadNexusID
	}
	if idx != nil {
		if id := idx.FirstNexusID(uniqueID); id > 0 {
			return id
		}
	}
	return bundleNexusID
}

// NexusUpdateKeysFromIndex returns synthetic Nexus update keys from the dataset index.
func NexusUpdateKeysFromIndex(idx *Index, uniqueID string) []string {
	if idx == nil {
		return nil
	}
	id := idx.FirstNexusID(uniqueID)
	if id <= 0 {
		return nil
	}
	return []string{fmt.Sprintf("Nexus:%d", id)}
}
