package profiles

import (
	"os"
	"path/filepath"

	"junimohut/internal/mods"
	"junimohut/internal/platform"
)

// Assembler builds symlinks in the game Mods folder for enabled mods in the active profile.
type Assembler struct {
	ActiveModsDir string
	ModsRoot      string
}

func NewAssembler(activeModsDir, modsRoot string) *Assembler {
	return &Assembler{ActiveModsDir: activeModsDir, ModsRoot: modsRoot}
}

// Assemble syncs enabled mods from the library into the game Mods folder via symlinks/junctions.
func (a *Assembler) Assemble(modList []mods.Mod, enabled map[string]bool) error {
	if a.ActiveModsDir == "" {
		return nil
	}
	if err := os.MkdirAll(a.ActiveModsDir, 0o755); err != nil {
		return err
	}

	desired := map[string]string{}
	modList = mods.ExpandModsForAssembly(modList)
	for _, m := range modList {
		en := true
		if enabled != nil {
			if v, ok := enabled[m.ID]; ok {
				en = v
			}
		}
		if mods.CoreModIDs[m.Manifest.UniqueID] {
			en = true
		}
		if !en {
			continue
		}
		key := desiredLinkKey(m.FolderPath)
		desired[key] = m.AbsolutePath
	}

	if err := removeStaleLinks(a.ActiveModsDir, a.ModsRoot, desired); err != nil {
		return err
	}

	for linkKey, target := range desired {
		linkPath := filepath.Join(a.ActiveModsDir, filepath.FromSlash(linkKey))
		if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
			return err
		}
		if platform.IsManagedModLink(linkPath, a.ModsRoot) {
			current, err := platform.LinkTarget(linkPath)
			if err == nil {
				absTarget, _ := filepath.Abs(target)
				if filepath.Clean(current) == filepath.Clean(absTarget) {
					continue
				}
			}
		}
		if err := platform.LinkDir(linkPath, target); err != nil {
			return err
		}
	}
	return nil
}

func removeStaleLinks(activeDir, libraryRoot string, desired map[string]string) error {
	if err := walkForManagedLinks(activeDir, activeDir, libraryRoot, desired); err != nil {
		return err
	}
	return removeEmptyDirs(activeDir)
}

func walkForManagedLinks(activeDir, dir, libraryRoot string, desired map[string]string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		if platform.IsManagedModLink(path, libraryRoot) {
			rel, err := filepath.Rel(activeDir, path)
			if err != nil {
				continue
			}
			rel = filepath.ToSlash(rel)
			if _, keep := desired[rel]; !keep {
				if err := os.RemoveAll(path); err != nil {
					return err
				}
			}
			continue
		}
		if !e.IsDir() {
			continue
		}
		// Unmanaged mod installs live in the game Mods folder — never walk their assets.
		if mods.HasManifestInDir(path) {
			continue
		}
		if err := walkForManagedLinks(activeDir, path, libraryRoot, desired); err != nil {
			return err
		}
	}
	return nil
}

// removeEmptyDirs deletes leftover directory shells created for nested symlinks.
func removeEmptyDirs(root string) error {
	return pruneEmptyContainerDirs(root, root)
}

func pruneEmptyContainerDirs(root, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub := filepath.Join(dir, e.Name())
		if platform.IsModLink(sub) || mods.HasManifestInDir(sub) {
			continue
		}
		if err := pruneEmptyContainerDirs(root, sub); err != nil {
			return err
		}
	}
	if dir == root {
		return nil
	}
	entries, err = os.ReadDir(dir)
	if err != nil || len(entries) > 0 {
		return nil
	}
	if err := os.Remove(dir); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// desiredLinkKey normalizes a mod folder path for desired/stale map lookups.
// Keys always use forward slashes so they match removeStaleLinks on every OS.
func desiredLinkKey(folderPath string) string {
	return filepath.ToSlash(folderPath)
}
