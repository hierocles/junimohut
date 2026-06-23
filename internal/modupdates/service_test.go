package modupdates

import (
	"path/filepath"
	"testing"

	"junimohut/internal/config"
	"junimohut/internal/mods"

	"github.com/stretchr/testify/require"
)

func TestServiceSyncAndRestore(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	dir := t.TempDir()
	store, err := config.NewStoreForDir(dir, config.DefaultSettings())
	must.NoError(err)

	svc, err := NewService(store.ModUpdatesPath())
	must.NoError(err)

	list := []mods.Mod{
		{
			ID:       "FolderA::Author.ModA",
			Manifest: mods.Manifest{Version: "1.0.0"},
			UpdateStatus: mods.UpdateStatus{
				State:         "update_available",
				LatestVersion: "2.0.0",
			},
		},
		{
			ID:       "FolderB::Author.ModB",
			Manifest: mods.Manifest{Version: "1.0.0"},
			UpdateStatus: mods.UpdateStatus{State: "current"},
		},
	}
	must.NoError(svc.SyncFromMods(list))

	reloaded, err := NewService(store.ModUpdatesPath())
	must.NoError(err)
	cached := reloaded.All()
	must.Len(cached, 1)
	must.Equal("2.0.0", cached["FolderA::Author.ModA"].LatestVersion)

	fresh := []mods.Mod{
		{ID: "FolderA::Author.ModA", Manifest: mods.Manifest{Version: "1.0.0"}},
	}
	raw := make(map[string]mods.CachedUpdate, len(cached))
	for id, e := range cached {
		raw[id] = mods.CachedUpdate{
			ManifestVersion: e.ManifestVersion,
			State:           e.State,
			LatestVersion:   e.LatestVersion,
			ModPageURL:      e.ModPageURL,
			Message:         e.Message,
		}
	}
	mods.ApplyCachedUpdateStatus(fresh, raw)
	must.Equal("update_available", fresh[0].UpdateStatus.State)

	updated := []mods.Mod{
		{
			ID:           "FolderA::Author.ModA",
			Manifest:     mods.Manifest{Version: "2.0.0"},
			UpdateStatus: mods.UpdateStatus{State: "current"},
		},
	}
	must.NoError(svc.SyncFromMods(updated))

	reloaded2, err := NewService(store.ModUpdatesPath())
	must.NoError(err)
	must.Empty(reloaded2.All())

	_, err = filepath.Abs(store.ModUpdatesPath())
	must.NoError(err)
}
