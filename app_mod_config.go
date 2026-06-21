package main

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"junimohut/internal/mods"
	"junimohut/internal/platform"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func (a *App) modByID(modID string) (mods.Mod, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, mod := range a.modsCache {
		if mod.ID == modID {
			return mod, true
		}
	}
	return mods.Mod{}, false
}

func (a *App) modHasJsonFiles(mod mods.Mod) bool {
	if mod.HasJsonFiles {
		return true
	}
	settings := a.store.Get()
	return mods.CountJsonFiles(mods.ModDir(settings.ModsRoot, mod.FolderPath)) > 0
}

func (a *App) ListModsWithJsonFiles() []mods.ModJsonSummary {
	if err := a.ensureInit(); err != nil {
		return nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]mods.ModJsonSummary, 0)
	for _, mod := range a.modsCache {
		count := mod.JsonFileCount
		if count == 0 {
			settings := a.store.Get()
			count = mods.CountJsonFiles(mods.ModDir(settings.ModsRoot, mod.FolderPath))
		}
		if count == 0 {
			continue
		}
		out = append(out, mods.ModJsonSummary{
			ModID:         mod.ID,
			ModName:       mod.Manifest.Name,
			FolderPath:    mod.FolderPath,
			JsonFileCount: count,
		})
	}
	return out
}

func (a *App) ListModJsonFiles(modID string) ([]mods.ModJsonFileNode, error) {
	if err := a.ensureInit(); err != nil {
		return nil, err
	}
	mod, ok := a.modByID(modID)
	if !ok {
		return nil, fmt.Errorf("mod not found")
	}
	settings := a.store.Get()
	paths, err := mods.ListJsonFileRelPaths(mods.ModDir(settings.ModsRoot, mod.FolderPath))
	if err != nil {
		return nil, err
	}
	return mods.BuildJsonFileTree(paths), nil
}

func (a *App) modConfigFileView(modID, relPath string) (mods.ModConfigView, error) {
	if err := a.ensureInit(); err != nil {
		return mods.ModConfigView{}, err
	}
	mod, ok := a.modByID(modID)
	if !ok {
		return mods.ModConfigView{}, fmt.Errorf("mod not found")
	}
	settings := a.store.Get()
	modDir := mods.ModDir(settings.ModsRoot, mod.FolderPath)
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
		ProfileName:            a.profiles.Active().Name,
		ProfileSpecificConfigs: settings.ProfileSpecificConfigs,
	}, nil
}

func (a *App) GetModConfig(modID string) (mods.ModConfigView, error) {
	return a.modConfigFileView(modID, "")
}

func (a *App) GetModConfigFile(modID, relPath string) (mods.ModConfigView, error) {
	return a.modConfigFileView(modID, relPath)
}

func (a *App) SaveModConfig(modID, content string) error {
	return a.SaveModConfigFile(modID, "config.json", content)
}

func (a *App) SaveModConfigFile(modID, relPath, content string) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	mod, ok := a.modByID(modID)
	if !ok {
		return fmt.Errorf("mod not found")
	}
	settings := a.store.Get()
	if err := mods.WriteModJsonFile(settings.ModsRoot, mod.FolderPath, relPath, content); err != nil {
		return err
	}
	if settings.ProfileSpecificConfigs {
		_ = a.configMgr.SaveModConfig(settings.ModsRoot, mod.ID, mod.Manifest.UniqueID)
	}
	a.SetConfigEditorDirty(false)
	a.emitModsChanged()
	return nil
}

func (a *App) OpenModConfigExternal(modID string) error {
	return a.OpenModConfigExternalFile(modID, "")
}

func (a *App) OpenModConfigExternalFile(modID, relPath string) error {
	view, err := a.modConfigFileView(modID, relPath)
	if err != nil {
		return err
	}
	return platform.OpenPath(view.AbsolutePath)
}

func (a *App) SetConfigEditorDirty(dirty bool) {
	a.configEditorMu.Lock()
	a.configEditorDirty = dirty
	a.configEditorMu.Unlock()
}

func (a *App) ConfigEditorIsDirty() bool {
	a.configEditorMu.Lock()
	defer a.configEditorMu.Unlock()
	return a.configEditorDirty
}

func (a *App) OpenModConfigEditor(modID string) error {
	return a.OpenModConfigEditorFile(modID, "")
}

func (a *App) OpenModConfigEditorFile(modID, relPath string) error {
	if err := a.ensureInit(); err != nil {
		return err
	}
	if a.app == nil {
		return fmt.Errorf("application not ready")
	}
	mod, ok := a.modByID(modID)
	if !ok {
		return fmt.Errorf("mod not found")
	}
	if !a.modHasJsonFiles(mod) {
		return fmt.Errorf("this mod has no JSON files")
	}

	view, err := a.modConfigFileView(modID, relPath)
	if err != nil {
		return err
	}
	title := configEditorWindowTitle(view.ModName, filepath.Base(view.RelPath))
	editorURL := configEditorURL(modID, view.RelPath)

	a.configEditorMu.Lock()
	defer a.configEditorMu.Unlock()

	if a.configEditorWindow != nil {
		a.configEditorModID = modID
		a.configEditorWindow.SetTitle(title)
		a.configEditorWindow.EmitEvent("config-editor-open-mod", map[string]string{
			"modId":   modID,
			"relPath": view.RelPath,
		})
		a.configEditorWindow.Show()
		a.configEditorWindow.Focus()
		return nil
	}

	w := a.app.Window.NewWithOptions(application.WebviewWindowOptions{
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
		a.configEditorMu.Lock()
		a.configEditorWindow = nil
		a.configEditorModID = ""
		a.configEditorDirty = false
		a.configEditorMu.Unlock()
	})
	a.configEditorWindow = w
	a.configEditorModID = modID
	return nil
}

func (a *App) ReloadConfigEditor() {
	a.configEditorMu.Lock()
	w := a.configEditorWindow
	a.configEditorMu.Unlock()
	if w != nil {
		w.EmitEvent("config-editor-reload", true)
	}
}

func configEditorWindowTitle(modName, fileName string) string {
	name := strings.TrimSpace(modName)
	if name == "" {
		name = "Mod"
	}
	file := strings.TrimSpace(fileName)
	if file == "" {
		file = "config.json"
	}
	return fmt.Sprintf("%s — %s", name, file)
}

func configEditorURL(modID, relPath string) string {
	u := "/config-editor.html?modId=" + url.QueryEscape(modID)
	if strings.TrimSpace(relPath) != "" {
		u += "&file=" + url.QueryEscape(relPath)
	}
	return u
}
