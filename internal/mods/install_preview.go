package mods

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"junimohut/internal/archive"
)

// PreviewInstallNames extracts archives and reports official vs folder display labels.
func PreviewInstallNames(archivePaths []string) ([]InstallNamePreview, error) {
	var previews []InstallNamePreview
	for _, archivePath := range archivePaths {
		preview, err := previewInstallNamesForArchive(archivePath)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(archivePath), err)
		}
		previews = append(previews, preview)
	}
	return previews, nil
}

func previewInstallNamesForArchive(archivePath string) (InstallNamePreview, error) {
	tmpDir, err := os.MkdirTemp("", "sdvm-name-preview-*")
	if err != nil {
		return InstallNamePreview{}, err
	}
	defer os.RemoveAll(tmpDir)

	if err := archive.Extract(archivePath, tmpDir); err != nil {
		return InstallNamePreview{}, fmt.Errorf("extract archive: %w", err)
	}

	manifests, err := findAllManifests(tmpDir)
	if err != nil {
		return InstallNamePreview{}, err
	}
	manifests = FilterRootManifests(manifests, tmpDir)
	if len(manifests) == 0 {
		return InstallNamePreview{}, fmt.Errorf("no manifest.json found in archive")
	}

	units, err := resolveInstallUnits(tmpDir, manifests)
	if err != nil {
		return InstallNamePreview{}, err
	}

	mods := make([]InstallModNamePreview, 0, len(units))
	for _, unit := range units {
		manifestPath, err := FindManifestPath(unit.srcDir)
		if err != nil {
			return InstallNamePreview{}, err
		}
		manifest, err := ParseManifest(manifestPath)
		if err != nil {
			return InstallNamePreview{}, err
		}
		mods = append(mods, InstallModNamePreview{
			OfficialName: strings.TrimSpace(manifest.Name),
			FolderLabel:  unit.destName,
			DestFolder:   unit.destName,
			UniqueID:     manifest.UniqueID,
		})
	}

	return InstallNamePreview{
		ArchivePath:            archivePath,
		Mods:                   mods,
		NeedsDisplayNameChoice: PreviewNeedsDisplayNameChoice(mods),
	}, nil
}

// PreviewNeedsDisplayNameChoice reports whether users should pick between official
// manifest names and folder-based labels. This applies only when multiple install
// units share the same manifest Name and folder labels would disambiguate them.
func PreviewNeedsDisplayNameChoice(mods []InstallModNamePreview) bool {
	if len(mods) < 2 {
		return false
	}

	byOfficial := map[string][]InstallModNamePreview{}
	for _, mod := range mods {
		key := sanitizeFolderName(mod.OfficialName)
		if key == "" {
			continue
		}
		byOfficial[key] = append(byOfficial[key], mod)
	}

	for _, group := range byOfficial {
		if len(group) < 2 {
			continue
		}
		for _, mod := range group {
			if InstallNameChoiceDiffers(mod.OfficialName, mod.FolderLabel) {
				return true
			}
		}
	}
	return false
}

// InstallNameChoiceDiffers reports whether official and folder labels diverge.
func InstallNameChoiceDiffers(officialName, folderLabel string) bool {
	official := sanitizeFolderName(officialName)
	folder := sanitizeFolderName(folderLabel)
	return official != "" && folder != "" && !strings.EqualFold(official, folder)
}
