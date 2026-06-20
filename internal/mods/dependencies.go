package mods

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"junimohut/internal/archive"
)

const (
	DependencyMissing       = "missing"
	DependencyVersionTooLow = "version_too_low"
	DependencyDisabled      = "disabled"
)

type depEntry struct {
	UniqueID       string
	MinimumVersion string
	IsRequired     bool
	IsContentPack  bool
}

// ResolveDependencies checks each mod's manifest dependencies against the installed library.
func ResolveDependencies(mods []Mod) []Mod {
	byUniqueID := modIndexByUniqueID(mods)

	out := make([]Mod, len(mods))
	for i, m := range mods {
		issues := resolveModDependencies(m, byUniqueID)
		m.DependencyIssues = issues
		m.MissingDependencyCount = len(issues)
		out[i] = m
	}
	return out
}

// ResolveManifestDependencies checks a manifest against the installed library.
func ResolveManifestDependencies(manifest Manifest, library []Mod) []DependencyIssue {
	mod := Mod{Manifest: manifest}
	return resolveModDependencies(mod, modIndexByUniqueID(library))
}

func modIndexByUniqueID(mods []Mod) map[string]Mod {
	byUniqueID := make(map[string]Mod, len(mods))
	for _, m := range mods {
		uid := m.Manifest.UniqueID
		if uid == "" {
			continue
		}
		if _, ok := byUniqueID[uid]; !ok {
			byUniqueID[uid] = m
		}
	}
	return byUniqueID
}

func resolveModDependencies(mod Mod, byUniqueID map[string]Mod) []DependencyIssue {
	entries := collectDependencyEntries(mod)
	if len(entries) == 0 {
		return nil
	}

	selfID := mod.Manifest.UniqueID
	issues := make([]DependencyIssue, 0, len(entries))
	for _, entry := range entries {
		if entry.UniqueID == "" || entry.UniqueID == selfID {
			continue
		}

		provider, ok := byUniqueID[entry.UniqueID]
		if !ok {
			if entry.IsRequired {
				issues = append(issues, newDependencyIssue(entry, DependencyMissing, nil))
			}
			continue
		}

		installedVersion := provider.Manifest.Version
		if entry.MinimumVersion != "" && !versionSatisfies(installedVersion, entry.MinimumVersion) {
			if entry.IsRequired {
				issues = append(issues, newDependencyIssue(entry, DependencyVersionTooLow, &provider))
			}
			continue
		}
		if !provider.Enabled && entry.IsRequired {
			issues = append(issues, newDependencyIssue(entry, DependencyDisabled, &provider))
		}
	}
	return issues
}

func newDependencyIssue(entry depEntry, state string, provider *Mod) DependencyIssue {
	issue := DependencyIssue{
		UniqueID:       entry.UniqueID,
		MinimumVersion: entry.MinimumVersion,
		IsRequired:     entry.IsRequired,
		IsContentPack:  entry.IsContentPack,
		State:          state,
	}
	if provider != nil {
		issue.InstalledName = provider.Manifest.Name
		issue.InstalledVersion = provider.Manifest.Version
		issue.ProviderModID = provider.ID
		issue.NexusModID = nexusModIDFromManifest(provider.Manifest)
	}
	return issue
}

func nexusModIDFromManifest(m Manifest) string {
	for _, key := range m.UpdateKeys {
		if strings.HasPrefix(key, "Nexus:") {
			return strings.TrimPrefix(key, "Nexus:")
		}
	}
	return ""
}

func collectDependencyEntries(mod Mod) []depEntry {
	seen := map[string]depEntry{}

	if cp := mod.Manifest.ContentPackFor; cp != nil && cp.UniqueID != "" {
		seen[cp.UniqueID] = depEntry{
			UniqueID:       cp.UniqueID,
			MinimumVersion: cp.MinimumVersion,
			IsRequired:     true,
			IsContentPack:  true,
		}
	}

	for _, dep := range mod.Manifest.Dependencies {
		if dep.UniqueID == "" {
			continue
		}
		required := dep.IsRequired == nil || bool(*dep.IsRequired)
		existing, ok := seen[dep.UniqueID]
		if !ok {
			seen[dep.UniqueID] = depEntry{
				UniqueID:       dep.UniqueID,
				MinimumVersion: dep.MinimumVersion,
				IsRequired:     required,
			}
			continue
		}
		if required && !existing.IsRequired {
			existing.IsRequired = true
		}
		if existing.MinimumVersion == "" && dep.MinimumVersion != "" {
			existing.MinimumVersion = dep.MinimumVersion
		}
		seen[dep.UniqueID] = existing
	}

	out := make([]depEntry, 0, len(seen))
	for _, entry := range seen {
		out = append(out, entry)
	}
	return out
}

func versionSatisfies(installedVersion, minimumVersion string) bool {
	minimumVersion = strings.TrimSpace(minimumVersion)
	if minimumVersion == "" {
		return true
	}
	installedVersion = strings.TrimSpace(installedVersion)
	if installedVersion == "" {
		return false
	}

	installed, err1 := semver.NewVersion(strings.TrimPrefix(installedVersion, "v"))
	minimum, err2 := semver.NewVersion(strings.TrimPrefix(minimumVersion, "v"))
	if err1 != nil || err2 != nil {
		return true
	}
	return !installed.LessThan(minimum)
}

// PreviewInstallDependencies extracts manifests from archives and checks dependencies.
func PreviewInstallDependencies(archivePaths []string, library []Mod) ([]InstallDependencyPreview, error) {
	library = DedupeByUniqueID(DedupeByID(library))
	var previews []InstallDependencyPreview
	for _, archivePath := range archivePaths {
		manifests, err := extractManifestsFromArchive(archivePath)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(archivePath), err)
		}
		for _, manifest := range manifests {
			issues := ResolveManifestDependencies(manifest, library)
			if len(issues) == 0 {
				continue
			}
			previews = append(previews, InstallDependencyPreview{
				ArchivePath: archivePath,
				ModName:     manifest.Name,
				UniqueID:    manifest.UniqueID,
				Issues:      issues,
			})
		}
	}
	return previews, nil
}

func extractManifestsFromArchive(archivePath string) ([]Manifest, error) {
	tmpDir, err := os.MkdirTemp("", "sdvm-preview-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	if err := archive.Extract(archivePath, tmpDir); err != nil {
		return nil, fmt.Errorf("extract archive: %w", err)
	}

	paths, err := findAllManifests(tmpDir)
	if err != nil {
		return nil, err
	}
	paths = FilterRootManifests(paths, tmpDir)
	if len(paths) == 0 {
		return nil, fmt.Errorf("no manifest.json found in archive")
	}

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
