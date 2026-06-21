package profiles

import (
	"os"
	"path/filepath"
	"strings"

	"junimohut/internal/mods"
	"junimohut/internal/platform"
)

// UnmanagedMod describes a folder in the game's Mods directory that Junimo Hut does not manage.
type UnmanagedMod struct {
	FolderName string `json:"folderName"`
	Name       string `json:"name"`
	UniqueID   string `json:"uniqueID,omitempty"`
}

// ScanUnmanagedMods finds entries in the active Mods folder that are not Junimo Hut symlinks
// and are not SMAPI's three bundled helper mods.
func ScanUnmanagedMods(activeDir, libraryRoot string) ([]UnmanagedMod, error) {
	if activeDir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(activeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var out []UnmanagedMod
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		path := filepath.Join(activeDir, e.Name())
		if isAllowedActiveEntry(path, libraryRoot) {
			continue
		}
		entry := UnmanagedMod{FolderName: e.Name()}
		if name, uid := modDisplayAt(path); name != "" {
			entry.Name = name
			entry.UniqueID = uid
		} else {
			entry.Name = e.Name()
		}
		out = append(out, entry)
	}
	return out, nil
}

func isAllowedActiveEntry(path, libraryRoot string) bool {
	if platform.IsManagedModLink(path, libraryRoot) {
		return true
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || platform.IsModLink(path) {
		return false
	}
	if isCoreModDir(path) {
		return true
	}
	return isManagedOnlyContainer(path, libraryRoot)
}

func isCoreModDir(dir string) bool {
	manifestPath, err := mods.FindManifestPath(dir)
	if err != nil {
		return false
	}
	manifest, err := mods.ParseManifest(manifestPath)
	if err != nil {
		return false
	}
	return mods.CoreModIDs[manifest.UniqueID]
}

func isManagedOnlyContainer(dir, libraryRoot string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	if len(entries) == 0 {
		return true
	}
	for _, e := range entries {
		child := filepath.Join(dir, e.Name())
		if platform.IsManagedModLink(child, libraryRoot) {
			continue
		}
		info, err := os.Lstat(child)
		if err != nil {
			return false
		}
		if info.IsDir() && !platform.IsModLink(child) && isManagedOnlyContainer(child, libraryRoot) {
			continue
		}
		return false
	}
	return true
}

func modDisplayAt(path string) (name, uniqueID string) {
	manifestPath, err := mods.FindManifestPath(path)
	if err != nil {
		return "", ""
	}
	manifest, err := mods.ParseManifest(manifestPath)
	if err != nil {
		return "", ""
	}
	return manifest.Name, manifest.UniqueID
}
