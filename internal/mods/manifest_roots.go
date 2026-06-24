package mods

import (
	"os"
	"path/filepath"
)

// HasManifestInDir reports whether dir contains a manifest.json (case-insensitive).
func HasManifestInDir(dir string) bool {
	_, err := FindManifestPath(dir)
	return err == nil
}

func hasDirectManifestFile(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "manifest.json"))
	return err == nil
}

// IsRootModManifest reports whether manifestPath is the outermost manifest under searchRoot.
// Nested manifests inside another mod folder (ancestor also has manifest.json) return false.
func IsRootModManifest(manifestPath, searchRoot string) bool {
	manifestPath = filepath.Clean(manifestPath)
	searchRoot = filepath.Clean(searchRoot)

	dir := filepath.Dir(manifestPath)
	for {
		if dir == searchRoot {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir || len(dir) < len(searchRoot) {
			return true
		}
		if hasDirectManifestFile(parent) {
			return false
		}
		dir = parent
	}
}

// FilterRootManifests returns only outermost manifests, preserving input order.
func FilterRootManifests(paths []string, searchRoot string) []string {
	if len(paths) == 0 {
		return nil
	}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if IsRootModManifest(path, searchRoot) {
			out = append(out, path)
		}
	}
	return out
}
