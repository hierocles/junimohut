package app

import (
	"fmt"
	"runtime"

	"junimohut/internal/config"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type SettingsService struct {
	core *Core
}

func NewSettingsService(core *Core) *SettingsService {
	return &SettingsService{core: core}
}

func (s *SettingsService) GetSettings() config.Settings {
	if err := s.core.RequireStarted(); err != nil {
		return config.DefaultSettings()
	}
	return s.core.Store.Get()
}

func (s *SettingsService) SaveSettings(settings config.Settings) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	prepared, err := config.EnsureModLibrary(settings, s.core.Store.DataDir())
	if err != nil {
		return err
	}
	if err := s.core.Store.Set(prepared); err != nil {
		return err
	}
	return s.core.Catalog.Refresh(s.core.Ctx())
}

func (s *SettingsService) DetectPaths() config.DetectionResult {
	if err := s.core.RequireStarted(); err != nil {
		return config.DetectPaths("")
	}
	return config.DetectPaths(config.DefaultModLibrary(s.core.Store.DataDir()))
}

func (s *SettingsService) CompleteSetup(gamePath, smapiPath, modsRoot string) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	if err := s.core.Store.Update(func(s *config.Settings) {
		s.GamePath = gamePath
		s.SMAPIPath = smapiPath
		s.ModsRoot = modsRoot
		s.SetupComplete = true
	}); err != nil {
		return err
	}
	settings := s.core.Store.Get()
	prepared, err := config.EnsureModLibrary(settings, s.core.Store.DataDir())
	if err != nil {
		return err
	}
	if prepared.ModsRoot != settings.ModsRoot {
		if err := s.core.Store.Set(prepared); err != nil {
			return err
		}
	}
	return s.core.Catalog.Refresh(s.core.Ctx())
}

func (s *SettingsService) BrowseGameFolder() (string, error) {
	return s.core.BrowseFolder("Select Stardew Valley game folder")
}

func (s *SettingsService) BrowseModsRoot() (string, error) {
	return s.core.BrowseFolder("Select mod library folder")
}

func (s *SettingsService) BrowseSMAPIPath() (string, error) {
	if err := s.core.RequireStarted(); err != nil {
		return "", err
	}
	if s.core.App == nil {
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
	path, err := s.core.App.Dialog.OpenFileWithOptions(opts).PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	return path, nil
}
