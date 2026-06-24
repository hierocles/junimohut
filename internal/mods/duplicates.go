package mods

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var timestampInstallSuffixPattern = regexp.MustCompile(`_\d{8}_\d{6}$`)

// DuplicateModGroup describes multiple library folders sharing the same SMAPI UniqueID.
type DuplicateModGroup struct {
	UniqueID  string   `json:"uniqueID"`
	ModName   string   `json:"modName"`
	Folders   []string `json:"folders"`
	Canonical string   `json:"canonical"`
}

// HasTimestampInstallSuffix reports whether a folder name ends with _YYYYMMDD_HHMMSS.
func HasTimestampInstallSuffix(name string) bool {
	return timestampInstallSuffixPattern.MatchString(strings.TrimSpace(name))
}

// DetectDuplicateMods returns groups of mods that share the same UniqueID on disk.
func DetectDuplicateMods(allMods []Mod) []DuplicateModGroup {
	byUID := map[string][]Mod{}
	for _, mod := range allMods {
		uid := strings.TrimSpace(mod.Manifest.UniqueID)
		if uid == "" {
			continue
		}
		key := CanonicalUniqueID(uid)
		byUID[key] = append(byUID[key], mod)
	}

	groups := make([]DuplicateModGroup, 0)
	for _, mods := range byUID {
		if len(mods) < 2 {
			continue
		}
		folders := make([]string, 0, len(mods))
		modName := ""
		for _, mod := range mods {
			folders = append(folders, mod.FolderPath)
			if modName == "" {
				modName = mod.Manifest.Name
			}
		}
		sort.Strings(folders)
		groups = append(groups, DuplicateModGroup{
			UniqueID:  mods[0].Manifest.UniqueID,
			ModName:   modName,
			Folders:   folders,
			Canonical: pickCanonicalDuplicateFolder(folders),
		})
	}

	sort.Slice(groups, func(i, j int) bool {
		if groups[i].ModName != groups[j].ModName {
			return groups[i].ModName < groups[j].ModName
		}
		return groups[i].UniqueID < groups[j].UniqueID
	})
	return groups
}

func pickCanonicalDuplicateFolder(folders []string) string {
	var plain []string
	var stamped []string
	for _, folder := range folders {
		if HasTimestampInstallSuffix(filepath.Base(folder)) {
			stamped = append(stamped, folder)
			continue
		}
		plain = append(plain, folder)
	}
	if len(plain) == 1 {
		return plain[0]
	}
	if len(plain) > 1 {
		sort.Strings(plain)
		return plain[0]
	}
	sort.Strings(stamped)
	if len(stamped) > 0 {
		return stamped[0]
	}
	return folders[0]
}

// DuplicateGroupForFolder returns the duplicate group containing folderPath, if any.
func DuplicateGroupForFolder(groups []DuplicateModGroup, folderPath string) (DuplicateModGroup, bool) {
	folderPath = filepath.ToSlash(strings.TrimSpace(folderPath))
	for _, group := range groups {
		for _, folder := range group.Folders {
			if folder == folderPath {
				return group, true
			}
		}
	}
	return DuplicateModGroup{}, false
}
