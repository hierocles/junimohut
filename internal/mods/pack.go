package mods

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PackUniqueIDPrefix marks synthetic pack mod IDs derived from Nexus update keys.
const PackUniqueIDPrefix = "pack:nexus:"

// CollapseSiblingPacks merges sibling content-pack mods under a container folder into one pack mod.
func CollapseSiblingPacks(mods []Mod, modsRoot string, enabled map[string]bool) []Mod {
	if len(mods) == 0 {
		return mods
	}

	byContainer := map[string][]Mod{}
	var order []string
	for _, m := range mods {
		container := containerPath(m.FolderPath)
		if container == "" {
			continue
		}
		if _, ok := byContainer[container]; !ok {
			order = append(order, container)
		}
		byContainer[container] = append(byContainer[container], m)
	}

	collapsedContainers := map[string]bool{}
	for _, container := range order {
		group := byContainer[container]
		if len(group) < 2 {
			continue
		}
		if !isSiblingContentPackGroup(modsRoot, container, group) {
			continue
		}
		collapsedContainers[container] = true
	}

	if len(collapsedContainers) == 0 {
		return mods
	}

	out := make([]Mod, 0, len(mods))
	seen := map[string]bool{}
	for _, m := range mods {
		container := containerPath(m.FolderPath)
		if collapsedContainers[container] {
			if seen[container] {
				continue
			}
			seen[container] = true
			out = append(out, buildPackMod(container, byContainer[container], modsRoot, enabled))
			continue
		}
		out = append(out, m)
	}
	return out
}

func containerPath(folderPath string) string {
	parts := strings.Split(filepath.ToSlash(folderPath), "/")
	if len(parts) < 2 {
		return ""
	}
	return strings.Join(parts[:len(parts)-1], "/")
}

func isSiblingContentPackGroup(modsRoot, container string, group []Mod) bool {
	containerAbs := filepath.Join(modsRoot, filepath.FromSlash(container))
	if HasManifestInDir(containerAbs) {
		return false
	}

	nexusID := 0
	for _, m := range group {
		if m.Manifest.ContentPackFor == nil {
			return false
		}
		id := nexusIDFromUpdateKeys(m.Manifest.UpdateKeys)
		if id == 0 {
			return false
		}
		if nexusID == 0 {
			nexusID = id
		} else if nexusID != id {
			return false
		}
	}
	return true
}

func buildPackMod(container string, children []Mod, modsRoot string, enabled map[string]bool) Mod {
	first := children[0]
	childIDs := make([]string, len(children))
	var maxInstall, maxUpdated int64
	hasConfig := false
	hasJsonFiles := false
	jsonFileCount := 0
	for i, c := range children {
		childIDs[i] = c.ID
		if c.InstallTime > maxInstall {
			maxInstall = c.InstallTime
		}
		if c.LastUpdated > maxUpdated {
			maxUpdated = c.LastUpdated
		}
		if c.HasConfig {
			hasConfig = true
		}
		if c.HasJsonFiles {
			hasJsonFiles = true
		}
		jsonFileCount += c.JsonFileCount
	}

	packUID := PackUniqueIDFromManifest(first.Manifest)
	name := PackDisplayName(filepath.Base(container))
	abs := filepath.Join(modsRoot, filepath.FromSlash(container))
	packID := ModID(container, packUID)

	siblingUIDs := make([]string, 0, len(children))
	for _, c := range children {
		if uid := c.Manifest.UniqueID; uid != "" {
			siblingUIDs = append(siblingUIDs, uid)
		}
	}

	manifest := first.Manifest
	manifest.Name = name
	manifest.UniqueID = packUID
	manifest.EntryDll = ""
	manifest.UpdateKeys = first.Manifest.UpdateKeys

	groupKey, groupLabel := groupForMod(container, manifest, GroupingFolderCondensed)

	return Mod{
		ID:              packID,
		FolderPath:      container,
		AbsolutePath:    abs,
		Manifest:        manifest,
		Enabled:         resolvePackEnabled(packID, childIDs, enabled),
		GroupKey:        groupKey,
		GroupLabel:      groupLabel,
		HasConfig:       hasConfig,
		HasJsonFiles:    hasJsonFiles,
		JsonFileCount:   jsonFileCount,
		InstallTime:     maxInstall,
		LastUpdated:     maxUpdated,
		PackSiblingUIDs: siblingUIDs,
	}
}

// PackUniqueIDFromManifest returns a stable synthetic UniqueID for a mod pack.
func PackUniqueIDFromManifest(m Manifest) string {
	if id := nexusIDFromUpdateKeys(m.UpdateKeys); id != 0 {
		return fmt.Sprintf("%s%d", PackUniqueIDPrefix, id)
	}
	return "pack:" + m.UniqueID
}

// PackDisplayName derives a user-facing pack name from a container folder basename.
func PackDisplayName(folderBase string) string {
	name := strings.TrimSpace(folderBase)
	name = strings.TrimPrefix(name, "(AT) ")
	return strings.TrimSpace(name)
}

func nexusIDFromUpdateKeys(keys []string) int {
	for _, key := range keys {
		var id int
		if _, err := fmt.Sscanf(key, "Nexus:%d", &id); err == nil && id > 0 {
			return id
		}
	}
	return 0
}

func resolvePackEnabled(packID string, childIDs []string, enabled map[string]bool) bool {
	if enabled != nil {
		if v, ok := enabled[packID]; ok {
			return v
		}
		anySet := false
		anyEnabled := false
		for _, cid := range childIDs {
			if v, ok := enabled[cid]; ok {
				anySet = true
				if v {
					anyEnabled = true
				}
			}
		}
		if anySet {
			return anyEnabled
		}
	}
	return true
}

// IsPackModID reports whether modID refers to a collapsed content-pack mod.
func IsPackModID(modID string) bool {
	return strings.Contains(modID, PackUniqueIDPrefix)
}

// MigratePackEnableState removes stale child mod IDs when toggling a pack mod.
func MigratePackEnableState(enabled map[string]bool, packModID string, enabledValue bool) {
	if enabled == nil {
		return
	}
	folderPath, _, ok := strings.Cut(packModID, "::")
	if !ok || !strings.Contains(packModID, PackUniqueIDPrefix) {
		return
	}
	prefix := folderPath + "/"
	for id := range enabled {
		if strings.HasPrefix(id, prefix) {
			delete(enabled, id)
		}
	}
	enabled[packModID] = enabledValue
}
