package mods

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// PackUniqueIDPrefix marks synthetic pack mod IDs derived from Nexus update keys.
const PackUniqueIDPrefix = "pack:nexus:"

// CollapseSiblingPacks merges mods that share a Nexus mod ID into one bundle row.
func CollapseSiblingPacks(mods []Mod, modsRoot string, enabled map[string]bool) []Mod {
	_ = modsRoot
	return CollapseNexusBundles(mods, enabled)
}

// CollapseNexusBundles merges two or more mods with the same Nexus update key into one bundle.
func CollapseNexusBundles(mods []Mod, enabled map[string]bool) []Mod {
	if len(mods) == 0 {
		return mods
	}

	byNexus := map[int][]Mod{}
	for _, m := range mods {
		if m.IsCoreMod {
			continue
		}
		nexusID := nexusIDFromUpdateKeys(m.Manifest.UpdateKeys)
		if nexusID == 0 {
			continue
		}
		byNexus[nexusID] = append(byNexus[nexusID], m)
	}

	collapseNexus := map[int]bool{}
	for nexusID, group := range byNexus {
		if len(group) >= 2 {
			collapseNexus[nexusID] = true
		}
	}
	if len(collapseNexus) == 0 {
		return mods
	}

	out := make([]Mod, 0, len(mods))
	emitted := map[int]bool{}
	for _, m := range mods {
		nexusID := nexusIDFromUpdateKeys(m.Manifest.UpdateKeys)
		if nexusID != 0 && collapseNexus[nexusID] {
			if emitted[nexusID] {
				continue
			}
			emitted[nexusID] = true
			group := byNexus[nexusID]
			sort.Slice(group, func(i, j int) bool {
				return group[i].FolderPath < group[j].FolderPath
			})
			out = append(out, buildNexusBundle(nexusID, group, enabled))
			continue
		}
		out = append(out, m)
	}
	return out
}

func PackUniqueID(nexusID int) string {
	return fmt.Sprintf("%s%d", PackUniqueIDPrefix, nexusID)
}

func buildNexusBundle(nexusID int, children []Mod, enabled map[string]bool) Mod {
	packUID := PackUniqueID(nexusID)
	folderPath := bundleFolderPath(children)
	packID := ModID(folderPath, packUID)

	childIDs := make([]string, len(children))
	siblingUIDs := make([]string, 0, len(children))
	var maxInstall, maxUpdated int64
	hasConfig := false
	hasJsonFiles := false
	jsonFileCount := 0
	containsOverwrites := false

	for i, c := range children {
		childIDs[i] = c.ID
		if uid := c.Manifest.UniqueID; uid != "" {
			siblingUIDs = append(siblingUIDs, uid)
		}
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
		if c.ContainsOverwrites {
			containsOverwrites = true
		}
	}

	primary := pickBundlePrimaryChild(children)
	manifest := primary.Manifest
	manifest.Name = bundleDisplayName(children)
	manifest.UniqueID = packUID
	manifest.EntryDll = ""
	manifest.Version = newestChildVersion(children)
	if len(manifest.UpdateKeys) == 0 {
		manifest.UpdateKeys = []string{fmt.Sprintf("Nexus:%d", nexusID)}
	}

	bundleChildren := append([]Mod{}, children...)
	clearBundleChildUpdateStatus(bundleChildren)
	enabledState := resolveBundleEnabled(childIDs, enabled)

	return Mod{
		ID:                     packID,
		FolderPath:             folderPath,
		AbsolutePath:           primary.AbsolutePath,
		Manifest:               manifest,
		Enabled:                enabledState.enabledCount > 0,
		EnabledPartial:         enabledState.partial,
		EnabledCount:           enabledState.enabledCount,
		EnabledTotal:           len(children),
		CategoryIDs:            append([]string{}, primary.CategoryIDs...),
		UpdateStatus:           mergeBundleUpdateStatus(children),
		HasConfig:              hasConfig,
		HasJsonFiles:           hasJsonFiles,
		JsonFileCount:          jsonFileCount,
		InstallTime:            maxInstall,
		LastUpdated:            maxUpdated,
		PackSiblingUIDs:        siblingUIDs,
		BundleChildren:         bundleChildren,
		BundleNexusID:          nexusID,
		ContainsOverwrites:     containsOverwrites,
		DependencyIssues:       mergeBundleDependencyIssues(children),
		MissingDependencyCount: sumMissingDependencyCount(children),
		CustomName:             primary.CustomName,
	}
}

type bundleEnabledState struct {
	all          bool
	partial      bool
	enabledCount int
}

func resolveBundleEnabled(childIDs []string, enabled map[string]bool) bundleEnabledState {
	if len(childIDs) == 0 {
		return bundleEnabledState{all: true}
	}
	enabledCount := 0
	explicit := 0
	for _, id := range childIDs {
		if v, ok := enabled[id]; ok {
			explicit++
			if v {
				enabledCount++
			}
		} else {
			enabledCount++
		}
	}
	if explicit == 0 {
		return bundleEnabledState{all: true, enabledCount: len(childIDs)}
	}
	all := enabledCount == len(childIDs)
	none := enabledCount == 0
	return bundleEnabledState{
		all:          all,
		partial:      !all && !none,
		enabledCount: enabledCount,
	}
}

func bundleFolderPath(children []Mod) string {
	if len(children) == 0 {
		return ""
	}
	paths := make([]string, len(children))
	for i, c := range children {
		paths[i] = c.FolderPath
	}
	prefix := commonFolderPrefix(paths)
	if prefix != "" {
		return prefix
	}
	if len(children) == 1 {
		return children[0].FolderPath
	}
	return ""
}

func commonFolderPrefix(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	split := func(p string) []string {
		if p == "" {
			return nil
		}
		return strings.Split(filepath.ToSlash(p), "/")
	}
	parts := split(paths[0])
	for _, path := range paths[1:] {
		other := split(path)
		i := 0
		for i < len(parts) && i < len(other) && parts[i] == other[i] {
			i++
		}
		parts = parts[:i]
		if len(parts) == 0 {
			return ""
		}
	}
	return strings.Join(parts, "/")
}

func pickBundlePrimaryChild(children []Mod) Mod {
	for _, c := range children {
		if strings.TrimSpace(c.CustomName) != "" {
			return c
		}
	}
	for _, c := range children {
		if c.Manifest.EntryDll != "" {
			return c
		}
	}
	return children[0]
}

func newestChildVersion(children []Mod) string {
	best := ""
	for _, c := range children {
		v := strings.TrimSpace(c.Manifest.Version)
		if v == "" {
			continue
		}
		if best == "" || versionGreater(v, best) {
			best = v
		}
	}
	return best
}

func bundleDisplayName(children []Mod) string {
	if prefix := bundleFolderPath(children); prefix != "" {
		if name := PackDisplayName(filepath.Base(prefix)); name != "" {
			return name
		}
	}
	counts := map[string]int{}
	var best string
	bestScore := -1
	for _, c := range children {
		name := strings.TrimSpace(c.CustomName)
		if name == "" {
			name = stripModNamePrefix(c.Manifest.Name)
		}
		if name == "" {
			name = PackDisplayName(filepath.Base(c.FolderPath))
		}
		counts[name]++
		score := counts[name]
		if score > bestScore || (score == bestScore && len(name) > len(best)) {
			best = name
			bestScore = score
		}
	}
	if best != "" {
		return best
	}
	return "Mod bundle"
}

func stripModNamePrefix(name string) string {
	name = strings.TrimSpace(name)
	for {
		changed := false
		for _, prefix := range []string{"[CP] ", "[AT] ", "(CP) ", "(AT) ", "[CP]", "[AT]"} {
			if strings.HasPrefix(name, prefix) {
				name = strings.TrimSpace(strings.TrimPrefix(name, prefix))
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return name
}

// StripBundleChildUpdateStatus clears update status on bundle parts so only the
// synthetic parent row owns Nexus update state.
func StripBundleChildUpdateStatus(list []Mod) {
	for i := range list {
		if len(list[i].BundleChildren) == 0 {
			continue
		}
		clearBundleChildUpdateStatus(list[i].BundleChildren)
	}
}

func clearBundleChildUpdateStatus(children []Mod) {
	for i := range children {
		children[i].UpdateStatus = UpdateStatus{}
	}
}

func mergeBundleUpdateStatus(children []Mod) UpdateStatus {
	priority := map[string]int{
		"incompatible":      0,
		"update":            1,
		"update_available":  1,
		"update_ignored":    2,
		"unofficial":        3,
		"current":           4,
	}
	best := UpdateStatus{State: "current"}
	bestPri := 99
	for _, c := range children {
		state := c.UpdateStatus.State
		if state == "" {
			state = "current"
		}
		pri := priority[state]
		if pri < bestPri {
			bestPri = pri
			best = c.UpdateStatus
		}
	}
	return best
}

func mergeBundleDependencyIssues(children []Mod) []DependencyIssue {
	var out []DependencyIssue
	seen := map[string]bool{}
	for _, c := range children {
		for _, issue := range c.DependencyIssues {
			key := issue.UniqueID + "|" + issue.State
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, issue)
		}
	}
	return out
}

func sumMissingDependencyCount(children []Mod) int {
	total := 0
	for _, c := range children {
		total += c.MissingDependencyCount
	}
	return total
}

// PackUniqueIDFromManifest returns a stable synthetic UniqueID for a mod pack.
func PackUniqueIDFromManifest(m Manifest) string {
	if id := nexusIDFromUpdateKeys(m.UpdateKeys); id != 0 {
		return PackUniqueID(id)
	}
	return "pack:" + m.UniqueID
}

// PackDisplayName derives a user-facing pack name from a container folder basename.
func PackDisplayName(folderBase string) string {
	name := strings.TrimSpace(folderBase)
	name = strings.TrimPrefix(name, "(AT) ")
	return strings.TrimSpace(name)
}

// NexusModIDFromUpdateKeys returns the first Nexus mod ID in manifest UpdateKeys.
func NexusModIDFromUpdateKeys(keys []string) int {
	return nexusIDFromUpdateKeys(keys)
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

// IsPackModID reports whether modID refers to a collapsed bundle mod.
func IsPackModID(modID string) bool {
	return strings.Contains(modID, PackUniqueIDPrefix)
}

// IsBundleMod reports whether m is a synthetic Nexus bundle parent row.
func IsBundleMod(m Mod) bool {
	return len(m.BundleChildren) > 0
}

// BundleChildIDs returns mod IDs for bundle children when m is a bundle.
func BundleChildIDs(m Mod) []string {
	if len(m.BundleChildren) == 0 {
		return nil
	}
	ids := make([]string, len(m.BundleChildren))
	for i, child := range m.BundleChildren {
		ids[i] = child.ID
	}
	return ids
}

// MigratePackEnableState normalizes profile enable state when toggling a bundle.
func MigratePackEnableState(enabled map[string]bool, packModID string, enabledValue bool) {
	if enabled == nil {
		return
	}
	delete(enabled, packModID)
	folderPath, _, ok := strings.Cut(packModID, "::")
	if ok && folderPath != "" {
		prefix := folderPath + "/"
		for id := range enabled {
			if strings.HasPrefix(id, prefix) {
				delete(enabled, id)
			}
		}
	}
}

// SetBundleChildrenEnabled writes per-part enable state for a bundle.
func SetBundleChildrenEnabled(enabled map[string]bool, childIDs []string, enabledValue bool) {
	if enabled == nil {
		return
	}
	for _, id := range childIDs {
		enabled[id] = enabledValue
	}
}

// ExpandModsForAssembly returns physical mods to link for profile assembly.
func ExpandModsForAssembly(modList []Mod) []Mod {
	out := make([]Mod, 0, len(modList))
	for _, m := range modList {
		if len(m.BundleChildren) > 0 {
			out = append(out, m.BundleChildren...)
			continue
		}
		out = append(out, m)
	}
	return out
}

// BundleDeleteFolderPaths returns folder paths that should be removed for a mod row.
func BundleDeleteFolderPaths(m Mod) []string {
	if len(m.BundleChildren) == 0 {
		if m.FolderPath == "" {
			return nil
		}
		return []string{m.FolderPath}
	}
	paths := make([]string, 0, len(m.BundleChildren))
	for _, child := range m.BundleChildren {
		if child.FolderPath != "" {
			paths = append(paths, child.FolderPath)
		}
	}
	return paths
}
