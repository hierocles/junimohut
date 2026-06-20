package mods

import "fmt"

// ArchiveIndex tracks saved download archives for optional cleanup on mod delete.
type ArchiveIndex interface {
	FindForMod(uniqueID string, nexusModID int) (archivePath string, ok bool)
	Delete(archivePath string) error
}

// ModResolver looks up mod metadata by folder path relative to the mods root.
type ModResolver func(folderPath string) (Mod, bool)

// NexusModIDFromKeys extracts a Nexus mod ID from manifest UpdateKeys.
type NexusModIDFromKeys func(keys []string) int

// DeleteMods removes mod folders and optionally their saved download archives.
func DeleteMods(
	installer *Installer,
	folderPaths []string,
	deleteArchives bool,
	resolve ModResolver,
	archives ArchiveIndex,
	nexusModID NexusModIDFromKeys,
) DeleteModsResult {
	var result DeleteModsResult
	if installer == nil {
		return result
	}

	for _, folderPath := range folderPaths {
		if folderPath == "" {
			continue
		}

		var archivePath string
		if deleteArchives && resolve != nil && archives != nil && nexusModID != nil {
			if mod, ok := resolve(folderPath); ok {
				if path, ok := archives.FindForMod(mod.Manifest.UniqueID, nexusModID(mod.Manifest.UpdateKeys)); ok {
					archivePath = path
				}
			}
		}

		if err := installer.DeleteMod(folderPath); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", folderPath, err))
			continue
		}
		result.DeletedCount++

		if deleteArchives && archivePath != "" && archives != nil {
			if err := archives.Delete(archivePath); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("archive %s: %v", archivePath, err))
				continue
			}
			result.ArchivesDeletedCount++
		}
	}

	return result
}

// DeleteMod removes one mod folder and optionally its saved download archive.
func DeleteMod(
	installer *Installer,
	folderPath string,
	deleteArchive bool,
	resolve ModResolver,
	archives ArchiveIndex,
	nexusModID NexusModIDFromKeys,
) error {
	result := DeleteMods(installer, []string{folderPath}, deleteArchive, resolve, archives, nexusModID)
	if len(result.Errors) > 0 {
		return fmt.Errorf("%s", result.Errors[0])
	}
	if result.DeletedCount == 0 && folderPath != "" {
		return fmt.Errorf("mod not found: %s", folderPath)
	}
	return nil
}
