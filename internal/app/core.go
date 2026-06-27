package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"junimohut/internal/categories"
	"junimohut/internal/config"
	"junimohut/internal/moddataset"
	"junimohut/internal/modnames"
	"junimohut/internal/modoverwrites"
	"junimohut/internal/mods"
	"junimohut/internal/modtimes"
	"junimohut/internal/modupdates"
	"junimohut/internal/nexus"
	"junimohut/internal/profiles"
	"junimohut/internal/smapi"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// CoreOptions configures shared application state constructed before Wails starts.
type CoreOptions struct {
	StartSMAPI bool
	CLIArgs    []string
}

// Core holds dependencies and shared state for Wails domain services.
type Core struct {
	ctx    context.Context
	App    *application.App
	Events EventPublisher

	Store           *config.Store
	Profiles        *profiles.Service
	Categories      *categories.Service
	ModNames        *modnames.Service
	ModTimes        *modtimes.Service
	ModUpdates      *modupdates.Service
	OverwriteMerges *modoverwrites.Service
	Nexus           *nexus.Client
	Downloads       *nexus.DownloadManager
	DownloadIndex   *nexus.DownloadIndex
	Scanner         *mods.Scanner
	ConfigMgr       *profiles.ConfigManager
	ModDataset      *moddataset.Index
	Catalog         *ModCatalog

	StartSMAPI bool
	CLIArgs    []string

	startupOnce sync.Once
	startupErr  error
}

// NewCore constructs an uninitialized Core. Call Startup via SystemService before RPC use.
func NewCore(opts CoreOptions) *Core {
	events := NewEventBridge()
	c := &Core{
		Events:     events,
		Scanner:    mods.NewScanner(),
		StartSMAPI: opts.StartSMAPI,
		CLIArgs:    append([]string(nil), opts.CLIArgs...),
	}
	c.Catalog = NewModCatalog(c)
	return c
}

// SetApplication attaches the Wails app for dialogs and events.
func (c *Core) SetApplication(app *application.App) {
	c.App = app
	if bridge, ok := c.Events.(*EventBridge); ok {
		bridge.SetApp(app)
	}
}

// Ctx returns the startup context.
func (c *Core) Ctx() context.Context {
	if c.ctx != nil {
		return c.ctx
	}
	return context.Background()
}

// RequireStarted returns the startup error if initialization failed.
func (c *Core) RequireStarted() error {
	return c.startupErr
}

// Startup wires services and performs first mod scan. Idempotent.
func (c *Core) Startup(ctx context.Context) error {
	c.startupOnce.Do(func() {
		slog.Info("Core.Startup")
		c.ctx = ctx
		c.startupErr = c.startup(ctx)
	})
	return c.startupErr
}

func (c *Core) startup(ctx context.Context) error {
	c.ctx = ctx
	store, err := config.NewStore()
	if err != nil {
		return err
	}
	c.Store = store
	_ = store.EnsureDirs()

	for _, arg := range c.CLIArgs {
		if strings.HasPrefix(arg, "--start-smapi") {
			c.StartSMAPI = strings.HasSuffix(arg, "true") || arg == "--start-smapi"
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
	c.Profiles = profSvc
	c.ConfigMgr = profiles.NewConfigManager(store.ProfilesDir(), profSvc)

	catSvc, err := categories.NewService(store.CategoriesPath())
	if err != nil {
		return err
	}
	c.Categories = catSvc

	modNamesSvc, err := modnames.NewService(store.ModNamesPath())
	if err != nil {
		return err
	}
	c.ModNames = modNamesSvc

	modTimesSvc, err := modtimes.NewService(store.ModTimesPath())
	if err != nil {
		return err
	}
	c.ModTimes = modTimesSvc

	modUpdatesSvc, err := modupdates.NewService(store.ModUpdatesPath())
	if err != nil {
		return err
	}
	c.ModUpdates = modUpdatesSvc

	overwriteSvc, err := modoverwrites.NewService(store.OverwriteMergesPath())
	if err != nil {
		return err
	}
	c.OverwriteMerges = overwriteSvc

	c.Nexus = nexus.NewClient()
	downloadIndex, err := nexus.NewDownloadIndex(store.DataDir(), store.DownloadsDir())
	if err != nil {
		return err
	}
	downloadIndex.SetArchiveEnricher(enrichDownloadRecordFromArchive)
	c.DownloadIndex = downloadIndex
	c.Downloads = nexus.NewDownloadManager(store.DownloadsDir(), downloadIndex)
	go downloadIndex.ReconcileAsync()

	modDatasetIdx, err := moddataset.NewIndex(store.ModDatasetIndexPath(), store.ModDatasetDir())
	if err != nil {
		return err
	}
	c.ModDataset = modDatasetIdx
	go modDatasetIdx.RefreshIfStaleAsync()

	_ = c.Catalog.Refresh(ctx)
	_, _ = mods.NewWatcher(store.Get().ModsRoot, func() {
		_ = c.Catalog.Refresh(c.Ctx())
		c.Events.EmitModsChanged()
	})

	if c.StartSMAPI {
		go func() {
			time.Sleep(2 * time.Second)
			_ = smapiServiceLaunch(c)
		}()
	}
	return nil
}

// SMAPIVersion returns the installed SMAPI version string.
func (c *Core) SMAPIVersion() string {
	if c.startupErr != nil || c.Store == nil {
		return ""
	}
	settings := c.Store.Get()
	return smapi.NewLauncher(settings.GamePath, settings.SMAPIPath).Version()
}

func smapiServiceLaunch(c *Core) error {
	settings := c.Store.Get()
	if err := c.Catalog.Refresh(c.Ctx()); err != nil {
		return err
	}
	if settings.ProfileSpecificConfigs {
		if err := c.ConfigMgr.SaveConfigs(settings.ModsRoot, modUniqueIDMapFromEnabled(c, c.Profiles.EnabledMods())); err != nil {
			return err
		}
	}
	launcher := smapi.NewLauncher(settings.GamePath, settings.SMAPIPath)
	return launcher.Launch()
}

// ProcessCLIArgs handles nxm:// URLs from the command line after startup.
func (c *Core) ProcessCLIArgs() {
	for _, arg := range c.CLIArgs {
		if strings.HasPrefix(arg, "nxm://") {
			c.Events.EmitNXMURL(arg)
			return
		}
	}
}

// BrowseFolder opens a native directory picker.
func (c *Core) BrowseFolder(title string) (string, error) {
	if c.startupErr != nil {
		return "", c.startupErr
	}
	if c.App == nil {
		return "", fmt.Errorf("application not ready")
	}
	path, err := c.App.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		Title:                title,
		CanChooseFiles:       false,
		CanChooseDirectories: true,
	}).PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	return path, nil
}

// SelectArchivePaths opens a native multi-file picker for mod archives.
func (c *Core) SelectArchivePaths() ([]string, error) {
	if c.startupErr != nil {
		return nil, c.startupErr
	}
	if c.App == nil {
		return nil, fmt.Errorf("application not ready")
	}
	paths, err := c.App.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
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

// ParseStartSMAPIFlag reads --start-smapi from os.Args when CoreOptions omit it.
func ParseStartSMAPIFlag() bool {
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "--start-smapi") {
			return strings.HasSuffix(arg, "true") || arg == "--start-smapi"
		}
	}
	return false
}
