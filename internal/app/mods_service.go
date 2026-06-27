package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"junimohut/internal/config"
	"junimohut/internal/moddataset"
	"junimohut/internal/mods"
	"junimohut/internal/nexus"
	"junimohut/internal/platform"
	"junimohut/internal/profiles"
)

type ModsService struct { core *Core }
func NewModsService(core *Core) *ModsService { return &ModsService{core: core} }

func (s *ModsService) ListMods(search, hideDisabled string) []mods.Mod {
	if err := s.core.RequireStarted(); err != nil {
		return nil
	}
	return s.core.Catalog.ListFiltered(search, hideDisabled)
}

func (s *ModsService) SetModEnabled(modID string, enabled bool) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	if bundle, ok := modByID(s.core, modID); ok && len(bundle.BundleChildren) > 0 {
		settings := s.core.Store.Get()
		for _, child := range bundle.BundleChildren {
			if settings.ProfileSpecificConfigs {
				if uniqueID := child.Manifest.UniqueID; uniqueID != "" {
					if enabled {
						if err := s.core.ConfigMgr.RestoreModConfig(settings.ModsRoot, child.ID, uniqueID); err != nil {
							return err
						}
					} else if err := s.core.ConfigMgr.SaveModConfig(settings.ModsRoot, child.ID, uniqueID); err != nil {
						return err
					}
				}
			}
		}
		enabledMap := s.core.Profiles.EnabledMods()
		mods.MigratePackEnableState(enabledMap, modID, enabled)
		mods.SetBundleChildrenEnabled(enabledMap, mods.BundleChildIDs(bundle), enabled)
		if err := s.core.Profiles.SaveEnabledMods(enabledMap); err != nil {
			return err
		}
		return s.core.Catalog.Refresh(s.core.Ctx())
	}

	settings := s.core.Store.Get()
	if settings.ProfileSpecificConfigs {
		if uniqueID := modUniqueIDFor(s.core, modID); uniqueID != "" {
			if enabled {
				if err := s.core.ConfigMgr.RestoreModConfig(settings.ModsRoot, modID, uniqueID); err != nil {
					return err
				}
			} else {
				if err := s.core.ConfigMgr.SaveModConfig(settings.ModsRoot, modID, uniqueID); err != nil {
					return err
				}
			}
		}
	}
	if err := s.core.Profiles.SetModEnabled(modID, enabled); err != nil {
		return err
	}
	return s.core.Catalog.Refresh(s.core.Ctx())
}

func (s *ModsService) SetModCustomName(modID, customName string) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	found := false
	s.core.Catalog.mu.RLock()
	for _, m := range s.core.Catalog.mods {
		if m.ID == modID {
			found = true
			break
		}
	}
	s.core.Catalog.mu.RUnlock()
	if !found {
		return fmt.Errorf("mod not found: %s", modID)
	}
	if err := s.core.ModNames.Set(modID, customName); err != nil {
		return err
	}
	trimmed := strings.TrimSpace(customName)
	s.core.Catalog.mu.Lock()
	for i := range s.core.Catalog.mods {
		if s.core.Catalog.mods[i].ID == modID {
			if trimmed == "" {
				s.core.Catalog.mods[i].CustomName = mods.EffectiveCustomName(
					"",
					s.core.Catalog.mods[i].FolderPath,
					s.core.Catalog.mods[i].Manifest.Name,
					mods.DisplayNameOfficial,
				)
			} else {
				s.core.Catalog.mods[i].CustomName = trimmed
			}
			break
		}
	}
	s.core.Catalog.mu.Unlock()
	s.core.Events.EmitModsChanged()
	return nil
}

func (s *ModsService) PreviewInstallNames(archivePaths []string) ([]mods.InstallNamePreview, error) {
	if err := s.core.RequireStarted(); err != nil {
		return nil, err
	}
	return mods.PreviewInstallNames(archivePaths)
}

func (s *ModsService) InstallMods(archivePaths []string, useFolderDisplayNames bool, overwriteTargets map[string][]string) ([]mods.InstallResult, error) {
	if err := s.core.RequireStarted(); err != nil {
		return nil, err
	}
	settings := s.core.Store.Get()
	installer := mods.NewInstaller(settings.ModsRoot)
	s.core.Catalog.mu.RLock()
	library := append([]mods.Mod{}, s.core.Catalog.mods...)
	s.core.Catalog.mu.RUnlock()
	var all []mods.InstallResult
	for _, p := range archivePaths {
		targets, err := mods.ResolveInstallMergeTargets(p, overwriteTargets, settings.ModsRoot, library)
		if err != nil {
			all = append(all, mods.InstallResult{Error: err.Error()})
			continue
		}
		if len(targets) > 0 {
			for _, targetFolder := range targets {
				targetFolder = strings.TrimSpace(targetFolder)
				if targetFolder == "" {
					continue
				}
				result, err := installer.MergeArchiveIntoMod(p, targetFolder)
				if err != nil {
					all = append(all, mods.InstallResult{Error: err.Error()})
					continue
				}
				all = append(all, result)
				if result.ModID != "" {
					_ = s.core.OverwriteMerges.RecordMerge(result.ModID)
					fallback := mods.ManifestModTime(filepath.Join(settings.ModsRoot, filepath.FromSlash(result.FolderPath)))
					_ = s.core.ModTimes.RecordUpdate(result.ModID, fallback)
					s.core.DownloadIndex.RecordInstall(p, modUniqueIDFor(s.core, result.ModID), 0)
				}
				if settings.AutoEnableOnInstall && result.ModID != "" {
					_ = s.core.Profiles.SetModEnabled(result.ModID, true)
				}
			}
			continue
		}

		results, err := installer.InstallArchive(p)
		if err != nil {
			all = append(all, mods.InstallResult{Error: err.Error()})
			continue
		}
		all = append(all, results...)
		for _, r := range results {
			if r.Error != "" || r.ModID == "" {
				continue
			}
			_ = s.core.ModTimes.RecordInstall(r.ModID)
			uniqueID := modUniqueIDFor(s.core, r.ModID)
			s.core.DownloadIndex.RecordInstall(p, uniqueID, 0)
		}
		if settings.AutoEnableOnInstall {
			for _, r := range results {
				if r.ModID != "" {
					_ = s.core.Profiles.SetModEnabled(r.ModID, true)
				}
			}
		}
	}
	for i := range all {
		if all[i].Error != "" {
			continue
		}
		if useFolderDisplayNames && all[i].ModID != "" {
			if label := mods.DefaultDisplayName(all[i].FolderPath, all[i].Name); label != "" {
				_ = s.core.ModNames.Set(all[i].ModID, label)
			}
		}
		all[i].Name = mods.InstallResultDisplayName(all[i].FolderPath, all[i].Name, useFolderDisplayNames)
	}
	_ = s.core.Catalog.Refresh(s.core.Ctx())
	s.core.Events.EmitModsChanged()
	return all, nil
}

func (s *ModsService) DeleteMod(folderPath string, deleteArchive bool) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	if mod, ok := modByFolderPath(s.core, folderPath); ok {
		_ = s.core.ModNames.Delete(mod.ID)
		_ = s.core.ModTimes.Delete(mod.ID)
		_ = s.core.OverwriteMerges.Delete(mod.ID)
	}
	settings := s.core.Store.Get()
	installer := mods.NewInstaller(settings.ModsRoot)
	err := mods.DeleteMod(
		installer,
		folderPath,
		deleteArchive,
		func(fp string) (mods.Mod, bool) { return modByFolderPath(s.core, fp) },
		s.core.DownloadIndex,
		nexus.ModIDFromUpdateKeys,
	)
	if err == nil {
		_ = s.core.Catalog.Refresh(s.core.Ctx())
		s.core.Events.EmitModsChanged()
	}
	return err
}

func (s *ModsService) DeleteMods(folderPaths []string, deleteArchives bool) (mods.DeleteModsResult, error) {
	if err := s.core.RequireStarted(); err != nil {
		return mods.DeleteModsResult{}, err
	}
	for _, folderPath := range folderPaths {
		if mod, ok := modByFolderPath(s.core, folderPath); ok {
			_ = s.core.ModNames.Delete(mod.ID)
			_ = s.core.ModTimes.Delete(mod.ID)
			_ = s.core.OverwriteMerges.Delete(mod.ID)
		}
	}
	settings := s.core.Store.Get()
	installer := mods.NewInstaller(settings.ModsRoot)
	result := mods.DeleteMods(
		installer,
		folderPaths,
		deleteArchives,
		func(fp string) (mods.Mod, bool) { return modByFolderPath(s.core, fp) },
		s.core.DownloadIndex,
		nexus.ModIDFromUpdateKeys,
	)
	if result.DeletedCount > 0 {
		_ = s.core.Catalog.Refresh(s.core.Ctx())
		s.core.Events.EmitModsChanged()
	}
	return result, nil
}

func (s *ModsService) UpdateMod(folderPath, archivePath string, deleteOld bool) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	settings := s.core.Store.Get()
	installer := mods.NewInstaller(settings.ModsRoot)
	s.core.Catalog.mu.RLock()
	targets := mods.ResolveUpdateFolderPaths(s.core.Catalog.mods, folderPath)
	s.core.Catalog.mu.RUnlock()
	if err := installer.UpdateMods(targets, archivePath, deleteOld); err != nil {
		return err
	}
	for _, fp := range targets {
		if mod, ok := modByFolderPath(s.core, fp); ok {
			fallback := mods.ManifestModTime(mod.AbsolutePath)
			_ = s.core.ModTimes.RecordUpdate(mod.ID, fallback)
			s.core.DownloadIndex.RecordInstall(archivePath, mod.Manifest.UniqueID, nexus.ModIDFromUpdateKeys(mod.Manifest.UpdateKeys))
		}
	}
	_ = s.core.Catalog.Refresh(s.core.Ctx())
	s.core.Catalog.mu.Lock()
	mods.ClearUpdateStatusAfterModUpdate(s.core.Catalog.mods, targets)
	s.core.Catalog.mu.Unlock()
	s.core.Catalog.SyncModUpdateCache()
	s.core.Events.EmitModsChanged()
	return nil
}

func (s *ModsService) OpenModFolder(folderPath string) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	settings := s.core.Store.Get()
	return platform.OpenPath(filepath.Join(settings.ModsRoot, filepath.FromSlash(folderPath)))
}

func (s *ModsService) OpenManifest(folderPath string) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	settings := s.core.Store.Get()
	p, err := mods.FindManifestPath(filepath.Join(settings.ModsRoot, filepath.FromSlash(folderPath)))
	if err != nil {
		return err
	}
	return platform.OpenPath(p)
}

func (s *ModsService) PreviewInstallOverwrites(archivePaths []string) ([]mods.InstallOverwritePreview, error) {
	if err := s.core.RequireStarted(); err != nil {
		return nil, err
	}
	settings := s.core.Store.Get()
	s.core.Catalog.mu.RLock()
	library := append([]mods.Mod{}, s.core.Catalog.mods...)
	s.core.Catalog.mu.RUnlock()
	return mods.PreviewInstallOverwrites(archivePaths, settings.ModsRoot, library)
}

func (s *ModsService) PreviewInstallDependencies(archivePaths []string) ([]mods.InstallDependencyPreview, error) {
	if err := s.core.RequireStarted(); err != nil {
		return nil, err
	}
	s.core.Catalog.mu.RLock()
	library := append([]mods.Mod{}, s.core.Catalog.mods...)
	s.core.Catalog.mu.RUnlock()
	return mods.PreviewInstallDependencies(archivePaths, library)
}

func (s *ModsService) ListUnmanagedMods() []profiles.UnmanagedMod {
	if err := s.core.RequireStarted(); err != nil {
		return nil
	}
	s.core.Catalog.mu.RLock()
	defer s.core.Catalog.mu.RUnlock()
	out := make([]profiles.UnmanagedMod, len(s.core.Catalog.unmanaged))
	copy(out, s.core.Catalog.unmanaged)
	return out
}

func (s *ModsService) UnmanagedModCount() int {
	if err := s.core.RequireStarted(); err != nil {
		return 0
	}
	s.core.Catalog.mu.RLock()
	defer s.core.Catalog.mu.RUnlock()
	return len(s.core.Catalog.unmanaged)
}

func (s *ModsService) ListDuplicateMods() []mods.DuplicateModGroup {
	if err := s.core.RequireStarted(); err != nil {
		return nil
	}
	s.core.Catalog.mu.RLock()
	defer s.core.Catalog.mu.RUnlock()
	out := make([]mods.DuplicateModGroup, len(s.core.Catalog.duplicates))
	copy(out, s.core.Catalog.duplicates)
	return out
}

func (s *ModsService) DuplicateModCount() int {
	if err := s.core.RequireStarted(); err != nil {
		return 0
	}
	s.core.Catalog.mu.RLock()
	defer s.core.Catalog.mu.RUnlock()
	return len(s.core.Catalog.duplicates)
}

func (s *ModsService) CleanupDuplicateModGroup(keepFolder string) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	keepFolder = filepath.ToSlash(strings.TrimSpace(keepFolder))
	if keepFolder == "" {
		return fmt.Errorf("keep folder is required")
	}

	s.core.Catalog.mu.RLock()
	group, ok := mods.DuplicateGroupForFolder(s.core.Catalog.duplicates, keepFolder)
	s.core.Catalog.mu.RUnlock()
	if !ok {
		return fmt.Errorf("no duplicate mod group found for folder %q", keepFolder)
	}
	if group.Canonical != "" {
		keepFolder = group.Canonical
	}

	settings := s.core.Store.Get()
	installer := mods.NewInstaller(settings.ModsRoot)
	for _, folder := range group.Folders {
		if folder == keepFolder {
			continue
		}
		if err := installer.DeleteMod(folder); err != nil {
			return fmt.Errorf("delete %s: %w", folder, err)
		}
	}
	return s.core.Catalog.Refresh(s.core.Ctx())
}

func (s *ModsService) OpenActiveModsFolder() error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	settings := s.core.Store.Get()
	activeModsDir := config.ActiveModsDir(settings.GamePath)
	if activeModsDir == "" {
		return fmt.Errorf("game path not configured")
	}
	return platform.OpenPath(activeModsDir)
}

func (s *ModsService) GetModDatasetPage(uniqueID string) (moddataset.ModPage, error) {
	if err := s.core.RequireStarted(); err != nil {
		return moddataset.ModPage{}, err
	}
	if s.core.ModDataset == nil {
		return moddataset.ModPage{}, fmt.Errorf("mod dataset not available")
	}
	ctx := s.core.Ctx()
	if ctx == nil {
		ctx = context.Background()
	}
	return moddataset.FetchModPageForUniqueID(ctx, s.core.ModDataset, uniqueID)
}

func (s *ModsService) ModsReadyToUpdate() int {
	if err := s.core.RequireStarted(); err != nil {
		return 0
	}
	s.core.Catalog.mu.RLock()
	defer s.core.Catalog.mu.RUnlock()
	n := 0
	for _, m := range s.core.Catalog.mods {
		if m.UpdateStatus.State == "update_available" {
			n++
		}
	}
	return n
}

func (s *ModsService) ModsWithDependencyIssues() int {
	if err := s.core.RequireStarted(); err != nil {
		return 0
	}
	s.core.Catalog.mu.RLock()
	defer s.core.Catalog.mu.RUnlock()
	n := 0
	for _, m := range s.core.Catalog.mods {
		if m.MissingDependencyCount > 0 {
			n++
		}
	}
	return n
}

func (s *ModsService) ModsIncompatible() int {
	if err := s.core.RequireStarted(); err != nil {
		return 0
	}
	s.core.Catalog.mu.RLock()
	defer s.core.Catalog.mu.RUnlock()
	n := 0
	for _, m := range s.core.Catalog.mods {
		if m.UpdateStatus.State == "incompatible" {
			n++
		}
	}
	return n
}
