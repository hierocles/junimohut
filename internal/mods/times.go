package mods

import (
	"os"
	"path/filepath"
)

// ManifestModTime returns manifest.json modification time as Unix seconds.
func ManifestModTime(modDir string) int64 {
	info, err := os.Stat(filepath.Join(modDir, "manifest.json"))
	if err != nil || info == nil {
		return 0
	}
	return info.ModTime().Unix()
}
