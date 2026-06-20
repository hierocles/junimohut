package mods

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type installUnit struct {
	srcDir   string
	destName string
}

func resolveInstallUnits(extractRoot string, rootManifests []string) ([]installUnit, error) {
	if wrapper, ok := singleTopLevelWrapper(extractRoot); ok {
		if units, ok := variantSplitUnits(wrapper); ok {
			disambiguateInstallDestNames(units)
			return units, nil
		}
		return []installUnit{{
			srcDir:   wrapper,
			destName: filepath.Base(wrapper),
		}}, nil
	}

	units := make([]installUnit, 0, len(rootManifests))
	for _, mf := range rootManifests {
		manifest, err := ParseManifest(mf)
		if err != nil {
			return nil, err
		}
		destName := sanitizeFolderName(manifest.Name)
		if destName == "" {
			destName = filepath.Base(filepath.Dir(mf))
		}
		units = append(units, installUnit{
			srcDir:   filepath.Dir(mf),
			destName: destName,
		})
	}
	disambiguateInstallDestNames(units)
	return units, nil
}

// disambiguateInstallDestNames assigns each unit its source folder name when multiple
// mods would otherwise install into the same destination (e.g. CP color variants sharing
// an identical manifest Name).
func disambiguateInstallDestNames(units []installUnit) {
	if len(units) < 2 {
		return
	}

	nameCount := map[string]int{}
	for _, u := range units {
		nameCount[u.destName]++
	}
	hasCollidingNames := false
	for _, count := range nameCount {
		if count > 1 {
			hasCollidingNames = true
			break
		}
	}
	if !hasCollidingNames {
		return
	}

	used := map[string]int{}
	for i := range units {
		base := filepath.Base(units[i].srcDir)
		dest := base
		if n := used[dest]; n > 0 {
			if manifest, err := ParseManifest(filepath.Join(units[i].srcDir, "manifest.json")); err == nil && manifest.UniqueID != "" {
				dest = base + "_" + sanitizeFolderName(manifest.UniqueID)
			} else {
				dest = base + "_" + fmt.Sprint(n+1)
			}
		}
		used[dest]++
		units[i].destName = dest
	}
}

func singleTopLevelWrapper(extractRoot string) (string, bool) {
	entries, err := os.ReadDir(extractRoot)
	if err != nil {
		return "", false
	}

	var dirs []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "__MACOSX") || name == ".DS_Store" {
			continue
		}
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(extractRoot, name))
			continue
		}
		if strings.EqualFold(name, "manifest.json") {
			return "", false
		}
	}
	if len(dirs) != 1 {
		return "", false
	}
	if HasManifestInDir(extractRoot) {
		return "", false
	}
	return dirs[0], true
}

// variantSplitUnits splits a wrapper folder into separate installs when it contains
// sibling mods targeting different content-pack frameworks (e.g. [AT] + [CP] bundles)
// or multiple variants that share the same manifest Name.
func variantSplitUnits(wrapperDir string) ([]installUnit, bool) {
	manifests, err := findAllManifests(wrapperDir)
	if err != nil {
		return nil, false
	}
	manifests = FilterRootManifests(manifests, wrapperDir)
	if !areDirectSiblingManifests(wrapperDir, manifests) {
		return nil, false
	}
	if !shouldSplitSiblingVariants(manifests) {
		return nil, false
	}

	units := make([]installUnit, 0, len(manifests))
	for _, mf := range manifests {
		manifest, err := ParseManifest(mf)
		if err != nil {
			return nil, false
		}
		destName := sanitizeFolderName(manifest.Name)
		if destName == "" {
			destName = filepath.Base(filepath.Dir(mf))
		}
		units = append(units, installUnit{
			srcDir:   filepath.Dir(mf),
			destName: destName,
		})
	}
	return units, true
}

func areDirectSiblingManifests(wrapperDir string, rootManifests []string) bool {
	if len(rootManifests) < 2 {
		return false
	}
	wrapperDir = filepath.Clean(wrapperDir)
	for _, mf := range rootManifests {
		mf = filepath.Clean(mf)
		parent := filepath.Dir(mf)
		if parent == wrapperDir {
			return false
		}
		if filepath.Dir(parent) != wrapperDir {
			return false
		}
	}
	return true
}

func shouldSplitSiblingVariants(manifestPaths []string) bool {
	if len(manifestPaths) < 2 {
		return false
	}
	if !areSplittableContentPackSiblings(manifestPaths) {
		return false
	}
	if hasMultipleFrameworks(manifestPaths) {
		return true
	}
	return hasCollidingManifestNames(manifestPaths)
}

func areSplittableContentPackSiblings(manifestPaths []string) bool {
	parsed := make([]Manifest, 0, len(manifestPaths))
	for _, mf := range manifestPaths {
		manifest, err := ParseManifest(mf)
		if err != nil {
			return false
		}
		if manifest.EntryDll != "" || manifest.ContentPackFor == nil {
			return false
		}
		parsed = append(parsed, manifest)
	}

	siblingUIDs := map[string]bool{}
	for _, manifest := range parsed {
		if manifest.UniqueID != "" {
			siblingUIDs[manifest.UniqueID] = true
		}
	}

	for _, manifest := range parsed {
		for _, dep := range manifest.Dependencies {
			if dep.UniqueID == "" {
				continue
			}
			required := dep.IsRequired == nil || bool(*dep.IsRequired)
			if required && siblingUIDs[dep.UniqueID] {
				return false
			}
		}
	}
	return true
}

func hasMultipleFrameworks(manifestPaths []string) bool {
	frameworks := map[string]bool{}
	for _, mf := range manifestPaths {
		manifest, err := ParseManifest(mf)
		if err != nil {
			return false
		}
		if manifest.ContentPackFor != nil {
			frameworks[manifest.ContentPackFor.UniqueID] = true
		}
	}
	return len(frameworks) >= 2
}

func hasCollidingManifestNames(manifestPaths []string) bool {
	seen := map[string]bool{}
	for _, mf := range manifestPaths {
		manifest, err := ParseManifest(mf)
		if err != nil {
			return false
		}
		name := sanitizeFolderName(manifest.Name)
		if name == "" {
			return false
		}
		if seen[name] {
			return true
		}
		seen[name] = true
	}
	return false
}

// isOptionalFrameworkVariantBundle reports whether sibling manifests are optional AT/CP-style
// variants (split install) rather than a required multi-part expansion bundle.
func isOptionalFrameworkVariantBundle(manifestPaths []string) bool {
	return shouldSplitSiblingVariants(manifestPaths)
}

func installResultsForDest(installer *Installer, destRel string) ([]InstallResult, error) {
	destAbs := filepath.Join(installer.ModsRoot, filepath.FromSlash(destRel))
	manifests, err := findAllManifests(destAbs)
	if err != nil {
		return nil, err
	}
	manifests = FilterRootManifests(manifests, destAbs)
	if len(manifests) == 0 {
		return nil, nil
	}

	var discovered []Mod
	for _, mf := range manifests {
		manifest, err := ParseManifest(mf)
		if err != nil {
			continue
		}
		modDir := filepath.Dir(mf)
		rel, err := filepath.Rel(installer.ModsRoot, modDir)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		info, _ := os.Stat(mf)
		var modTime int64
		if info != nil {
			modTime = info.ModTime().Unix()
		}
		discovered = append(discovered, Mod{
			ID:           ModID(rel, manifest.UniqueID),
			FolderPath:   rel,
			AbsolutePath: modDir,
			Manifest:     manifest,
			InstallTime:  modTime,
			LastUpdated:  modTime,
		})
	}

	collapsed := CollapseSiblingPacks(discovered, installer.ModsRoot, nil)
	results := make([]InstallResult, 0, len(collapsed))
	for _, m := range collapsed {
		results = append(results, InstallResult{
			FolderPath: m.FolderPath,
			ModID:      m.ID,
			Name:       m.Manifest.Name,
		})
	}
	return results, nil
}

func resolveUpdateSourceDir(extractRoot string, rootManifests []string, dest string) (string, error) {
	units, err := resolveInstallUnits(extractRoot, rootManifests)
	if err != nil {
		return "", err
	}
	if len(units) == 1 {
		return units[0].srcDir, nil
	}
	destBase := filepath.Base(dest)
	for _, unit := range units {
		if filepath.Base(unit.srcDir) == destBase || unit.destName == destBase {
			return unit.srcDir, nil
		}
	}
	manifestPath, err := pickUpdateManifest(rootManifests, dest)
	if err != nil {
		return "", err
	}
	return filepath.Dir(manifestPath), nil
}
