package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"junimohut/internal/categories"
	"junimohut/internal/config"
	"junimohut/internal/modnames"
	"junimohut/internal/modoverwrites"
	"junimohut/internal/mods"
	"junimohut/internal/modtimes"
	"junimohut/internal/modupdates"
	"junimohut/internal/nexus"
	"junimohut/internal/platform"
	"junimohut/internal/profiles"
	"junimohut/internal/smapi"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// App is the main application service exposed to the frontend.
type App struct {
	ctx        context.Context
	app        *application.App
	mu         sync.RWMutex
	store      *config.Store
	profiles   *profiles.Service
	categories *categories.Service
	modNames        *modnames.Service
	modTimes        *modtimes.Service
	modUpdates      *modupdates.Service
	overwriteMerges *modoverwrites.Service
	nexus           *nexus.Client
	downloads     *nexus.DownloadManager
	downloadIndex *nexus.DownloadIndex
	scanner       *mods.Scanner
	configMgr  *profiles.ConfigManager
	modsCache  []mods.Mod
	startSMAPI bool
	initOnce   sync.Once
	initErr    error

	configEditorMu      sync.Mutex
	configEditorWindow  *application.WebviewWindow
	configEditorModID   string
	configEditorDirty   bool

	refreshMu          sync.Mutex
	assembleMu         sync.Mutex
	unmanagedModsCache []profiles.UnmanagedMod
}

func NewApp() *App {
	return &App{scanner: mods.NewScanner()}
}

func (a *App) SetApplication(app *application.App) {
	a.app = app
}

func (a *App) ensureInit() error {
	a.initOnce.Do(func() {
		ctx := a.ctx
		if ctx == nil {
			ctx = context.Background()
		}
		a.initErr = a.startup(ctx)
	})
	return a.initErr
}

func (a *App) Startup(ctx context.Context) error {
	slog.Info("App.Startup called (RPC)")
	a.ctx = ctx
	return a.ensureInit()
}

func (a *App) startup(ctx context.Context) error {
	a.ctx = ctx
	store, err := config.NewStore()
	if err != nil {
		return err
	}
	a.store = store
	_ = store.EnsureDirs()

	// Parse CLI flags
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "--start-smapi") {
			a.startSMAPI = strings.HasSuffix(arg, "true") || arg == "--start-smapi"
		}
	}

	settings := store.Get()
	prepared, err := config.EnsureModLibrary(settings, store.DataDir())
	if err != nil {
		return err
	}
	if prepared.ModsRoot != settings.ModsRoot {
		settings = prepared
		_ = store.Set(settings)
	}

	if !settings.SetupComplete {
		detected := config.DetectPaths(config.DefaultModLibrary(store.DataDir()))
		if detected.GamePath != "" {
			settings.GamePath = detected.GamePath
			settings.SMAPIPath = detected.SMAPIPath
			settings.ModsRoot = detected.ModsRoot
		}
		_ = store.Set(settings)
	}

	profSvc, err := profiles.NewService(store.ProfilesDir())
	if err != nil {
		return err
	}
	a.profiles = profSvc
	a.configMgr = profiles.NewConfigManager(store.ProfilesDir(), profSvc)

	catSvc, err := categories.NewService(store.CategoriesPath())
	if err != nil {
		return err
	}
	a.categories = catSvc

	modNamesSvc, err := modnames.NewService(store.ModNamesPath())
	if err != nil {
		return err
	}
	a.modNames = modNamesSvc

	modTimesSvc, err := modtimes.NewService(store.ModTimesPath())
	if err != nil {
		return err
	}
	a.modTimes = modTimesSvc

	modUpdatesSvc, err := modupdates.NewService(store.ModUpdatesPath())
	if err != nil {
		return err
	}
	a.modUpdates = modUpdatesSvc

	overwriteSvc, err := modoverwrites.NewService(store.OverwriteMergesPath())
	if err != nil {
		return err
	}
	a.overwriteMerges = overwriteSvc

	a.nexus = nexus.NewClient()
	downloadIndex, err := nexus.NewDownloadIndex(store.DataDir(), store.DownloadsDir())
	if err != nil {
		return err
	}
	downloadIndex.SetArchiveEnricher(enrichDownloadRecordFromArchive)
	a.downloadIndex = downloadIndex
	a.downloads = nexus.NewDownloadManager(store.DownloadsDir(), downloadIndex)
	go downloadIndex.ReconcileAsync()

	_ = a.refreshMods()
	_, _ = mods.NewWatcher(store.Get().ModsRoot, func() {
		_ = a.refreshMods()
		a.emitModsChanged()
	})

	if a.startSMAPI {
		go func() {
			time.Sleep(2 * time.Second)
			_ = a.LaunchSMAPI()
		}()
	}
	return nil
}

func (a *App) emitModsChanged() {
	if a.app != nil {
		a.app.Event.Emit("mods-changed", true)
	}
}

// EmitNXMURL notifies the frontend to handle an nxm:// link.
func (a *App) EmitNXMURL(url string) {
	if a.app == nil || !strings.HasPrefix(url, "nxm://") {
		return
	}
	a.app.Event.Emit("nxm-url", url)
}

// ProcessCommandLineArgs handles nxm:// URLs passed on startup.
func (a *App) ProcessCommandLineArgs(args []string) {
	for _, arg := range args {
		if strings.HasPrefix(arg, "nxm://") {
			a.EmitNXMURL(arg)
			return
		}
	}
}

func (a *App) refreshCategoryIDs() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.modsCache {
		a.modsCache[i].CategoryIDs = a.categories.ModCategoryIDs(a.modsCache[i].ID)
	}
}

func (a *App) refreshMods() error {
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()

	settings := a.store.Get()
	enabled := a.profiles.EnabledMods()
	list, err := a.scanner.Scan(mods.ScanOptions{
		ModsRoot:            settings.ModsRoot,
		IgnoreHiddenFolders: settings.IgnoreHiddenFolders,
		EnabledMods:         enabled,
		Grouping:            settings.ModGrouping,
		SkipPackCollapse:    true,
	})
	if err != nil {
		return err
	}
	for i := range list {
		list[i].CategoryIDs = a.categories.ModCategoryIDs(list[i].ID)
		list[i].CustomName = mods.EffectiveCustomName(
			a.modNames.Get(list[i].ID),
			list[i].FolderPath,
			list[i].Manifest.Name,
			mods.DisplayNameOfficial,
		)
		list[i].ContainsOverwrites = a.overwriteMerges.ContainsOverwrites(list[i].ID)
	}
	a.enrichModTimes(list)
	list = mods.CollapseSiblingPacks(list, settings.ModsRoot, enabled)
	list = mods.DedupeByUniqueID(mods.DedupeByID(list))
	list = mods.ResolveDependencies(list)
	a.mu.Lock()
	previous := append([]mods.Mod{}, a.modsCache...)
	a.mu.Unlock()
	mods.PreserveUpdateStatus(list, previous)
	mods.ApplyCachedUpdateStatus(list, cachedUpdatesFromService(a.modUpdates))
	mods.ApplyIgnoredUpdates(list, settings.IgnoredModUpdates)
	a.mu.Lock()
	a.modsCache = list
	a.mu.Unlock()

	go a.finishModRefresh(list, enabled, settings)
	return nil
}

func cachedUpdatesFromService(svc *modupdates.Service) map[string]mods.CachedUpdate {
	if svc == nil {
		return nil
	}
	raw := svc.All()
	out := make(map[string]mods.CachedUpdate, len(raw))
	for id, e := range raw {
		out[id] = mods.CachedUpdate{
			ManifestVersion: e.ManifestVersion,
			State:           e.State,
			LatestVersion:   e.LatestVersion,
			ModPageURL:      e.ModPageURL,
			Message:         e.Message,
		}
	}
	return out
}

func (a *App) syncModUpdateCache() {
	if a.modUpdates == nil {
		return
	}
	a.mu.RLock()
	list := append([]mods.Mod{}, a.modsCache...)
	a.mu.RUnlock()
	_ = a.modUpdates.SyncFromMods(list)
}

func (a *App) enrichModTimes(list []mods.Mod) {
	seeds := map[string]int64{}
	for i := range list {
		if rec, ok := a.modTimes.Get(list[i].ID); ok {
			list[i].InstallTime = rec.InstallTime
			list[i].LastUpdated = rec.LastUpdated
			continue
		}
		t := mods.ManifestModTime(list[i].AbsolutePath)
		if t > 0 {
			seeds[list[i].ID] = t
			list[i].InstallTime = t
			list[i].LastUpdated = t
		}
	}
	_ = a.modTimes.SeedBatch(seeds)
}

func (a *App) finishModRefresh(list []mods.Mod, enabled map[string]bool, settings config.Settings) {
	a.assembleMu.Lock()
	defer a.assembleMu.Unlock()

	for i := range list {
		if list[i].IsCoreMod {
			continue
		}
		nexusModID := nexus.ModIDFromUpdateKeys(list[i].Manifest.UpdateKeys)
		if path, ok := a.downloadIndex.FindForMod(list[i].Manifest.UniqueID, nexusModID); ok {
			list[i].SavedDownloadPath = path
		}
	}
	a.mu.Lock()
	a.modsCache = list
	a.mu.Unlock()

	activeModsDir := config.ActiveModsDir(settings.GamePath)
	assembler := profiles.NewAssembler(activeModsDir, settings.ModsRoot)
	if err := assembler.Assemble(list, enabled); err != nil {
		return
	}
	unmanaged, err := profiles.ScanUnmanagedMods(activeModsDir, settings.ModsRoot)
	if err != nil {
		return
	}
	a.mu.Lock()
	a.unmanagedModsCache = unmanaged
	a.mu.Unlock()
}

// --- Settings ---

func (a *App) GetSettings() config.Settings {
	if err := a.ensureInit(); err != nil {
		return config.DefaultSettings()
	}
	return a.store.Get()
}

func (a *App) SaveSettings(settings config.Settings) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	prepared, err := config.EnsureModLibrary(settings, a.store.DataDir())
	if err != nil {
		return err
	}
	if err := a.store.Set(prepared); err != nil {
		return err
	}
	return a.refreshMods()
}

func (a *App) DetectPaths() config.DetectionResult {
	if err := a.ensureInit(); err != nil {
		return config.DetectPaths("")
	}
	return config.DetectPaths(config.DefaultModLibrary(a.store.DataDir()))
}

func (a *App) CompleteSetup(gamePath, smapiPath, modsRoot string) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	if err := a.store.Update(func(s *config.Settings) {
		s.GamePath = gamePath
		s.SMAPIPath = smapiPath
		s.ModsRoot = modsRoot
		s.SetupComplete = true
	}); err != nil {
		return err
	}
	settings := a.store.Get()
	prepared, err := config.EnsureModLibrary(settings, a.store.DataDir())
	if err != nil {
		return err
	}
	if prepared.ModsRoot != settings.ModsRoot {
		if err := a.store.Set(prepared); err != nil {
			return err
		}
	}
	return a.refreshMods()
}

// --- Mods ---

func (a *App) ListMods(search, hideDisabled string) []mods.Mod {
	if err := a.ensureInit(); err != nil {
		return nil
	}
	a.mu.RLock()
	list := append([]mods.Mod{}, a.modsCache...)
	a.mu.RUnlock()
	return mods.FilterMods(list, search, hideDisabled)
}

func (a *App) ListModGroups(search, hideDisabled string) []mods.ModGroup {
	return mods.GroupMods(a.ListMods(search, hideDisabled))
}

func (a *App) SetModEnabled(modID string, enabled bool) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	settings := a.store.Get()
	if settings.ProfileSpecificConfigs {
		if uniqueID := a.modUniqueIDFor(modID); uniqueID != "" {
			if enabled {
				if err := a.configMgr.RestoreModConfig(settings.ModsRoot, modID, uniqueID); err != nil {
					return err
				}
			} else {
				if err := a.configMgr.SaveModConfig(settings.ModsRoot, modID, uniqueID); err != nil {
					return err
				}
			}
		}
	}
	if err := a.profiles.SetModEnabled(modID, enabled); err != nil {
		return err
	}
	return a.refreshMods()
}

func (a *App) SetModCustomName(modID, customName string) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	found := false
	a.mu.RLock()
	for _, m := range a.modsCache {
		if m.ID == modID {
			found = true
			break
		}
	}
	a.mu.RUnlock()
	if !found {
		return fmt.Errorf("mod not found: %s", modID)
	}
	if err := a.modNames.Set(modID, customName); err != nil {
		return err
	}
	trimmed := strings.TrimSpace(customName)
	a.mu.Lock()
	for i := range a.modsCache {
		if a.modsCache[i].ID == modID {
			if trimmed == "" {
				a.modsCache[i].CustomName = mods.EffectiveCustomName(
					"",
					a.modsCache[i].FolderPath,
					a.modsCache[i].Manifest.Name,
					mods.DisplayNameOfficial,
				)
			} else {
				a.modsCache[i].CustomName = trimmed
			}
			break
		}
	}
	a.mu.Unlock()
	a.emitModsChanged()
	return nil
}

func (a *App) PreviewInstallNames(archivePaths []string) ([]mods.InstallNamePreview, error) {
	if err := a.ensureInit(); err != nil {
		return nil, err
	}
	return mods.PreviewInstallNames(archivePaths)
}

func (a *App) InstallMods(archivePaths []string, useFolderDisplayNames bool, overwriteTargets map[string]string) ([]mods.InstallResult, error) {
	if err := a.ensureInit(); err != nil {
		return nil, err
	}
	settings := a.store.Get()
	installer := mods.NewInstaller(settings.ModsRoot)
	var all []mods.InstallResult
	for _, p := range archivePaths {
		targetFolder := ""
		if overwriteTargets != nil {
			targetFolder = overwriteTargets[p]
		}
		if targetFolder != "" {
			result, err := installer.MergeArchiveIntoMod(p, targetFolder)
			if err != nil {
				all = append(all, mods.InstallResult{Error: err.Error()})
				continue
			}
			all = append(all, result)
			if result.ModID != "" {
				_ = a.overwriteMerges.RecordMerge(result.ModID)
				fallback := mods.ManifestModTime(filepath.Join(settings.ModsRoot, filepath.FromSlash(result.FolderPath)))
				_ = a.modTimes.RecordUpdate(result.ModID, fallback)
				a.downloadIndex.RecordInstall(p, a.modUniqueIDFor(result.ModID), 0)
			}
			if settings.AutoEnableOnInstall && result.ModID != "" {
				_ = a.profiles.SetModEnabled(result.ModID, true)
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
			_ = a.modTimes.RecordInstall(r.ModID)
			uniqueID := a.modUniqueIDFor(r.ModID)
			a.downloadIndex.RecordInstall(p, uniqueID, 0)
		}
		if settings.AutoEnableOnInstall {
			for _, r := range results {
				if r.ModID != "" {
					_ = a.profiles.SetModEnabled(r.ModID, true)
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
				_ = a.modNames.Set(all[i].ModID, label)
			}
		}
		all[i].Name = mods.InstallResultDisplayName(all[i].FolderPath, all[i].Name, useFolderDisplayNames)
	}
	_ = a.refreshMods()
	a.emitModsChanged()
	return all, nil
}

func (a *App) DeleteMod(folderPath string, deleteArchive bool) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	if mod, ok := a.modByFolderPath(folderPath); ok {
		_ = a.modNames.Delete(mod.ID)
		_ = a.modTimes.Delete(mod.ID)
		_ = a.overwriteMerges.Delete(mod.ID)
	}
	settings := a.store.Get()
	installer := mods.NewInstaller(settings.ModsRoot)
	return mods.DeleteMod(
		installer,
		folderPath,
		deleteArchive,
		a.modByFolderPath,
		a.downloadIndex,
		nexus.ModIDFromUpdateKeys,
	)
}

func (a *App) DeleteMods(folderPaths []string, deleteArchives bool) (mods.DeleteModsResult, error) {
	if err := a.ensureInit(); err != nil {
		return mods.DeleteModsResult{}, err
	}
	for _, folderPath := range folderPaths {
		if mod, ok := a.modByFolderPath(folderPath); ok {
			_ = a.modNames.Delete(mod.ID)
			_ = a.modTimes.Delete(mod.ID)
			_ = a.overwriteMerges.Delete(mod.ID)
		}
	}
	settings := a.store.Get()
	installer := mods.NewInstaller(settings.ModsRoot)
	result := mods.DeleteMods(
		installer,
		folderPaths,
		deleteArchives,
		a.modByFolderPath,
		a.downloadIndex,
		nexus.ModIDFromUpdateKeys,
	)
	if result.DeletedCount > 0 {
		_ = a.refreshMods()
		a.emitModsChanged()
	}
	return result, nil
}

func (a *App) UpdateMod(folderPath, archivePath string, deleteOld bool) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	settings := a.store.Get()
	installer := mods.NewInstaller(settings.ModsRoot)
	a.mu.RLock()
	targets := mods.NexusSiblingFolderPaths(a.modsCache, folderPath)
	a.mu.RUnlock()
	if err := installer.UpdateMods(targets, archivePath, deleteOld); err != nil {
		return err
	}
	for _, fp := range targets {
		if mod, ok := a.modByFolderPath(fp); ok {
			fallback := mods.ManifestModTime(mod.AbsolutePath)
			_ = a.modTimes.RecordUpdate(mod.ID, fallback)
			a.downloadIndex.RecordInstall(archivePath, mod.Manifest.UniqueID, nexus.ModIDFromUpdateKeys(mod.Manifest.UpdateKeys))
		}
	}
	_ = a.refreshMods()
	a.emitModsChanged()
	return nil
}

func (a *App) OpenModFolder(folderPath string) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	settings := a.store.Get()
	return platform.OpenPath(filepath.Join(settings.ModsRoot, filepath.FromSlash(folderPath)))
}

func (a *App) OpenManifest(folderPath string) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	settings := a.store.Get()
	p, err := mods.FindManifestPath(filepath.Join(settings.ModsRoot, filepath.FromSlash(folderPath)))
	if err != nil {
		return err
	}
	return platform.OpenPath(p)
}

// --- Profiles ---

func (a *App) ListProfiles() []profiles.Profile {
	if err := a.ensureInit(); err != nil {
		return nil
	}
	return a.profiles.List()
}

func (a *App) CreateProfile(name string) (profiles.Profile, error) {
	if err := a.ensureInit(); err != nil {
		return profiles.Profile{}, err
	}
	return a.profiles.Create(name)
}

func (a *App) DeleteProfile(id string) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	return a.profiles.Delete(id)
}

func (a *App) RenameProfile(id, name string) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	return a.profiles.Rename(id, name)
}

func (a *App) SetActiveProfile(id string) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	settings := a.store.Get()
	if settings.ProfileSpecificConfigs {
		if err := a.configMgr.SaveConfigs(settings.ModsRoot, a.modUniqueIDMapFromEnabled(a.profiles.EnabledMods())); err != nil {
			return err
		}
	}
	if err := a.profiles.SetActive(id); err != nil {
		return err
	}
	if settings.ProfileSpecificConfigs {
		if err := a.configMgr.RestoreConfigs(settings.ModsRoot, a.modUniqueIDMapFromEnabled(a.profiles.EnabledMods())); err != nil {
			return err
		}
	}
	return a.refreshMods()
}

func (a *App) SaveProfile() error {
	return nil
}

func (a *App) modUniqueIDMap() map[string]string {
	return a.modUniqueIDMapFromEnabled(a.profiles.EnabledMods())
}

func (a *App) modUniqueIDMapFromEnabled(enabled map[string]bool) map[string]string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	m := map[string]string{}
	for _, mod := range a.modsCache {
		en := true
		if enabled != nil {
			if v, ok := enabled[mod.ID]; ok {
				en = v
			}
		}
		if mods.CoreModIDs[mod.Manifest.UniqueID] {
			en = true
		}
		if !en {
			continue
		}
		m[mod.ID] = mod.Manifest.UniqueID
	}
	return m
}

func (a *App) modUniqueIDFor(modID string) string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, mod := range a.modsCache {
		if mod.ID == modID {
			return mod.Manifest.UniqueID
		}
	}
	return ""
}

func (a *App) modByFolderPath(folderPath string) (mods.Mod, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, mod := range a.modsCache {
		if mod.FolderPath == folderPath {
			return mod, true
		}
	}
	return mods.Mod{}, false
}

// --- Categories ---

func (a *App) ListCategories() []categories.Category {
	if err := a.ensureInit(); err != nil {
		return nil
	}
	return a.categories.List()
}

func (a *App) CreateCategory(name, color string) (categories.Category, error) {
	if err := a.ensureInit(); err != nil {
		return categories.Category{}, err
	}
	return a.categories.Create(name, color)
}

func (a *App) UpdateCategory(id, name, color string, visible bool, sortOrder int) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	return a.categories.Update(id, name, color, visible, sortOrder)
}

func (a *App) DeleteCategory(id string) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	return a.categories.Delete(id)
}

func (a *App) SetCategoryVisibility(id string, visible bool) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	return a.categories.SetVisibility(id, visible)
}

func (a *App) AssignModToCategory(categoryID, modID string) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	if err := a.categories.AssignMod(categoryID, modID); err != nil {
		return err
	}
	a.refreshCategoryIDs()
	a.emitModsChanged()
	return nil
}

func (a *App) UnassignModFromCategory(categoryID, modID string) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	if err := a.categories.UnassignMod(categoryID, modID); err != nil {
		return err
	}
	a.refreshCategoryIDs()
	a.emitModsChanged()
	return nil
}

func (a *App) ReorderCategories(ids []string) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	return a.categories.Reorder(ids)
}

// --- SMAPI ---

func (a *App) LaunchSMAPI() error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	settings := a.store.Get()
	if err := a.refreshMods(); err != nil {
		return err
	}
	if settings.ProfileSpecificConfigs {
		if err := a.configMgr.SaveConfigs(settings.ModsRoot, a.modUniqueIDMapFromEnabled(a.profiles.EnabledMods())); err != nil {
			return err
		}
	}
	launcher := smapi.NewLauncher(settings.GamePath, settings.SMAPIPath)
	return launcher.Launch()
}

func (a *App) GetSMAPIVersion() string {
	if err := a.ensureInit(); err != nil {
		return ""
	}
	settings := a.store.Get()
	return smapi.NewLauncher(settings.GamePath, settings.SMAPIPath).Version()
}

func (a *App) CheckSMAPIUpdate() (smapi.UpdateInfo, error) {
	if err := a.ensureInit(); err != nil {
		return smapi.UpdateInfo{}, err
	}
	return smapi.CheckSMAPIUpdate(a.GetSMAPIVersion())
}

func (a *App) InstallSMAPI() error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	info, _ := smapi.CheckSMAPIUpdate(a.GetSMAPIVersion())
	return smapi.InstallSMAPI(info.DownloadURL, a.store.Get().GamePath)
}

// --- Updates ---

func (a *App) CheckModUpdates() ([]smapi.ModUpdateResult, error) {
	if err := a.ensureInit(); err != nil {
		return nil, err
	}
	settings := a.store.Get()
	now := time.Now()
	if smapi.UpdateCheckRateLimited(settings.LastUpdateCheck, now) {
		retryAfter := smapi.UpdateCheckRetryAfter(settings.LastUpdateCheck, now)
		return nil, fmt.Errorf("%w — try again in %s", smapi.ErrUpdateCheckRateLimited, smapi.FormatUpdateCheckRetryAfter(retryAfter))
	}
	a.mu.RLock()
	list := a.modsCache
	a.mu.RUnlock()

	var requests []smapi.ModUpdateRequest
	seenNexus := map[int]bool{}
	for _, m := range list {
		if len(m.Manifest.UpdateKeys) == 0 {
			continue
		}
		nexusID := nexus.ModIDFromUpdateKeys(m.Manifest.UpdateKeys)
		if nexusID > 0 {
			if seenNexus[nexusID] {
				continue
			}
			seenNexus[nexusID] = true
			rep, ok := mods.PickNexusGroupRepresentative(list, nexusID)
			if !ok {
				rep = m
			}
			requests = append(requests, smapi.ModUpdateRequest{
				UniqueID:   rep.Manifest.UniqueID,
				Version:    rep.Manifest.Version,
				UpdateKeys: rep.Manifest.UpdateKeys,
			})
			continue
		}
		requests = append(requests, smapi.ModUpdateRequest{
			UniqueID:   m.Manifest.UniqueID,
			Version:    m.Manifest.Version,
			UpdateKeys: m.Manifest.UpdateKeys,
		})
	}
	results, err := smapi.CheckModUpdates(requests, a.GetSMAPIVersion())
	if err != nil {
		return nil, err
	}
	_ = a.store.Update(func(s *config.Settings) {
		s.LastUpdateCheck = time.Now().Unix()
	})
	resultMap := map[string]smapi.ModUpdateResult{}
	for _, r := range results {
		resultMap[r.UniqueID] = r
	}
	nexusResults := map[int]smapi.ModUpdateResult{}
	for _, req := range requests {
		if r, ok := resultMap[req.UniqueID]; ok {
			if id := nexus.ModIDFromUpdateKeys(req.UpdateKeys); id > 0 {
				nexusResults[id] = r
			}
		}
	}
	a.mu.Lock()
	for i, m := range a.modsCache {
		var r smapi.ModUpdateResult
		var ok bool
		if id := nexus.ModIDFromUpdateKeys(m.Manifest.UpdateKeys); id > 0 {
			r, ok = nexusResults[id]
		} else {
			r, ok = resultMap[m.Manifest.UniqueID]
		}
		if !ok {
			continue
		}
		a.modsCache[i].UpdateStatus = mods.UpdateStatus{
			State:         mods.NormalizeUpdateState(r.Status),
			LatestVersion: r.LatestVersion,
			ModPageURL:    r.ModPageURL,
			Message:       r.Message,
		}
	}
	mods.PropagateNexusUpdateStatus(a.modsCache)
	mods.ApplyIgnoredUpdates(a.modsCache, settings.IgnoredModUpdates)
	a.mu.Unlock()
	a.syncModUpdateCache()
	return results, nil
}

func (a *App) SetModUpdateIgnored(modID string, ignored bool) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	a.mu.RLock()
	var target mods.Mod
	found := false
	for _, m := range a.modsCache {
		if m.ID == modID {
			target = m
			found = true
			break
		}
	}
	a.mu.RUnlock()
	if !found {
		return fmt.Errorf("mod not found")
	}
	nexusID := nexus.ModIDFromUpdateKeys(target.Manifest.UpdateKeys)
	if nexusID == 0 {
		return fmt.Errorf("mod has no Nexus update key")
	}
	latest := strings.TrimSpace(target.UpdateStatus.LatestVersion)
	if ignored && latest == "" {
		return fmt.Errorf("no available update to ignore")
	}
	key := strconv.Itoa(nexusID)
	if err := a.store.Update(func(s *config.Settings) {
		if s.IgnoredModUpdates == nil {
			s.IgnoredModUpdates = map[string]string{}
		}
		if ignored {
			s.IgnoredModUpdates[key] = latest
		} else {
			delete(s.IgnoredModUpdates, key)
		}
	}); err != nil {
		return err
	}
	a.mu.Lock()
	mods.ApplyIgnoredUpdates(a.modsCache, a.store.Get().IgnoredModUpdates)
	a.mu.Unlock()
	a.syncModUpdateCache()
	return nil
}

func (a *App) ModsReadyToUpdate() int {
	if err := a.ensureInit(); err != nil {
		return 0
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	n := 0
	for _, m := range a.modsCache {
		if m.UpdateStatus.State == "update_available" {
			n++
		}
	}
	return n
}

func (a *App) ModsWithDependencyIssues() int {
	if err := a.ensureInit(); err != nil {
		return 0
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	n := 0
	for _, m := range a.modsCache {
		if m.MissingDependencyCount > 0 {
			n++
		}
	}
	return n
}

func (a *App) ListUnmanagedMods() []profiles.UnmanagedMod {
	if err := a.ensureInit(); err != nil {
		return nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]profiles.UnmanagedMod, len(a.unmanagedModsCache))
	copy(out, a.unmanagedModsCache)
	return out
}

func (a *App) UnmanagedModCount() int {
	if err := a.ensureInit(); err != nil {
		return 0
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.unmanagedModsCache)
}

func (a *App) OpenActiveModsFolder() error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	settings := a.store.Get()
	activeModsDir := config.ActiveModsDir(settings.GamePath)
	if activeModsDir == "" {
		return fmt.Errorf("game path not configured")
	}
	return platform.OpenPath(activeModsDir)
}

func (a *App) PreviewInstallOverwrites(archivePaths []string) ([]mods.InstallOverwritePreview, error) {
	if err := a.ensureInit(); err != nil {
		return nil, err
	}
	settings := a.store.Get()
	a.mu.RLock()
	library := append([]mods.Mod{}, a.modsCache...)
	a.mu.RUnlock()
	return mods.PreviewInstallOverwrites(archivePaths, settings.ModsRoot, library)
}

func (a *App) PreviewInstallDependencies(archivePaths []string) ([]mods.InstallDependencyPreview, error) {
	if err := a.ensureInit(); err != nil {
		return nil, err
	}
	a.mu.RLock()
	library := append([]mods.Mod{}, a.modsCache...)
	a.mu.RUnlock()
	return mods.PreviewInstallDependencies(archivePaths, library)
}

// --- Nexus ---

func (a *App) SetNexusAPIKey(key string) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	return a.nexus.SetAPIKey(key)
}

func (a *App) ValidateNexusAPIKey() (bool, error) {
	if err := a.ensureInit(); err != nil {
		return false, err
	}
	return a.nexus.ValidateKey()
}

func (a *App) ProbeNexusAPIKey() bool {
	if err := a.ensureInit(); err != nil {
		return false
	}
	if !a.nexus.IsConnected() {
		return false
	}
	ok, err := a.nexus.ValidateKey()
	if err != nil && nexus.IsTransientNetworkError(err) {
		return true
	}
	return ok && err == nil
}

func (a *App) IsNexusConnected() bool {
	if err := a.ensureInit(); err != nil {
		return false
	}
	return a.nexus.IsConnected()
}

// GetInstallSuggestedTags maps install archives and Nexus mod categories to user tag IDs.
func (a *App) GetInstallSuggestedTags(archivePaths []string, modIDs []int) ([]string, error) {
	if err := a.ensureInit(); err != nil {
		return nil, err
	}

	knownTags := map[string]bool{}
	for _, c := range a.categories.List() {
		knownTags[c.ID] = true
	}

	fashionSense, err := mods.ArchivesContainFashionSense(archivePaths)
	if err != nil {
		return nil, err
	}

	var nexusTagIDs []string
	hasArchives := len(archivePaths) > 0
	if a.nexus.IsConnected() {
		seen := map[string]bool{}
		for _, modID := range modIDs {
			if modID <= 0 {
				continue
			}
			name, err := a.nexus.CategoryNameForMod(modID)
			if err != nil || name == "" {
				continue
			}
			tagID := categories.TagIDForNexusCategory(name)
			if tagID == "" || seen[tagID] || !knownTags[tagID] {
				continue
			}
			if !hasArchives && categories.NexusCategoryDefersUntilManifest(name) {
				continue
			}
			seen[tagID] = true
			nexusTagIDs = append(nexusTagIDs, tagID)
		}
	}

	return categories.MergeInstallSuggestedTags(nexusTagIDs, fashionSense, knownTags), nil
}

// GetNexusSuggestedTags maps Nexus mod page categories to existing user tag IDs.
func (a *App) GetNexusSuggestedTags(modIDs []int) ([]string, error) {
	return a.GetInstallSuggestedTags(nil, modIDs)
}

func (a *App) EndorseMod(updateKey, version string) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	id, ok := nexus.ExtractNexusID(updateKey)
	if !ok {
		return fmt.Errorf("Mod has no Nexus update key")
	}
	return a.nexus.EndorseMod(id, version)
}

func (a *App) ListDownloads() []nexus.DownloadEntry {
	if err := a.ensureInit(); err != nil {
		return nil
	}
	return a.downloads.List()
}

func (a *App) ListSavedDownloads() []nexus.DownloadRecord {
	if err := a.ensureInit(); err != nil {
		return nil
	}
	a.downloadIndex.Reconcile()
	return a.downloadIndex.List()
}

func (a *App) DeleteSavedDownload(archivePath string) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	return a.downloadIndex.Delete(archivePath)
}

func (a *App) RevealArchiveInFileManager(archivePath string) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	return platform.RevealInFileManager(archivePath)
}

func (a *App) emitDownloadReady(path string) {
	if a.app == nil || path == "" {
		return
	}
	a.app.Event.Emit("nexus-download-ready", path)
}

func (a *App) downloadNexusFile(modID, fileID int, modName string, auth *nexus.DownloadAuth) (string, error) {
	path, err := a.downloads.DownloadFile(a.nexus, modID, fileID, modName, auth)
	if err != nil {
		return "", err
	}
	a.emitDownloadReady(path)
	return path, nil
}

func (a *App) DownloadModUpdate(updateKey string, modName string) (string, error) {
	if err := a.ensureInit(); err != nil {
		return "", err
	}
	id, ok := nexus.ExtractNexusID(updateKey)
	if !ok {
		return "", fmt.Errorf("Not a Nexus mod")
	}
	return a.downloadNexusFile(id, 0, modName, nil)
}

func (a *App) HandleNXMURL(url string) (string, error) {
	if err := a.ensureInit(); err != nil {
		return "", err
	}
	parsed, err := nexus.ParseNXMURL(url)
	if err != nil {
		return "", err
	}
	return a.downloadNexusFile(parsed.ModID, parsed.FileID, fmt.Sprintf("mod_%d", parsed.ModID), parsed.Auth)
}

func enrichDownloadRecordFromArchive(rec *nexus.DownloadRecord) {
	if rec == nil || rec.ArchivePath == "" {
		return
	}
	manifests, err := mods.ManifestsFromArchive(rec.ArchivePath)
	if err != nil || len(manifests) == 0 {
		return
	}
	manifest := manifests[0]
	if rec.UniqueID == "" && manifest.UniqueID != "" {
		rec.UniqueID = manifest.UniqueID
	}
	if rec.ModName == "" && manifest.Name != "" {
		rec.ModName = manifest.Name
	}
	if rec.NexusModID == 0 {
		if id := nexus.ModIDFromUpdateKeys(manifest.UpdateKeys); id > 0 {
			rec.NexusModID = id
		}
	}
}

// --- File dialogs ---

func (a *App) SelectArchives() ([]string, error) {
	if err := a.ensureInit(); err != nil {
		return nil, err
	}
	if a.app == nil {
		return nil, fmt.Errorf("App not ready")
	}
	paths, err := a.app.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		Title:                   "Select mod archives",
		AllowsMultipleSelection: true,
		CanChooseFiles:          true,
		CanChooseDirectories:    false,
		Filters: []application.FileFilter{
			{DisplayName: "Mod archives", Pattern: "*.zip;*.7z;*.rar"},
		},
	}).PromptForMultipleSelection()
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return []string{}, nil
	}
	return paths, nil
}

func (a *App) BrowseGameFolder() (string, error) {
	return a.browseFolder("Select Stardew Valley game folder")
}

func (a *App) BrowseModsRoot() (string, error) {
	return a.browseFolder("Select mod library folder")
}

func (a *App) BrowseSMAPIPath() (string, error) {
	if err := a.ensureInit(); err != nil {
		return "", err
	}
	if a.app == nil {
		return "", fmt.Errorf("App not ready")
	}
	opts := &application.OpenFileDialogOptions{
		Title:                "Select SMAPI launcher",
		CanChooseFiles:       true,
		CanChooseDirectories: false,
	}
	if runtime.GOOS == "windows" {
		opts.Filters = []application.FileFilter{
			{DisplayName: "SMAPI launcher", Pattern: "StardewModdingAPI.exe;*.exe"},
		}
	}
	path, err := a.app.Dialog.OpenFileWithOptions(opts).PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) browseFolder(title string) (string, error) {
	if err := a.ensureInit(); err != nil {
		return "", err
	}
	if a.app == nil {
		return "", fmt.Errorf("App not ready")
	}
	path, err := a.app.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		Title:                title,
		CanChooseFiles:       false,
		CanChooseDirectories: true,
	}).PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	return path, nil
}

// --- i18n ---

func (a *App) GetTranslations(locale string) map[string]string {
	return defaultTranslations(locale)
}

func defaultTranslations(locale string) map[string]string {
	// English defaults; extend with locale files in Phase 5
	_ = locale
	return map[string]string{
		"app.title":          "Junimo Hut",
		"mods.search":        "Search mods...",
		"mods.install":       "Install Mod",
		"mods.checkUpdates":  "Check for Updates",
		"mods.readyToUpdate": "mods ready to update",
		"smapi.launch":       "Launch SMAPI",
		"profiles.new":       "New Profile",
		"categories.new":     "New Category",
		"settings.title":     "Settings",
		"setup.welcome":      "Welcome to Junimo Hut! Point us at your game folder and mod library to get started.",
		"setup.gamePath":     "Game Path",
		"setup.smapiPath":    "SMAPI Path",
		"setup.modsRoot":     "Mod Library",
		"setup.detect":       "Auto-detect",
		"setup.complete":     "Get Started",
	}
}
