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
	s.scanDir(opts, opts.ModsRoot, &mods, seen)
	mods = CollapseSiblingPacks(mods, opts.ModsRoot, opts.EnabledMods)
	return mods, nil
}

// scanDir walks the library tree without descending into mod asset folders.
func (s *Scanner) scanDir(opts ScanOptions, dir string, mods *[]Mod, seen map[string]bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var manifestPath string
	var subdirs []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			subpath := filepath.Join(dir, name)
			if opts.IgnoreHiddenFolders && (strings.HasPrefix(name, ".") || hasHiddenParent(subpath, opts.ModsRoot)) {
				continue
			}
			subdirs = append(subdirs, subpath)
			continue
		}
		if manifestPath == "" && strings.EqualFold(name, "manifest.json") {
			manifestPath = filepath.Join(dir, name)
		}
	}

	if manifestPath != "" && IsRootModManifest(manifestPath, opts.ModsRoot) {
		if mod, ok := modFromManifest(opts, manifestPath, seen); ok {
			*mods = append(*mods, mod)
		}
		return
	}

	for _, sub := range subdirs {
		s.scanDir(opts, sub, mods, seen)
	}
}

func modFromManifest(opts ScanOptions, manifestPath string, seen map[string]bool) (Mod, bool) {
	modDir := filepath.Dir(manifestPath)
	rel, err := filepath.Rel(opts.ModsRoot, modDir)
	if err != nil {
		return Mod{}, false
	}
	rel = filepath.ToSlash(rel)
	if seen[rel] {
		return Mod{}, false
	}
	seen[rel] = true

	manifest, err := ParseManifest(manifestPath)
	if err != nil || manifest.UniqueID == "" {
		return Mod{}, false
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

	info, _ := os.Stat(manifestPath)
	var modTime int64
	if info != nil {
		modTime = info.ModTime().Unix()
	}

	configPath := filepath.Join(modDir, "config.json")
	_, hasConfig := os.Stat(configPath)
	groupKey, groupLabel := groupForMod(rel, manifest, opts.Grouping)

	return Mod{
		ID:           id,
		FolderPath:   rel,
		AbsolutePath: modDir,
		Manifest:     manifest,
		Enabled:      enabled,
		GroupKey:     groupKey,
		GroupLabel:   groupLabel,
		UpdateStatus: UpdateStatus{State: "current"},
		HasConfig:    hasConfig == nil,
		// Cheap signal for the config editor; full JSON tree is counted on demand.
		HasJsonFiles: hasConfig == nil,
		IsCoreMod:    CoreModIDs[manifest.UniqueID],
		InstallTime:  modTime,
		LastUpdated:  modTime,
	}, true
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
