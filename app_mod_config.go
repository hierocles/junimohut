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
		for _, child := range mod.BundleChildren {
			if child.ID == modID {
				return child, true
			}
		}
	}
	return mods.Mod{}, false
}

func (a *App) modDirForJSON(mod mods.Mod, modsRoot string) string {
	if mod.AbsolutePath != "" {
		return mod.AbsolutePath
	}
	return mods.ModDir(modsRoot, mod.FolderPath)
}

func (a *App) modHasJsonFiles(mod mods.Mod) bool {
	if mod.HasJsonFiles && mod.JsonFileCount > 0 {
		return true
	}
	settings := a.store.Get()
	return mods.CountJsonFiles(a.modDirForJSON(mod, settings.ModsRoot)) > 0
}

func (a *App) resolveConfigMod(modID string) (mods.Mod, bool) {
	mod, ok := a.modByID(modID)
	if !ok {
		return mods.Mod{}, false
	}
	if a.modHasJsonFiles(mod) {
		return mod, true
	}
	parent, ok := a.bundleParentFor(modID)
	if !ok || len(parent.BundleChildren) == 0 {
		return mod, ok
	}
	settings := a.store.Get()
	for _, child := range parent.BundleChildren {
		if a.modHasJsonFiles(child) {
			return child, true
		}
		if mods.CountJsonFiles(a.modDirForJSON(child, settings.ModsRoot)) > 0 {
			return child, true
		}
	}
	return mod, ok
}

func (a *App) resolveUpdateMod(modID string) (mods.Mod, bool) {
	return a.bundleParentFor(modID)
}

func (a *App) bundleParentFor(modID string) (mods.Mod, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, mod := range a.modsCache {
		if mod.ID == modID {
			return mod, true
		}
		for _, child := range mod.BundleChildren {
			if child.ID == modID {
				return mod, true
			}
		}
	}
	return mods.Mod{}, false
}

func (a *App) ListModsWithJsonFiles() []mods.ModJsonSummary {
	if err := a.ensureInit(); err != nil {
		return nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	settings := a.store.Get()
	out := make([]mods.ModJsonSummary, 0)
	for _, mod := range a.modsCache {
		a.appendModJsonSummary(&out, mod, settings.ModsRoot)
		for _, child := range mod.BundleChildren {
			a.appendModJsonSummary(&out, child, settings.ModsRoot)
		}
	}
	return out
}

func (a *App) appendModJsonSummary(out *[]mods.ModJsonSummary, mod mods.Mod, modsRoot string) {
	count := mod.JsonFileCount
	if count == 0 {
		count = mods.CountJsonFiles(a.modDirForJSON(mod, modsRoot))
	}
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

func (a *App) ListModJsonFiles(modID string) ([]mods.ModJsonFileNode, error) {
	if err := a.ensureInit(); err != nil {
		return nil, err
	}
	mod, ok := a.resolveConfigMod(modID)
	if !ok {
		return nil, fmt.Errorf("mod not found")
	}
	settings := a.store.Get()
	paths, err := mods.ListJsonFileRelPaths(a.modDirForJSON(mod, settings.ModsRoot))
	if err != nil {
		return nil, err
	}
	return mods.BuildJsonFileTree(paths), nil
}

func (a *App) modConfigFileView(modID, relPath string) (mods.ModConfigView, error) {
	if err := a.ensureInit(); err != nil {
		return mods.ModConfigView{}, err
	}
	mod, ok := a.resolveConfigMod(modID)
	if !ok {
		return mods.ModConfigView{}, fmt.Errorf("mod not found")
	}
	settings := a.store.Get()
	modDir := a.modDirForJSON(mod, settings.ModsRoot)
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
	mod, ok := a.resolveConfigMod(modID)
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
	mod, ok := a.resolveConfigMod(modID)
	if !ok {
		return fmt.Errorf("mod not found")
	}
	if !a.modHasJsonFiles(mod) {
		return fmt.Errorf("this mod has no JSON files")
	}

	view, err := a.modConfigFileView(mod.ID, relPath)
	if err != nil {
		return err
	}
	title := configEditorWindowTitle(view.ModName, filepath.Base(view.RelPath))
	editorURL := configEditorURL(mod.ID, view.RelPath)

	a.configEditorMu.Lock()
	defer a.configEditorMu.Unlock()

	if a.configEditorWindow != nil {
		a.configEditorModID = mod.ID
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
	a.configEditorModID = mod.ID
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
