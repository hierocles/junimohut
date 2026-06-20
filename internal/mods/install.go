package mods

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
		dest := filepath.Join(i.ModsRoot, destName)
		if _, err := os.Stat(dest); err == nil {
			dest = filepath.Join(i.ModsRoot, destName+"_"+time.Now().Format("20060102_150405"))
		}
		if err := copyDir(unit.srcDir, dest); err != nil {
			results = append(results, InstallResult{Name: destName, Error: err.Error()})
			continue
		}
		rel, _ := filepath.Rel(i.ModsRoot, dest)
		destRel := filepath.ToSlash(rel)
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
			if strings.EqualFold(name, "config.json") || strings.EqualFold(name, "manifest.json") {
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
	return copyDir(srcDir, dest)
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
				if err == nil && incoming.UniqueID == existing.UniqueID {
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
