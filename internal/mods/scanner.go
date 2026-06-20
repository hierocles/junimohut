package mods

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ScanOptions configures mod discovery.
type ScanOptions struct {
	ModsRoot            string
	IgnoreHiddenFolders bool
	EnabledMods         map[string]bool
	Grouping            string
}

// Scanner discovers mods in the mods root folder.
type Scanner struct{}

func NewScanner() *Scanner { return &Scanner{} }

// Scan finds all mods under modsRoot.
func (s *Scanner) Scan(opts ScanOptions) ([]Mod, error) {
	if opts.ModsRoot == "" {
		return nil, nil
	}
	if _, err := os.Stat(opts.ModsRoot); os.IsNotExist(err) {
		return nil, nil
	}

	var mods []Mod
	seen := map[string]bool{}

	err := filepath.WalkDir(opts.ModsRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path == opts.ModsRoot {
				return nil
			}
			if opts.IgnoreHiddenFolders && hasHiddenParent(path, opts.ModsRoot) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(d.Name(), "manifest.json") {
			return nil
		}

		if !IsRootModManifest(path, opts.ModsRoot) {
			return nil
		}

		modDir := filepath.Dir(path)
		rel, err := filepath.Rel(opts.ModsRoot, modDir)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if seen[rel] {
			return nil
		}
		seen[rel] = true

		manifest, err := ParseManifest(path)
		if err != nil || manifest.UniqueID == "" {
			return nil
		}

		id := ModID(rel, manifest.UniqueID)
		enabled := true
		if opts.EnabledMods != nil {
			if v, ok := opts.EnabledMods[id]; ok {
				enabled = v
			}
		}
		if CoreModIDs[manifest.UniqueID] {
			enabled = true
		}

		info, _ := d.Info()
		var modTime int64
		if info != nil {
			modTime = info.ModTime().Unix()
		}

		configPath := filepath.Join(modDir, "config.json")
		_, hasConfig := os.Stat(configPath)

		groupKey, groupLabel := groupForMod(rel, manifest, opts.Grouping)

		mods = append(mods, Mod{
			ID:           id,
			FolderPath:   rel,
			AbsolutePath: modDir,
			Manifest:     manifest,
			Enabled:      enabled,
			GroupKey:     groupKey,
			GroupLabel:   groupLabel,
			UpdateStatus: UpdateStatus{State: "current"},
			HasConfig:    hasConfig == nil,
			IsCoreMod:    CoreModIDs[manifest.UniqueID],
			InstallTime:  modTime,
			LastUpdated:  modTime,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	mods = CollapseSiblingPacks(mods, opts.ModsRoot, opts.EnabledMods)
	return mods, nil
}

func hasHiddenParent(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

func groupForMod(rel string, m Manifest, grouping string) (string, string) {
	switch grouping {
	case "contentpack":
		if m.ContentPackFor != nil {
			return "cp:" + m.ContentPackFor.UniqueID, "Content Pack: " + m.ContentPackFor.UniqueID
		}
		return "mod", "Mods"
	case "folder_condensed":
		parts := strings.Split(rel, "/")
		if len(parts) > 1 {
			return "folder:" + parts[0], parts[0]
		}
		return "root", "Root"
	default: // folder
		parts := strings.Split(rel, "/")
		if len(parts) > 1 {
			return "folder:" + strings.Join(parts[:len(parts)-1], "/"), strings.Join(parts[:len(parts)-1], "/")
		}
		return "root", "Root"
	}
}

// GroupMods organizes mods into groups for display.
func GroupMods(mods []Mod) []ModGroup {
	order := []string{}
	groups := map[string]*ModGroup{}
	seen := map[string]bool{}
	for _, m := range mods {
		if seen[m.ID] {
			continue
		}
		seen[m.ID] = true
		g, ok := groups[m.GroupKey]
		if !ok {
			g = &ModGroup{Key: m.GroupKey, Label: m.GroupLabel}
			groups[m.GroupKey] = g
			order = append(order, m.GroupKey)
		}
		g.Mods = append(g.Mods, m)
	}
	result := make([]ModGroup, 0, len(order))
	for _, k := range order {
		result = append(result, *groups[k])
	}
	return result
}

// FilterMods applies search and hide-disabled filters.
func FilterMods(mods []Mod, search, hideDisabled string) []Mod {
	search = strings.ToLower(strings.TrimSpace(search))
	var out []Mod
	for _, m := range mods {
		if hideDisabled == "enabled" && !m.Enabled {
			continue
		}
		if hideDisabled == "disabled" && m.Enabled {
			continue
		}
		if search != "" {
			hay := strings.ToLower(m.Manifest.Name + " " + m.CustomName + " " + m.Manifest.Author + " " + m.Manifest.UniqueID + " " + m.FolderPath)
			if !strings.Contains(hay, search) {
				continue
			}
		}
		out = append(out, m)
	}
	return out
}

// CategoryVisibility drives tag filtering in the mod list.
type CategoryVisibility struct {
	ID      string
	Visible bool
}

// FilterByCategories returns mods matching active tag filters.
// When every tag is visible (or there are no tags), all mods are shown.
// When the filter is narrowed (some tags toggled off), mods with at least one
// remaining visible tag are shown; untagged mods are hidden.
func FilterByCategories(mods []Mod, categories []CategoryVisibility) []Mod {
	if len(categories) == 0 {
		return mods
	}

	visible := map[string]bool{}
	visibleCount := 0
	for _, c := range categories {
		if c.Visible {
			visible[c.ID] = true
			visibleCount++
		}
	}
	if visibleCount == 0 || visibleCount == len(categories) {
		return mods
	}

	var out []Mod
	for _, m := range mods {
		if len(m.CategoryIDs) == 0 {
			continue
		}
		for _, catID := range m.CategoryIDs {
			if visible[catID] {
				out = append(out, m)
				break
			}
		}
	}
	return out
}

// TouchMod updates last-updated timestamp metadata (placeholder for install tracking).
func TouchMod(m *Mod) {
	m.LastUpdated = time.Now().Unix()
}
