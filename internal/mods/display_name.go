package mods

import (
	"path/filepath"
	"strings"
)

const (
	DisplayNameOfficial = "official"
	DisplayNameFolder   = "folder"
)

// DefaultDisplayName returns a folder-based label when the install folder is more
// specific than the manifest Name (e.g. "[CP] Seasonal Open Windows - BIRCH" vs
// manifest Name "[CP] Seasonal Open Windows").
func DefaultDisplayName(folderPath, manifestName string) string {
	official := strings.TrimSpace(manifestName)
	if official == "" {
		return ""
	}
	leaf := filepath.Base(filepath.FromSlash(folderPath))
	if leaf == "" || leaf == "." {
		return ""
	}
	if strings.EqualFold(sanitizeFolderName(leaf), sanitizeFolderName(official)) {
		return ""
	}
	return leaf
}

// EffectiveCustomName returns the user-defined custom name, optionally followed
// by a folder-derived default when source is DisplayNameFolder.
func EffectiveCustomName(storedCustomName, folderPath, manifestName, source string) string {
	if name := strings.TrimSpace(storedCustomName); name != "" {
		return name
	}
	if source != DisplayNameFolder {
		return ""
	}
	return DefaultDisplayName(folderPath, manifestName)
}

// InstallResultDisplayName returns the mod list label shown after install.
func InstallResultDisplayName(folderPath, manifestName string, useFolderDisplayNames bool) string {
	official := strings.TrimSpace(manifestName)
	if useFolderDisplayNames {
		if label := DefaultDisplayName(folderPath, official); label != "" {
			return label
		}
	}
	if official != "" {
		return official
	}
	leaf := filepath.Base(filepath.FromSlash(folderPath))
	if leaf != "" && leaf != "." {
		return leaf
	}
	return "Archive"
}
