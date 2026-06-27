package app

import (
	"junimohut/internal/mods"
)

func modByID(c *Core, modID string) (mods.Mod, bool) {
	c.Catalog.mu.RLock()
	defer c.Catalog.mu.RUnlock()
	for _, mod := range c.Catalog.mods {
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

func modByFolderPath(c *Core, folderPath string) (mods.Mod, bool) {
	c.Catalog.mu.RLock()
	defer c.Catalog.mu.RUnlock()
	return mods.FindModByFolderPath(c.Catalog.mods, folderPath)
}

func bundleParentFor(c *Core, modID string) (mods.Mod, bool) {
	c.Catalog.mu.RLock()
	defer c.Catalog.mu.RUnlock()
	for _, mod := range c.Catalog.mods {
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

func resolveConfigMod(c *Core, modID string) (mods.Mod, bool) {
	mod, ok := modByID(c, modID)
	if !ok {
		return mods.Mod{}, false
	}
	if modHasJsonFiles(c, mod) {
		return mod, true
	}
	parent, ok := bundleParentFor(c, modID)
	if !ok || len(parent.BundleChildren) == 0 {
		return mod, ok
	}
	settings := c.Store.Get()
	for _, child := range parent.BundleChildren {
		if modHasJsonFiles(c, child) {
			return child, true
		}
		if mods.CountJsonFiles(modDirForJSON(child, settings.ModsRoot)) > 0 {
			return child, true
		}
	}
	return mod, ok
}

func resolveUpdateMod(c *Core, modID string) (mods.Mod, bool) {
	return bundleParentFor(c, modID)
}

func modUniqueIDMap(c *Core) map[string]string {
	return modUniqueIDMapFromEnabled(c, c.Profiles.EnabledMods())
}

func modUniqueIDMapFromEnabled(c *Core, enabled map[string]bool) map[string]string {
	c.Catalog.mu.RLock()
	defer c.Catalog.mu.RUnlock()
	m := map[string]string{}
	for _, mod := range c.Catalog.mods {
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

func modUniqueIDFor(c *Core, modID string) string {
	c.Catalog.mu.RLock()
	defer c.Catalog.mu.RUnlock()
	for _, mod := range c.Catalog.mods {
		if mod.ID == modID {
			return mod.Manifest.UniqueID
		}
	}
	return ""
}

func modDirForJSON(mod mods.Mod, modsRoot string) string {
	if mod.AbsolutePath != "" {
		return mod.AbsolutePath
	}
	return mods.ModDir(modsRoot, mod.FolderPath)
}

func modHasJsonFiles(c *Core, mod mods.Mod) bool {
	if mod.HasJsonFiles && mod.JsonFileCount > 0 {
		return true
	}
	settings := c.Store.Get()
	return mods.CountJsonFiles(modDirForJSON(mod, settings.ModsRoot)) > 0
}
