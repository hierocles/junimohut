package mods

import "errors"

// FashionSenseFrameworkUID is the UniqueID of the Fashion Sense framework mod.
const FashionSenseFrameworkUID = "PeacefulEnd.FashionSense"

// IsFashionSenseRelated reports whether a mod is a Fashion Sense content pack
// (depends on the FS framework via ContentPackFor or Dependencies).
// The framework mod itself is excluded.
func IsFashionSenseRelated(m Manifest) bool {
	if m.UniqueID == FashionSenseFrameworkUID {
		return false
	}
	for _, entry := range collectDependencyEntries(Mod{Manifest: m}) {
		if entry.UniqueID == FashionSenseFrameworkUID {
			return true
		}
	}
	return false
}

// ArchivesContainFashionSense reports whether any manifest in the archives depends on FS.
func ArchivesContainFashionSense(archivePaths []string) (bool, error) {
	for _, archivePath := range archivePaths {
		manifests, err := extractManifestsFromArchive(archivePath)
		if err != nil {
			if errors.Is(err, errOverwritePatchArchive) {
				continue
			}
			return false, err
		}
		for _, manifest := range manifests {
			if IsFashionSenseRelated(manifest) {
				return true, nil
			}
		}
	}
	return false, nil
}
