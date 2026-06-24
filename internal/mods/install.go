package mods

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"junimohut/internal/archive"
)

// Installer handles mod archive installation.
type Installer struct {
	ModsRoot string
}

func NewInstaller(modsRoot string) *Installer {
	return &Installer{ModsRoot: modsRoot}
}

// InstallResult describes an installed mod.
type InstallResult struct {
	FolderPath string `json:"folderPath"`
	ModID      string `json:"modId"`
	Name       string `json:"name"`
	Error      string `json:"error,omitempty"`
}

// InstallArchive extracts an archive and returns discovered mods.
func (i *Installer) InstallArchive(archivePath string) ([]InstallResult, error) {
	if i.ModsRoot == "" {
		return nil, fmt.Errorf("Mod library path not configured. Set it in Settings.")
	}
	tmpDir, err := os.MkdirTemp("", "sdvm-install-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	if err := archive.Extract(archivePath, tmpDir); err != nil {
		return nil, fmt.Errorf("extract archive: %w", err)
	}

	manifests, err := findAllManifests(tmpDir)
	if err != nil {
		return nil, err
	}
	manifests = FilterRootManifests(manifests, tmpDir)
	if len(manifests) == 0 {
		return nil, fmt.Errorf("no manifest.json found in archive")
	}

	units, err := resolveInstallUnits(tmpDir, manifests)
	if err != nil {
		return nil, err
	}

	var results []InstallResult
	for _, unit := range units {
		destName := unit.destName
		dest, destRel, resolveErr := resolveInstallDestination(i, unit)
		if resolveErr != nil {
			results = append(results, InstallResult{Name: destName, Error: resolveErr.Error()})
			continue
		}
		if err := copyDir(unit.srcDir, dest); err != nil {
			results = append(results, InstallResult{Name: destName, Error: err.Error()})
			continue
		}
		unitResults, err := installResultsForDest(i, destRel)
		if err != nil {
			results = append(results, InstallResult{Name: destName, Error: err.Error()})
			continue
		}
		if len(unitResults) == 0 {
			results = append(results, InstallResult{
				FolderPath: destRel,
				Name:       destName,
			})
			continue
		}
		results = append(results, unitResults...)
	}
	return results, nil
}

func folderInstallCollisionError(destName string) string {
	return fmt.Sprintf(
		"A different mod already uses folder %q — choose a merge target in the install dialog or delete the existing folder.",
		destName,
	)
}

func resolveInstallDestination(installer *Installer, unit installUnit) (dest, destRel string, err error) {
	destName := unit.destName
	dest = filepath.Join(installer.ModsRoot, destName)
	_, statErr := os.Stat(dest)
	if os.IsNotExist(statErr) {
		if conflicts, conflictErr := findInstalledConflictsForUnit(installer.ModsRoot, unit); conflictErr != nil {
			return "", "", conflictErr
		} else if len(conflicts) > 0 {
			return "", "", fmt.Errorf("%s", existingModMergeRequiredError(conflicts[0].FolderPath))
		}
		rel, relErr := filepath.Rel(installer.ModsRoot, dest)
		if relErr != nil {
			return "", "", relErr
		}
		return dest, filepath.ToSlash(rel), nil
	}
	if statErr != nil {
		return "", "", statErr
	}

	incoming, incomingErr := manifestAtModDir(unit.srcDir)
	if incomingErr != nil {
		return "", "", fmt.Errorf("%s", folderInstallCollisionError(destName))
	}

	existing, existingErr := manifestAtModDir(dest)
	if existingErr != nil {
		return "", "", fmt.Errorf("%s", folderInstallCollisionError(destName))
	}
	if !UniqueIDsEqual(incoming.UniqueID, existing.UniqueID) {
		return "", "", fmt.Errorf("%s", folderInstallCollisionError(destName))
	}

	return "", "", fmt.Errorf("%s", existingModMergeRequiredError(destName))
}

func findInstalledConflictsForUnit(modsRoot string, unit installUnit) ([]Mod, error) {
	manifests, err := rootManifestObjectsAtDir(unit.srcDir)
	if err != nil {
		return nil, err
	}
	var conflicts []Mod
	seen := map[string]bool{}
	for _, manifest := range manifests {
		matches, err := FindInstalledModsByUniqueID(modsRoot, manifest.UniqueID)
		if err != nil {
			return nil, err
		}
		for _, mod := range matches {
			if seen[mod.FolderPath] {
				continue
			}
			seen[mod.FolderPath] = true
			conflicts = append(conflicts, mod)
		}
	}
	return conflicts, nil
}

func rootManifestObjectsAtDir(dir string) ([]Manifest, error) {
	paths, err := findAllManifests(dir)
	if err != nil {
		return nil, err
	}
	paths = FilterRootManifests(paths, dir)
	manifests := make([]Manifest, 0, len(paths))
	for _, path := range paths {
		manifest, err := ParseManifest(path)
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, manifest)
	}
	return manifests, nil
}

func existingModMergeRequiredError(destName string) string {
	return fmt.Sprintf(
		"Mod already installed at %q — select it as a merge target in the install dialog to update it.",
		destName,
	)
}

func manifestAtModDir(dir string) (Manifest, error) {
	path, err := FindManifestPath(dir)
	if err != nil {
		return Manifest{}, err
	}
	return ParseManifest(path)
}

func findAllManifests(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.EqualFold(d.Name(), "manifest.json") {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

func sanitizeFolderName(name string) string {
	name = strings.TrimSpace(name)
	replacer := strings.NewReplacer(
		"<", "", ">", "", ":", "", `"`, "", "/", "", `\`, "", "|", "", "?", "", "*", "",
	)
	return replacer.Replace(name)
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if strings.HasPrefix(rel, "__MACOSX") || strings.Contains(rel, ".DS_Store") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// UpdateMods applies an archive to each folder path (e.g. Nexus multi-part mods).
func (i *Installer) UpdateMods(folderPaths []string, archivePath string, deleteOld bool) error {
	for _, folderPath := range folderPaths {
		if err := i.UpdateMod(folderPath, archivePath, deleteOld); err != nil {
			return fmt.Errorf("%s: %w", folderPath, err)
		}
	}
	return nil
}

// DeleteMod removes a mod folder from the mods root.
func (i *Installer) DeleteMod(folderPath string) error {
	abs := filepath.Join(i.ModsRoot, filepath.FromSlash(folderPath))
	return os.RemoveAll(abs)
}

// UpdateMod replaces mod files from an archive, optionally deleting old files first.
func (i *Installer) UpdateMod(folderPath, archivePath string, deleteOld bool) error {
	dest := filepath.Join(i.ModsRoot, filepath.FromSlash(folderPath))
	if deleteOld {
		entries, _ := os.ReadDir(dest)
		for _, e := range entries {
			name := e.Name()
			if strings.EqualFold(name, "config.json") {
				continue
			}
			_ = os.RemoveAll(filepath.Join(dest, name))
		}
	}
	tmpDir, err := os.MkdirTemp("", "sdvm-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	if err := archive.Extract(archivePath, tmpDir); err != nil {
		return err
	}
	manifests, err := findAllManifests(tmpDir)
	if err != nil {
		return err
	}
	manifests = FilterRootManifests(manifests, tmpDir)
	srcDir, err := resolveUpdateSourceDir(tmpDir, manifests, dest)
	if err != nil {
		return err
	}
	if err := copyDir(srcDir, dest); err != nil {
		return err
	}
	return writeUpdateManifest(dest, manifests)
}

func writeUpdateManifest(dest string, rootManifests []string) error {
	incoming, err := pickUpdateManifest(rootManifests, dest)
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(incoming)
	if err != nil {
		return err
	}
	destManifest := filepath.Join(dest, "manifest.json")
	if existing, err := ParseManifest(destManifest); err == nil && existing.UniqueID != "" {
		incomingManifest, err := ParseManifest(incoming)
		if err == nil && !UniqueIDsEqual(existing.UniqueID, incomingManifest.UniqueID) {
			return nil
		}
	}
	return os.WriteFile(destManifest, data, 0o644)
}

func pickUpdateManifest(rootManifests []string, existingFolder string) (string, error) {
	if len(rootManifests) == 0 {
		return "", fmt.Errorf("no manifest.json found in archive")
	}
	if len(rootManifests) == 1 {
		return rootManifests[0], nil
	}
	if existingManifest, err := FindManifestPath(existingFolder); err == nil {
		existing, err := ParseManifest(existingManifest)
		if err == nil {
			for _, mf := range rootManifests {
				incoming, err := ParseManifest(mf)
				if err == nil && UniqueIDsEqual(incoming.UniqueID, existing.UniqueID) {
					return mf, nil
				}
			}
		}
	}
	folderBase := filepath.Base(existingFolder)
	for _, mf := range rootManifests {
		if filepath.Base(filepath.Dir(mf)) == folderBase {
			return mf, nil
		}
	}
	return rootManifests[0], nil
}
