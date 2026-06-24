package mods

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"junimohut/internal/archive"
)

const (
	overwritePreviewStateBlocked = "blocked"
	overwritePreviewStateConfirm = "confirm"

	overwriteMatchThreshold = 0.5

	// Farmer20FrameworkUID is the UniqueID of the Farmer 2.0 ESWF framework mod.
	Farmer20FrameworkUID = "Salty.Farmer2.0"
)

// ArchiveHasRootManifest reports whether extractRoot contains at least one root manifest.json.
func ArchiveHasRootManifest(extractRoot string) (bool, error) {
	paths, err := findAllManifests(extractRoot)
	if err != nil {
		return false, err
	}
	return len(FilterRootManifests(paths, extractRoot)) > 0, nil
}

// ArchiveLooksLikeOverwritePatch reports whether an extracted archive has files but no root manifest.
func ArchiveLooksLikeOverwritePatch(extractRoot string) (bool, error) {
	hasManifest, err := ArchiveHasRootManifest(extractRoot)
	if err != nil {
		return false, err
	}
	if hasManifest {
		return false, nil
	}
	paths, err := collectArchiveFilePaths(extractRoot)
	if err != nil {
		return false, err
	}
	return len(paths) > 0, nil
}

// PreviewInstallOverwrites detects no-manifest archives whose paths align with installed mods.
func PreviewInstallOverwrites(archivePaths []string, modsRoot string, library []Mod) ([]InstallOverwritePreview, error) {
	var previews []InstallOverwritePreview
	for _, archivePath := range archivePaths {
		preview, ok, err := previewInstallOverwriteForArchive(archivePath, modsRoot, library)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(archivePath), err)
		}
		if ok {
			previews = append(previews, preview)
		}
	}
	return previews, nil
}

// normalizeArchivePathKey canonicalizes archive paths for map lookups across OSes.
func normalizeArchivePathKey(path string) string {
	return filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
}

// LookupOverwriteTargets resolves merge targets from the install dialog map.
func LookupOverwriteTargets(overwriteTargets map[string][]string, archivePath string) []string {
	if overwriteTargets == nil {
		return nil
	}
	if targets := overwriteTargets[archivePath]; len(targets) > 0 {
		return targets
	}
	want := normalizeArchivePathKey(archivePath)
	for key, targets := range overwriteTargets {
		if len(targets) == 0 {
			continue
		}
		if normalizeArchivePathKey(key) == want {
			return targets
		}
	}
	return nil
}

// DefaultMergeTargetsFromPreview picks merge folders when the UI pre-selects candidates.
func DefaultMergeTargetsFromPreview(preview InstallOverwritePreview) []string {
	if preview.State != overwritePreviewStateConfirm {
		return nil
	}
	candidates := preview.Candidates
	if len(candidates) == 0 {
		return nil
	}

	uniqueIDs := map[string]struct{}{}
	for _, candidate := range candidates {
		if candidate.UniqueID != "" {
			uniqueIDs[candidate.UniqueID] = struct{}{}
		}
	}
	if len(uniqueIDs) > 1 {
		targets := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			if folder := strings.TrimSpace(candidate.FolderPath); folder != "" {
				targets = append(targets, folder)
			}
		}
		return targets
	}

	if target := strings.TrimSpace(preview.SuggestedTarget); target != "" {
		return []string{target}
	}
	if folder := strings.TrimSpace(candidates[0].FolderPath); folder != "" {
		return []string{folder}
	}
	return nil
}

// ResolveInstallMergeTargets returns merge folders for a patch archive, using explicit
// UI selections when present and falling back to overwrite preview defaults.
func ResolveInstallMergeTargets(
	archivePath string,
	overwriteTargets map[string][]string,
	modsRoot string,
	library []Mod,
) ([]string, error) {
	if targets := LookupOverwriteTargets(overwriteTargets, archivePath); len(targets) > 0 {
		return targets, nil
	}
	previews, err := PreviewInstallOverwrites([]string{archivePath}, modsRoot, library)
	if err != nil {
		return nil, err
	}
	if len(previews) == 0 {
		return nil, nil
	}
	return DefaultMergeTargetsFromPreview(previews[0]), nil
}

func previewInstallOverwriteForArchive(archivePath, modsRoot string, library []Mod) (InstallOverwritePreview, bool, error) {
	tmpDir, err := os.MkdirTemp("", "sdvm-overwrite-preview-*")
	if err != nil {
		return InstallOverwritePreview{}, false, err
	}
	defer os.RemoveAll(tmpDir)

	if err := archive.Extract(archivePath, tmpDir); err != nil {
		return InstallOverwritePreview{}, false, fmt.Errorf("extract archive: %w", err)
	}

	relPaths, err := collectArchiveFilePaths(tmpDir)
	if err != nil {
		return InstallOverwritePreview{}, false, err
	}
	if len(relPaths) == 0 {
		return InstallOverwritePreview{}, false, nil
	}

	hasManifest, err := ArchiveHasRootManifest(tmpDir)
	if err != nil {
		return InstallOverwritePreview{}, false, err
	}
	if hasManifest {
		return previewManifestArchiveMerge(archivePath, modsRoot, library, relPaths, tmpDir)
	}

	return previewNoManifestArchiveMerge(archivePath, modsRoot, library, relPaths)
}

func previewManifestArchiveMerge(archivePath, modsRoot string, library []Mod, relPaths []string, extractRoot string) (InstallOverwritePreview, bool, error) {
	incoming, err := extractRootManifestObjects(extractRoot)
	if err != nil {
		return InstallOverwritePreview{}, false, err
	}
	if len(incoming) != 1 {
		return InstallOverwritePreview{}, false, nil
	}

	installed := libraryModsForManifest(library, incoming[0])
	if len(installed) == 0 {
		return InstallOverwritePreview{}, false, nil
	}

	candidates := rankOverwriteCandidates(modsRoot, installed, relPaths)
	if len(candidates) == 0 {
		candidates = installedModCandidates(installed, len(relPaths))
	}

	preview := InstallOverwritePreview{
		ArchivePath:     archivePath,
		FileCount:       len(relPaths),
		Candidates:      candidates,
		State:           overwritePreviewStateConfirm,
		SuggestedTarget: candidates[0].FolderPath,
	}
	return preview, true, nil
}

func previewNoManifestArchiveMerge(archivePath, modsRoot string, library []Mod, relPaths []string) (InstallOverwritePreview, bool, error) {
	candidates := rankOverwriteCandidates(modsRoot, library, relPaths)
	if len(candidates) == 0 {
		candidates = inferHostedPresetCandidates(modsRoot, library, relPaths)
	}
	preview := InstallOverwritePreview{
		ArchivePath: archivePath,
		FileCount:   len(relPaths),
		Candidates:  candidates,
	}

	if len(candidates) == 0 {
		preview.State = overwritePreviewStateBlocked
		preview.BlockReason = blockedOverwriteReason(relPaths, library)
		return preview, true, nil
	}

	preview.State = overwritePreviewStateConfirm
	preview.SuggestedTarget = candidates[0].FolderPath
	return preview, true, nil
}

func extractRootManifestObjects(extractRoot string) ([]Manifest, error) {
	paths, err := findAllManifests(extractRoot)
	if err != nil {
		return nil, err
	}
	paths = FilterRootManifests(paths, extractRoot)
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

func libraryModsForManifest(library []Mod, manifest Manifest) []Mod {
	var mods []Mod
	for _, mod := range library {
		if UniqueIDsEqual(mod.Manifest.UniqueID, manifest.UniqueID) {
			mods = append(mods, mod)
		}
	}
	return mods
}

func installedModCandidates(mods []Mod, totalFiles int) []InstallOverwriteCandidate {
	candidates := make([]InstallOverwriteCandidate, 0, len(mods))
	for _, mod := range mods {
		candidates = append(candidates, InstallOverwriteCandidate{
			FolderPath: mod.FolderPath,
			ModName:    mod.Manifest.Name,
			UniqueID:   mod.Manifest.UniqueID,
			TotalFiles: totalFiles,
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].FolderPath < candidates[j].FolderPath
	})
	return candidates
}

func archiveUpdateMatchesDest(extractRoot, dest string) (bool, error) {
	incoming, err := extractRootManifestObjects(extractRoot)
	if err != nil {
		return false, err
	}
	if len(incoming) != 1 {
		return false, nil
	}

	destManifests, err := findAllManifests(dest)
	if err != nil {
		return false, err
	}
	destManifests = FilterRootManifests(destManifests, dest)
	for _, path := range destManifests {
		existing, err := ParseManifest(path)
		if err != nil {
			continue
		}
		if UniqueIDsEqual(existing.UniqueID, incoming[0].UniqueID) {
			return true, nil
		}
	}
	return false, nil
}

func rankOverwriteCandidates(modsRoot string, library []Mod, relPaths []string) []InstallOverwriteCandidate {
	total := len(relPaths)
	if total == 0 {
		return nil
	}

	byFolder := map[string]InstallOverwriteCandidate{}
	for _, mod := range library {
		for _, container := range containerPathsForMod(mod.FolderPath) {
			matched, samples := scoreModPathOverlap(modsRoot, container, relPaths)
			if matched == 0 {
				continue
			}
			denom := relevantArchiveFileCount(container, relPaths)
			score := float64(matched) / float64(denom)
			if score < overwriteMatchThreshold {
				continue
			}
			existing, ok := byFolder[container]
			if ok && existing.MatchedFiles >= matched {
				continue
			}
			byFolder[container] = InstallOverwriteCandidate{
				FolderPath:   container,
				ModName:      mod.Manifest.Name,
				UniqueID:     mod.Manifest.UniqueID,
				MatchedFiles: matched,
				TotalFiles:   denom,
				SamplePaths:  samples,
			}
		}
	}

	candidates := make([]InstallOverwriteCandidate, 0, len(byFolder))
	for _, candidate := range byFolder {
		candidates = append(candidates, candidate)
	}
	candidates = dedupeOverwriteCandidates(candidates)

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].MatchedFiles != candidates[j].MatchedFiles {
			return candidates[i].MatchedFiles > candidates[j].MatchedFiles
		}
		return candidates[i].ModName < candidates[j].ModName
	})
	return candidates
}

// relevantArchiveFileCount is the number of archive paths that plausibly belong to
// container. Assorted-asset packs use one top-level folder per mod; scoring against
// the full archive size would hide every partial match.
func relevantArchiveFileCount(container string, relPaths []string) int {
	container = filepath.ToSlash(strings.TrimSpace(container))
	if container == "" {
		return len(relPaths)
	}
	count := 0
	firstSeg := container
	if idx := strings.Index(container, "/"); idx >= 0 {
		firstSeg = container[:idx]
	}
	base := filepath.Base(container)
	for _, rel := range relPaths {
		rel = filepath.ToSlash(rel)
		if rel == container || strings.HasPrefix(rel, container+"/") {
			count++
			continue
		}
		parts := strings.Split(rel, "/")
		if len(parts) == 0 {
			continue
		}
		if strings.EqualFold(parts[0], firstSeg) || strings.EqualFold(parts[0], base) {
			count++
		}
	}
	if count == 0 {
		return len(relPaths)
	}
	return count
}

func countTopLevelPrefixes(relPaths []string) int {
	seen := map[string]struct{}{}
	for _, rel := range relPaths {
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) == 0 || parts[0] == "" {
			continue
		}
		seen[strings.ToLower(parts[0])] = struct{}{}
	}
	return len(seen)
}

func shouldUseFilteredOverwriteMerge(relPaths []string, dest string) bool {
	destBase := filepath.Base(dest)
	if archiveHasUniformPrefix(relPaths, destBase) {
		return false
	}
	return countTopLevelPrefixes(relPaths) > 1
}

func overwriteMergeDestPath(modAbs, rel string) (string, bool) {
	for _, target := range overwriteTargetPaths(modAbs, rel) {
		if fi, err := os.Stat(target); err == nil && !fi.IsDir() {
			return target, true
		}
	}
	targets := overwriteTargetPaths(modAbs, rel)
	if len(targets) >= 2 {
		return targets[1], true
	}
	if len(targets) == 1 {
		return targets[0], true
	}
	return "", false
}

func copyMatchedOverwriteFiles(extractRoot, dest string, relPaths []string) error {
	modAbs := filepath.Clean(dest)
	for _, rel := range relPaths {
		destPath, ok := overwriteMergeDestPath(modAbs, rel)
		if !ok {
			continue
		}
		srcPath := filepath.Join(extractRoot, filepath.FromSlash(rel))
		data, err := os.ReadFile(srcPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func dedupeOverwriteCandidates(candidates []InstallOverwriteCandidate) []InstallOverwriteCandidate {
	if len(candidates) < 2 {
		return candidates
	}
	keep := make([]bool, len(candidates))
	for i := range keep {
		keep[i] = true
	}
	for i, parent := range candidates {
		if !keep[i] {
			continue
		}
		for j, nested := range candidates {
			if i == j || !keep[j] {
				continue
			}
			if parent.UniqueID != nested.UniqueID || parent.MatchedFiles != nested.MatchedFiles {
				continue
			}
			if isFolderPrefix(parent.FolderPath, nested.FolderPath) {
				keep[j] = false
			}
		}
	}
	out := make([]InstallOverwriteCandidate, 0, len(candidates))
	for i, candidate := range candidates {
		if keep[i] {
			out = append(out, candidate)
		}
	}
	return out
}

func isFolderPrefix(parent, child string) bool {
	parent = filepath.ToSlash(strings.TrimSuffix(strings.TrimSpace(parent), "/"))
	child = filepath.ToSlash(strings.TrimSuffix(strings.TrimSpace(child), "/"))
	if parent == "" || child == "" || parent == child {
		return false
	}
	return strings.HasPrefix(child, parent+"/")
}

func containerPathsForMod(modFolder string) []string {
	modFolder = filepath.ToSlash(strings.TrimSpace(modFolder))
	if modFolder == "" {
		return nil
	}
	parts := strings.Split(modFolder, "/")
	paths := make([]string, 0, len(parts))
	for i := 1; i <= len(parts); i++ {
		paths = append(paths, strings.Join(parts[:i], "/"))
	}
	return paths
}

func scoreModPathOverlap(modsRoot, modFolder string, relPaths []string) (matched int, samples []string) {
	modAbs := filepath.Join(modsRoot, filepath.FromSlash(modFolder))
	for _, rel := range relPaths {
		if !overwritePathMatches(modAbs, rel) {
			continue
		}
		matched++
		if len(samples) < 3 {
			samples = append(samples, filepath.ToSlash(rel))
		}
	}
	return matched, samples
}

// overwriteTargetPaths returns filesystem paths under modAbs that an archive rel-path may merge into.
// When the archive prefix repeats the mod folder name (e.g. FashionSense/Framework/... into .../FashionSense),
// the duplicate segment is stripped so paths resolve correctly.
func overwriteTargetPaths(modAbs, rel string) []string {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" {
		return nil
	}
	modAbs = filepath.Clean(modAbs)
	targets := []string{filepath.Join(modAbs, filepath.FromSlash(rel))}

	parts := strings.Split(rel, "/")
	if len(parts) > 1 {
		base := filepath.Base(modAbs)
		if strings.EqualFold(parts[0], base) {
			trimmed := strings.Join(parts[1:], "/")
			targets = append(targets, filepath.Join(modAbs, filepath.FromSlash(trimmed)))
		}
	}
	return targets
}

func overwritePathMatches(modAbs, rel string) bool {
	for _, target := range overwriteTargetPaths(modAbs, rel) {
		if fi, err := os.Stat(target); err == nil && !fi.IsDir() {
			return true
		}
	}
	return false
}

func blockedOverwriteReason(relPaths []string, library []Mod) string {
	for _, rel := range relPaths {
		lower := strings.ToLower(filepath.ToSlash(rel))
		if strings.HasPrefix(lower, "fashionsense/") {
			for _, mod := range library {
				if mod.Manifest.UniqueID == FashionSenseFrameworkUID {
					return "This archive looks like a Fashion Sense UI patch, but its files do not match your installed Fashion Sense folder layout. Check that Fashion Sense is up to date, or pick a different merge target if prompted."
				}
			}
			return "This archive looks like a Fashion Sense UI patch, but Fashion Sense is not installed. Install Fashion Sense first, then apply this patch."
		}
	}
	if isHostedPresetArchive(relPaths) {
		for _, mod := range library {
			if mod.Manifest.UniqueID == Farmer20FrameworkUID {
				return "This archive looks like a Farmer 2.0 preset pack, but it could not be matched to your installed Farmer 2.0 ESWF folder. Check that Farmer 2.0 ESWF is installed and up to date."
			}
		}
		return "This archive looks like a Farmer 2.0 preset pack, but Farmer 2.0 ESWF is not installed. Install Farmer 2.0 ESWF first, then apply this preset."
	}
	return "This archive has no manifest.json and its files do not match any installed mod folder. Install the target mod first, then try again."
}

func inferHostedPresetCandidates(modsRoot string, library []Mod, relPaths []string) []InstallOverwriteCandidate {
	if !isHostedPresetArchive(relPaths) {
		return nil
	}

	var candidates []InstallOverwriteCandidate
	for _, mod := range library {
		if mod.Manifest.UniqueID != Farmer20FrameworkUID {
			continue
		}
		container := preferredMergeContainer(mod.FolderPath)
		if !modHasAssetsDir(modsRoot, container) {
			continue
		}
		candidates = append(candidates, InstallOverwriteCandidate{
			FolderPath:   container,
			ModName:      mod.Manifest.Name,
			UniqueID:     mod.Manifest.UniqueID,
			MatchedFiles: 0,
			TotalFiles:   len(relPaths),
			SamplePaths:  sampleArchivePaths(relPaths, 3),
		})
	}
	return dedupeOverwriteCandidates(candidates)
}

func isHostedPresetArchive(relPaths []string) bool {
	top := uniformTopLevelPrefix(relPaths)
	if top == "" || !archivePathsAllUnderPrefix(relPaths, top) {
		return false
	}
	if !archiveHasPathSuffix(relPaths, "/content.json") {
		return false
	}
	for _, rel := range relPaths {
		lower := strings.ToLower(filepath.ToSlash(rel))
		if strings.Contains(lower, strings.ToLower(top+"/assets/")) {
			return true
		}
	}
	return false
}

func uniformTopLevelPrefix(relPaths []string) string {
	if len(relPaths) == 0 {
		return ""
	}
	var prefix string
	for _, rel := range relPaths {
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 2 {
			return ""
		}
		if prefix == "" {
			prefix = parts[0]
			continue
		}
		if !strings.EqualFold(prefix, parts[0]) {
			return ""
		}
	}
	return prefix
}

func archivePathsAllUnderPrefix(relPaths []string, prefix string) bool {
	prefix = filepath.ToSlash(prefix)
	for _, rel := range relPaths {
		rel = filepath.ToSlash(rel)
		if strings.EqualFold(rel, prefix) {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(rel), strings.ToLower(prefix)+"/") {
			return false
		}
	}
	return true
}

func archiveHasPathSuffix(relPaths []string, suffix string) bool {
	suffix = strings.ToLower(filepath.ToSlash(suffix))
	for _, rel := range relPaths {
		if strings.HasSuffix(strings.ToLower(filepath.ToSlash(rel)), suffix) {
			return true
		}
	}
	return false
}

func modHasAssetsDir(modsRoot, container string) bool {
	path := filepath.Join(modsRoot, filepath.FromSlash(container), "assets")
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func preferredMergeContainer(modFolder string) string {
	paths := containerPathsForMod(modFolder)
	if len(paths) == 0 {
		return modFolder
	}
	return paths[0]
}

func sampleArchivePaths(relPaths []string, limit int) []string {
	if len(relPaths) <= limit {
		out := make([]string, len(relPaths))
		copy(out, relPaths)
		return out
	}
	return append([]string(nil), relPaths[:limit]...)
}

func resolveOverwriteCopySource(extractRoot, dest string, relPaths []string) string {
	destBase := filepath.Base(dest)
	if archiveHasUniformPrefix(relPaths, destBase) {
		return filepath.Join(extractRoot, destBase)
	}
	return extractRoot
}

// hostedPresetCopySource returns the inner folder for Farmer 2.0 preset packs whose
// archive uses a wrapper folder (e.g. FernPreset/) rather than the host mod name.
func hostedPresetCopySource(extractRoot string, relPaths []string) string {
	if !isHostedPresetArchive(relPaths) {
		return ""
	}
	top := uniformTopLevelPrefix(relPaths)
	if top == "" {
		return ""
	}
	return filepath.Join(extractRoot, top)
}

func archiveHasUniformPrefix(relPaths []string, prefix string) bool {
	prefix = filepath.ToSlash(strings.TrimSpace(prefix))
	if prefix == "" {
		return false
	}
	for _, rel := range relPaths {
		rel = filepath.ToSlash(rel)
		if strings.EqualFold(rel, prefix) {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(rel), strings.ToLower(prefix)+"/") {
			return false
		}
	}
	return len(relPaths) > 0
}

func collectArchiveFilePaths(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if shouldSkipArchiveRelPath(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	return paths, err
}

func shouldSkipArchiveRelPath(rel string) bool {
	rel = filepath.ToSlash(rel)
	if strings.HasPrefix(rel, "__MACOSX") {
		return true
	}
	return strings.Contains(rel, ".DS_Store")
}

// MergeArchiveIntoMod extracts an archive and merges its files into an existing mod folder.
func (i *Installer) MergeArchiveIntoMod(archivePath, targetFolderPath string) (InstallResult, error) {
	if i.ModsRoot == "" {
		return InstallResult{}, fmt.Errorf("Mod library path not configured. Set it in Settings.")
	}
	targetFolderPath = filepath.ToSlash(strings.TrimSpace(targetFolderPath))
	if targetFolderPath == "" {
		return InstallResult{}, fmt.Errorf("merge target folder is required")
	}

	dest := filepath.Join(i.ModsRoot, filepath.FromSlash(targetFolderPath))
	if _, err := os.Stat(dest); err != nil {
		return InstallResult{}, fmt.Errorf("target mod folder not found: %s", targetFolderPath)
	}
	manifests, err := findAllManifests(dest)
	if err != nil {
		return InstallResult{}, err
	}
	if len(manifests) == 0 {
		return InstallResult{}, fmt.Errorf("target folder contains no mod files: %s", targetFolderPath)
	}

	tmpDir, err := os.MkdirTemp("", "sdvm-overwrite-merge-*")
	if err != nil {
		return InstallResult{}, err
	}
	defer os.RemoveAll(tmpDir)

	if err := archive.Extract(archivePath, tmpDir); err != nil {
		return InstallResult{}, fmt.Errorf("extract archive: %w", err)
	}

	looksLikePatch, err := ArchiveLooksLikeOverwritePatch(tmpDir)
	if err != nil {
		return InstallResult{}, err
	}
	if !looksLikePatch {
		matchesDest, err := archiveUpdateMatchesDest(tmpDir, dest)
		if err != nil {
			return InstallResult{}, err
		}
		if !matchesDest {
			return InstallResult{}, fmt.Errorf("archive is not a file patch or matching mod update")
		}
	}

	relPaths, err := collectArchiveFilePaths(tmpDir)
	if err != nil {
		return InstallResult{}, err
	}
	if shouldUseFilteredOverwriteMerge(relPaths, dest) {
		if err := copyMatchedOverwriteFiles(tmpDir, dest, relPaths); err != nil {
			return InstallResult{}, err
		}
	} else {
		srcDir := resolveOverwriteCopySource(tmpDir, dest, relPaths)
		if inner := hostedPresetCopySource(tmpDir, relPaths); inner != "" {
			srcDir = inner
		}

		if err := copyDir(srcDir, dest); err != nil {
			return InstallResult{}, err
		}
	}

	results, err := installResultsForDest(i, targetFolderPath)
	if err != nil {
		return InstallResult{}, err
	}
	if len(results) > 0 {
		return results[0], nil
	}

	rootManifests := FilterRootManifests(manifests, dest)
	if len(rootManifests) == 0 {
		rootManifests = manifests
	}
	manifest, err := ParseManifest(rootManifests[0])
	if err != nil {
		return InstallResult{}, err
	}
	modDir := filepath.Dir(rootManifests[0])
	rel, err := filepath.Rel(i.ModsRoot, modDir)
	if err != nil {
		return InstallResult{}, err
	}
	rel = filepath.ToSlash(rel)
	return InstallResult{
		FolderPath: rel,
		ModID:      ModID(rel, manifest.UniqueID),
		Name:       manifest.Name,
	}, nil
}
