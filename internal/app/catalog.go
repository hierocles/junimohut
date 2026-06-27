package app

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"junimohut/internal/config"
	"junimohut/internal/moddataset"
	"junimohut/internal/mods"
	"junimohut/internal/nexus"
	"junimohut/internal/profiles"
)

// ModCatalog owns the scanned mod library cache and profile assembly.
// Lock order: refreshMu → mu → assembleMu.
type ModCatalog struct {
	core *Core

	mu         sync.RWMutex
	refreshMu  sync.Mutex
	assembleMu sync.Mutex

	mods       []mods.Mod
	duplicates []mods.DuplicateModGroup
	unmanaged  []profiles.UnmanagedMod
}

// NewModCatalog creates a catalog bound to core dependencies.
func NewModCatalog(core *Core) *ModCatalog {
	return &ModCatalog{core: core}
}

// CopyMods returns a snapshot of cached mods.
func (c *ModCatalog) CopyMods() []mods.Mod {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]mods.Mod{}, c.mods...)
}

// ListFiltered returns mods matching search/hide filters.
func (c *ModCatalog) ListFiltered(search, hideDisabled string) []mods.Mod {
	return mods.FilterMods(c.CopyMods(), search, hideDisabled)
}

// CopyDuplicates returns cached duplicate mod groups.
func (c *ModCatalog) CopyDuplicates() []mods.DuplicateModGroup {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]mods.DuplicateModGroup{}, c.duplicates...)
}

// CopyUnmanaged returns cached unmanaged mod entries.
func (c *ModCatalog) CopyUnmanaged() []profiles.UnmanagedMod {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]profiles.UnmanagedMod{}, c.unmanaged...)
}

// UnmanagedCount returns the number of unmanaged mods.
func (c *ModCatalog) UnmanagedCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.unmanaged)
}

// DuplicateCount returns the number of duplicate mod groups.
func (c *ModCatalog) DuplicateCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.duplicates)
}

// WithRead runs fn while holding the read lock on the mod cache.
func (c *ModCatalog) WithRead(fn func([]mods.Mod)) {
	c.mu.RLock()
	fn(c.mods)
	c.mu.RUnlock()
}

// WithWrite runs fn while holding the write lock on the mod cache.
func (c *ModCatalog) WithWrite(fn func(*[]mods.Mod)) {
	c.mu.Lock()
	fn(&c.mods)
	c.mu.Unlock()
}

// Refresh rescans the mod library and kicks off async assembly.
func (c *ModCatalog) Refresh(ctx context.Context) error {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()

	settings := c.core.Store.Get()
	enabled := c.core.Profiles.EnabledMods()
	list, err := c.core.Scanner.Scan(mods.ScanOptions{
		ModsRoot:            settings.ModsRoot,
		IgnoreHiddenFolders: settings.IgnoreHiddenFolders,
		EnabledMods:         enabled,
		SkipPackCollapse:    true,
	})
	if err != nil {
		return err
	}
	c.enrichModTimes(list)
	list = mods.CollapseSiblingPacks(list, settings.ModsRoot, enabled)
	mods.StripBundleChildUpdateStatus(list)
	c.mu.Lock()
	c.duplicates = mods.DetectDuplicateMods(list)
	c.mu.Unlock()
	c.migrateBundleTagAssignments(list)
	for i := range list {
		list[i].CategoryIDs = c.core.Categories.ModCategoryIDs(list[i].ID)
		list[i].CustomName = mods.EffectiveCustomName(
			c.core.ModNames.Get(list[i].ID),
			list[i].FolderPath,
			list[i].Manifest.Name,
			mods.DisplayNameOfficial,
		)
		list[i].ContainsOverwrites = c.core.OverwriteMerges.ContainsOverwrites(list[i].ID)
		if len(list[i].BundleChildren) > 0 {
			for j := range list[i].BundleChildren {
				child := &list[i].BundleChildren[j]
				child.CategoryIDs = c.core.Categories.ModCategoryIDs(child.ID)
				child.CustomName = mods.EffectiveCustomName(
					c.core.ModNames.Get(child.ID),
					child.FolderPath,
					child.Manifest.Name,
					mods.DisplayNameOfficial,
				)
				child.ContainsOverwrites = c.core.OverwriteMerges.ContainsOverwrites(child.ID)
			}
		}
	}
	list = mods.DedupeByUniqueID(mods.DedupeByID(list))
	list = mods.ResolveDependencies(list)
	c.enrichModsFromDataset(list)
	c.mu.Lock()
	previous := append([]mods.Mod{}, c.mods...)
	c.mu.Unlock()
	mods.PreserveUpdateStatus(list, previous)
	mods.ApplyCachedUpdateStatus(list, cachedUpdatesFromService(c.core.ModUpdates))
	mods.ApplyIgnoredUpdates(list, settings.IgnoredModUpdates)
	c.mu.Lock()
	c.mods = list
	c.mu.Unlock()

	go c.finishRefresh(list, enabled, settings)
	return nil
}

func (c *ModCatalog) migrateBundleTagAssignments(list []mods.Mod) {
	for _, m := range list {
		if len(m.BundleChildren) == 0 {
			continue
		}
		for _, child := range m.BundleChildren {
			for _, catID := range c.core.Categories.ModCategoryIDs(child.ID) {
				if err := c.core.Categories.AssignMod(catID, m.ID); err != nil {
					continue
				}
				_ = c.core.Categories.UnassignMod(catID, child.ID)
			}
		}
	}
}

// RefreshCategoryIDs updates category IDs on cached mods after assignment changes.
func (c *ModCatalog) RefreshCategoryIDs() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.mods {
		c.mods[i].CategoryIDs = c.core.Categories.ModCategoryIDs(c.mods[i].ID)
		for j := range c.mods[i].BundleChildren {
			childID := c.mods[i].BundleChildren[j].ID
			c.mods[i].BundleChildren[j].CategoryIDs = c.core.Categories.ModCategoryIDs(childID)
		}
	}
}

// SyncModUpdateCache persists update status from the mod cache.
func (c *ModCatalog) SyncModUpdateCache() {
	if c.core.ModUpdates == nil {
		return
	}
	list := c.CopyMods()
	_ = c.core.ModUpdates.SyncFromMods(list)
}

func (c *ModCatalog) enrichModTimes(list []mods.Mod) {
	seeds := map[string]int64{}
	for i := range list {
		if rec, ok := c.core.ModTimes.Get(list[i].ID); ok {
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
	_ = c.core.ModTimes.SeedBatch(seeds)
}

func (c *ModCatalog) enrichModsFromDataset(list []mods.Mod) {
	if c.core.ModDataset == nil {
		return
	}
	enrich := func(mod *mods.Mod) {
		manifestNexus := nexus.ModIDFromUpdateKeys(mod.Manifest.UpdateKeys)
		downloadNexus := 0
		if c.core.DownloadIndex != nil {
			downloadNexus = c.core.DownloadIndex.NexusModIDForMod(mod.Manifest.UniqueID, manifestNexus)
		}
		mod.ResolvedNexusModID = moddataset.ResolveNexusModID(
			mod.Manifest.UniqueID,
			mod.Manifest.UpdateKeys,
			c.core.ModDataset,
			downloadNexus,
			mod.BundleNexusID,
		)
		if mod.UpdateStatus.ModPageURL == "" && mod.ResolvedNexusModID > 0 {
			mod.UpdateStatus.ModPageURL = fmt.Sprintf(
				"https://www.nexusmods.com/stardewvalley/mods/%d",
				mod.ResolvedNexusModID,
			)
		}
	}
	for i := range list {
		enrich(&list[i])
		for j := range list[i].BundleChildren {
			enrich(&list[i].BundleChildren[j])
		}
	}
}

func (c *ModCatalog) finishRefresh(list []mods.Mod, enabled map[string]bool, settings config.Settings) {
	c.assembleMu.Lock()
	defer c.assembleMu.Unlock()

	for i := range list {
		if list[i].IsCoreMod {
			continue
		}
		nexusModID := list[i].ResolvedNexusModID
		if nexusModID == 0 {
			nexusModID = nexus.ModIDFromUpdateKeys(list[i].Manifest.UpdateKeys)
		}
		if path, ok := c.core.DownloadIndex.FindForMod(list[i].Manifest.UniqueID, nexusModID); ok {
			list[i].SavedDownloadPath = path
		}
	}
	c.mu.Lock()
	c.mods = list
	c.mu.Unlock()

	activeModsDir := config.ActiveModsDir(settings.GamePath)
	assembler := profiles.NewAssembler(activeModsDir, settings.ModsRoot)
	if err := assembler.Assemble(list, enabled); err != nil {
		slog.Warn("mod assembly failed", "error", err)
		return
	}
	unmanaged, err := profiles.ScanUnmanagedMods(activeModsDir, settings.ModsRoot)
	if err != nil {
		slog.Warn("scan unmanaged mods failed", "error", err)
		return
	}
	c.mu.Lock()
	c.unmanaged = unmanaged
	c.mu.Unlock()
}

// CountUpdateBadge returns mods matching an update status state.
func (c *ModCatalog) CountUpdateBadge(match func(mods.Mod) bool) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	n := 0
	for _, m := range c.mods {
		if match(m) {
			n++
		}
	}
	return n
}

// ApplyIgnoredUpdates refreshes ignored-update flags on the cache.
func (c *ModCatalog) ApplyIgnoredUpdates(ignored map[string]string) {
	c.mu.Lock()
	mods.ApplyIgnoredUpdates(c.mods, ignored)
	c.mu.Unlock()
}

// ClearUpdateStatusAfterModUpdate clears update status for updated folder paths.
func (c *ModCatalog) ClearUpdateStatusAfterModUpdate(targets []string) {
	c.mu.Lock()
	mods.ClearUpdateStatusAfterModUpdate(c.mods, targets)
	c.mu.Unlock()
}
