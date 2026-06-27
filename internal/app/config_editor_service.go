package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"junimohut/internal/mods"
	"junimohut/internal/platform"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type ConfigEditorService struct {
	core *Core
	editorMu sync.Mutex
	editorWindow *application.WebviewWindow
	editorModID string
	editorDirty bool
}
func NewConfigEditorService(core *Core) *ConfigEditorService { return &ConfigEditorService{core: core} }

func (s *ConfigEditorService) ListModsWithJsonFiles() []mods.ModJsonSummary {
	if err := s.core.RequireStarted(); err != nil {
		return nil
	}
	s.core.Catalog.mu.RLock()
	defer s.core.Catalog.mu.RUnlock()
	settings := s.core.Store.Get()
	out := make([]mods.ModJsonSummary, 0)
	for _, mod := range s.core.Catalog.mods {
		if len(mod.BundleChildren) > 0 {
			for _, child := range mod.BundleChildren {
				s.appendModJsonSummary(&out, child, settings.ModsRoot)
			}
		} else {
			s.appendModJsonSummary(&out, mod, settings.ModsRoot)
		}
	}
	return out
}

func (s *ConfigEditorService) ListModJsonFiles(modID string) ([]mods.ModJsonFileNode, error) {
	if err := s.core.RequireStarted(); err != nil {
		return nil, err
	}
	mod, ok := resolveConfigMod(s.core, modID)
	if !ok {
		return nil, fmt.Errorf("mod not found")
	}
	settings := s.core.Store.Get()
	paths, err := mods.ListJsonFileRelPaths(modDirForJSON(mod, settings.ModsRoot))
	if err != nil {
		return nil, err
	}
	return mods.BuildJsonFileTree(paths), nil
}

func (s *ConfigEditorService) GetModConfig(modID string) (mods.ModConfigView, error) {
	return s.modConfigFileView(modID, "")
}

func (s *ConfigEditorService) GetModConfigFile(modID, relPath string) (mods.ModConfigView, error) {
	return s.modConfigFileView(modID, relPath)
}

func (s *ConfigEditorService) SaveModConfig(modID, content string) error {
	return s.SaveModConfigFile(modID, "config.json", content)
}

func (s *ConfigEditorService) SaveModConfigFile(modID, relPath, content string) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	mod, ok := resolveConfigMod(s.core, modID)
	if !ok {
		return fmt.Errorf("mod not found")
	}
	settings := s.core.Store.Get()
	if err := mods.WriteModJsonFile(settings.ModsRoot, mod.FolderPath, relPath, content); err != nil {
		return err
	}
	if settings.ProfileSpecificConfigs {
		_ = s.core.ConfigMgr.SaveModConfig(settings.ModsRoot, mod.ID, mod.Manifest.UniqueID)
	}
	s.SetConfigEditorDirty(false)
	s.core.Events.EmitModsChanged()
	return nil
}

func (s *ConfigEditorService) OpenModConfigExternal(modID string) error {
	return s.OpenModConfigExternalFile(modID, "")
}

func (s *ConfigEditorService) OpenModConfigExternalFile(modID, relPath string) error {
	view, err := s.modConfigFileView(modID, relPath)
	if err != nil {
		return err
	}
	return platform.OpenPath(view.AbsolutePath)
}

func (s *ConfigEditorService) SetConfigEditorDirty(dirty bool) {
	s.editorMu.Lock()
	s.editorDirty = dirty
	s.editorMu.Unlock()
}

func (s *ConfigEditorService) ConfigEditorIsDirty() bool {
	s.editorMu.Lock()
	defer s.editorMu.Unlock()
	return s.editorDirty
}

func (s *ConfigEditorService) OpenModConfigEditor(modID string) error {
	return s.OpenModConfigEditorFile(modID, "")
}

func (s *ConfigEditorService) OpenModConfigEditorFile(modID, relPath string) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	if s.core.App == nil {
		return fmt.Errorf("application not ready")
	}
	mod, ok := resolveConfigMod(s.core, modID)
	if !ok {
		return fmt.Errorf("mod not found")
	}
	if !modHasJsonFiles(s.core, mod) {
		return fmt.Errorf("this mod has no JSON files")
	}

	view, err := s.modConfigFileView(mod.ID, relPath)
	if err != nil {
		return err
	}
	title := configEditorWindowTitle(view.ModName, filepath.Base(view.RelPath))
	editorURL := configEditorURL(mod.ID, view.RelPath)

	s.editorMu.Lock()
	defer s.editorMu.Unlock()

	if s.editorWindow != nil {
		s.editorModID = mod.ID
		s.editorWindow.SetTitle(title)
		s.editorWindow.EmitEvent("config-editor-open-mod", map[string]string{
			"modId":   modID,
			"relPath": view.RelPath,
		})
		s.editorWindow.Show()
		s.editorWindow.Focus()
		return nil
	}

	w := s.core.App.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "config-editor",
		Title:            title,
		Width:            1120,
		Height:           680,
		MinWidth:         760,
		MinHeight:        480,
		Frameless:        true,
		BackgroundColour: application.NewRGB(22, 23, 28),
		URL:              editorURL,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 40,
			Backdrop:                application.MacBackdropTranslucent,
		},
	})
	w.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		s.editorMu.Lock()
		s.editorWindow = nil
		s.editorModID = ""
		s.editorDirty = false
		s.editorMu.Unlock()
	})
	s.editorWindow = w
	s.editorModID = mod.ID
	return nil
}

func (s *ConfigEditorService) ReloadConfigEditor() {
	s.editorMu.Lock()
	w := s.editorWindow
	s.editorMu.Unlock()
	if w != nil {
		w.EmitEvent("config-editor-reload", true)
	}
}

func (s *ConfigEditorService) appendModJsonSummary(out *[]mods.ModJsonSummary, mod mods.Mod, modsRoot string) {
	count := mods.CountJsonFiles(modDirForJSON(mod, modsRoot))
	if count == 0 {
		return
	}
	*out = append(*out, mods.ModJsonSummary{
		ModID:         mod.ID,
		ModName:       mod.Manifest.Name,
		FolderPath:    mod.FolderPath,
		JsonFileCount: count,
	})
}

func (s *ConfigEditorService) modConfigFileView(modID, relPath string) (mods.ModConfigView, error) {
	if err := s.core.RequireStarted(); err != nil {
		return mods.ModConfigView{}, err
	}
	mod, ok := resolveConfigMod(s.core, modID)
	if !ok {
		return mods.ModConfigView{}, fmt.Errorf("mod not found")
	}
	settings := s.core.Store.Get()
	modDir := modDirForJSON(mod, settings.ModsRoot)
	paths, err := mods.ListJsonFileRelPaths(modDir)
	if err != nil {
		return mods.ModConfigView{}, err
	}
	if len(paths) == 0 {
		return mods.ModConfigView{}, fmt.Errorf("this mod has no JSON files")
	}
	if strings.TrimSpace(relPath) == "" {
		relPath = mods.DefaultJsonRelPath(paths)
	}
	content, err := mods.ReadModJsonFile(settings.ModsRoot, mod.FolderPath, relPath)
	if err != nil {
		return mods.ModConfigView{}, err
	}
	abs, err := mods.ResolveModJSONPath(settings.ModsRoot, mod.FolderPath, relPath)
	if err != nil {
		return mods.ModConfigView{}, err
	}
	displayPath := filepath.ToSlash(filepath.Join("Mods", mod.FolderPath, relPath))
	return mods.ModConfigView{
		ModID:                  mod.ID,
		ModName:                mod.Manifest.Name,
		FolderPath:             mod.FolderPath,
		RelPath:                relPath,
		DisplayPath:            displayPath,
		AbsolutePath:           abs,
		Content:                content,
		ProfileName:            s.core.Profiles.Active().Name,
		ProfileSpecificConfigs: settings.ProfileSpecificConfigs,
	}, nil
}
