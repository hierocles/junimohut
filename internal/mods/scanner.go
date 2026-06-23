package mods

import (
	"os"
	"path/filepath"
	"strings"
)

// ScanOptions configures mod discovery.
type ScanOptions struct {
	ModsRoot            string
	IgnoreHiddenFolders bool
	EnabledMods         map[string]bool
	SkipPackCollapse    bool
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
	if !opts.SkipPackCollapse {
		mods = CollapseSiblingPacks(mods, opts.ModsRoot, opts.EnabledMods)
	}
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

	configPath := filepath.Join(modDir, "config.json")
	_, hasConfig := os.Stat(configPath)
	jsonFileCount := CountJsonFiles(modDir)

	return Mod{
		ID:           id,
		FolderPath:   rel,
		AbsolutePath: modDir,
		Manifest:     manifest,
		Enabled:      enabled,
		UpdateStatus: UpdateStatus{State: "current"},
		HasConfig:    hasConfig == nil,
		HasJsonFiles: jsonFileCount > 0,
		JsonFileCount: jsonFileCount,
		IsCoreMod:    CoreModIDs[manifest.UniqueID],
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

func FilterMods(mods []Mod, search, hideDisabled string) []Mod {
	search = strings.ToLower(strings.TrimSpace(search))
	var out []Mod
	for _, m := range mods {
		if !modMatchesListFilters(m, search, hideDisabled) {
			continue
		}
		out = append(out, m)
	}
	return out
}

func modMatchesListFilters(m Mod, search, hideDisabled string) bool {
	if len(m.BundleChildren) > 0 {
		if search != "" {
			for _, child := range m.BundleChildren {
				if modMatchesListFilters(child, search, "") {
					return true
				}
			}
			return false
		}
		switch hideDisabled {
		case "enabled":
			return m.EnabledCount > 0
		case "disabled":
			return m.EnabledCount < m.EnabledTotal
		default:
			return true
		}
	}

	switch hideDisabled {
	case "enabled":
		if !m.Enabled {
			return false
		}
	case "disabled":
		if m.Enabled {
			return false
		}
	}
	if search == "" {
		return true
	}
	hay := strings.ToLower(m.Manifest.Name + " " + m.CustomName + " " + m.Manifest.Author + " " + m.Manifest.UniqueID + " " + m.FolderPath)
	return strings.Contains(hay, search)
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
