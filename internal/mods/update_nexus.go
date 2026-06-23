package mods

import (
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// PropagateNexusUpdateStatus unifies update status across mods that share a Nexus mod ID.
// If any sibling reports an update, the whole group is marked with the newest latest version.
func PropagateNexusUpdateStatus(list []Mod) {
	groups := nexusModGroups(list)
	for _, indices := range groups {
		if len(indices) < 2 {
			continue
		}
		best := UpdateStatus{}
		hasUpdate := false
		allCurrent := true
		for _, i := range indices {
			st := list[i].UpdateStatus
			switch st.State {
			case "update", "update_available":
				hasUpdate = true
				allCurrent = false
				if best.LatestVersion == "" || versionGreater(st.LatestVersion, best.LatestVersion) {
					best = st
				}
			case "current":
				// keep scanning
			default:
				allCurrent = false
			}
		}
		if hasUpdate {
			best.State = "update_available"
			for _, i := range indices {
				list[i].UpdateStatus = best
			}
			continue
		}
		if allCurrent {
			for _, i := range indices {
				list[i].UpdateStatus.State = "current"
			}
		}
	}
}

// ApplyIgnoredUpdates marks matching Nexus groups as update_ignored.
func ApplyIgnoredUpdates(list []Mod, ignored map[string]string) {
	if len(ignored) == 0 {
		return
	}
	for i := range list {
		nexusID := NexusModIDFromUpdateKeys(list[i].Manifest.UpdateKeys)
		if nexusID == 0 {
			continue
		}
		ignoredVer, ok := ignored[strconv.Itoa(nexusID)]
		if !ok || ignoredVer == "" {
			continue
		}
		st := list[i].UpdateStatus
		if st.State == "update_available" && st.LatestVersion == ignoredVer {
			st.State = "update_ignored"
			list[i].UpdateStatus = st
		}
	}
	PropagateNexusIgnoredStatus(list)
}

func PropagateNexusIgnoredStatus(list []Mod) {
	groups := nexusModGroups(list)
	for _, indices := range groups {
		if len(indices) < 2 {
			continue
		}
		var ignored UpdateStatus
		found := false
		for _, i := range indices {
			if list[i].UpdateStatus.State == "update_ignored" {
				ignored = list[i].UpdateStatus
				found = true
				break
			}
		}
		if !found {
			continue
		}
		for _, i := range indices {
			st := list[i].UpdateStatus
			if st.State == "update_available" && st.LatestVersion == ignored.LatestVersion {
				st.State = "update_ignored"
				list[i].UpdateStatus = st
			}
		}
	}
}

// NexusSiblingFolderPaths returns every installed folder path sharing the same Nexus mod ID.
func NexusSiblingFolderPaths(list []Mod, folderPath string) []string {
	targetID := 0
	for _, m := range list {
		if m.FolderPath == folderPath {
			targetID = NexusModIDFromUpdateKeys(m.Manifest.UpdateKeys)
			break
		}
	}
	if targetID == 0 {
		return []string{folderPath}
	}
	var paths []string
	for _, m := range list {
		if NexusModIDFromUpdateKeys(m.Manifest.UpdateKeys) == targetID {
			paths = append(paths, m.FolderPath)
		}
	}
	if len(paths) == 0 {
		return []string{folderPath}
	}
	return paths
}

// PickNexusGroupRepresentative returns the mod with the lowest version in a Nexus group.
func PickNexusGroupRepresentative(list []Mod, nexusID int) (Mod, bool) {
	var rep Mod
	found := false
	for _, m := range list {
		if NexusModIDFromUpdateKeys(m.Manifest.UpdateKeys) != nexusID {
			continue
		}
		if !found || versionLess(m.Manifest.Version, rep.Manifest.Version) {
			rep = m
			found = true
		}
	}
	return rep, found
}

func nexusModGroups(list []Mod) map[int][]int {
	groups := map[int][]int{}
	for i, m := range list {
		if id := NexusModIDFromUpdateKeys(m.Manifest.UpdateKeys); id > 0 {
			groups[id] = append(groups[id], i)
		}
	}
	return groups
}

func versionGreater(a, b string) bool {
	if b == "" {
		return a != ""
	}
	if a == "" {
		return false
	}
	av, err1 := semver.NewVersion(strings.TrimPrefix(strings.TrimSpace(a), "v"))
	bv, err2 := semver.NewVersion(strings.TrimPrefix(strings.TrimSpace(b), "v"))
	if err1 != nil || err2 != nil {
		return a > b
	}
	return av.GreaterThan(bv)
}

func versionLess(a, b string) bool {
	if strings.TrimSpace(b) == "" {
		return false
	}
	if strings.TrimSpace(a) == "" {
		return true
	}
	av, err1 := semver.NewVersion(strings.TrimPrefix(strings.TrimSpace(a), "v"))
	bv, err2 := semver.NewVersion(strings.TrimPrefix(strings.TrimSpace(b), "v"))
	if err1 != nil || err2 != nil {
		return a < b
	}
	return av.LessThan(bv)
}
