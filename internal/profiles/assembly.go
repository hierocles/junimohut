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
		linkName := sanitizeLinkName(m.FolderPath)
		desired[linkName] = m.AbsolutePath
	}

	if err := removeStaleLinks(a.ActiveModsDir, a.ModsRoot, desired); err != nil {
		return err
	}

	for linkName, target := range desired {
		linkPath := filepath.Join(a.ActiveModsDir, linkName)
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
	return filepath.WalkDir(activeDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == activeDir {
			return nil
		}
		if !platform.IsManagedModLink(path, libraryRoot) {
			if d.IsDir() && !platform.IsModLink(path) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(activeDir, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if _, keep := desired[rel]; !keep {
			if err := os.RemoveAll(path); err != nil {
				return err
			}
		}
		return nil
	})
}

func sanitizeLinkName(folderPath string) string {
	return filepath.FromSlash(folderPath)
}
