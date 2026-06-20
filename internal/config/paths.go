package config

import (
	"os"
	"path/filepath"
)

// ActiveModsDir returns the Stardew Valley Mods folder SMAPI loads at runtime.
func ActiveModsDir(gamePath string) string {
	if gamePath == "" {
		return ""
	}
	return filepath.Join(gamePath, "Mods")
}

// DefaultModLibrary returns the default mod library path under app data.
func DefaultModLibrary(dataDir string) string {
	return filepath.Join(dataDir, "mod-library")
}

// EnsureModLibrary defaults ModsRoot when unset and creates the library directory.
func EnsureModLibrary(settings Settings, dataDir string) (Settings, error) {
	if settings.ModsRoot == "" {
		settings.ModsRoot = DefaultModLibrary(dataDir)
	}
	if err := os.MkdirAll(settings.ModsRoot, 0o755); err != nil {
		return settings, err
	}
	return settings, nil
}
