package mods

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPropagateNexusUpdateStatus(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	list := []Mod{
		{
			ID:       "SVE::FlashShifter.SVECode",
			Manifest: Manifest{UniqueID: "FlashShifter.SVECode", UpdateKeys: []string{"Nexus:3753"}, Version: "1.15.10"},
			UpdateStatus: UpdateStatus{State: "current"},
		},
		{
			ID:       "SVE Content::FlashShifter.SVE-FTM",
			Manifest: Manifest{UniqueID: "FlashShifter.SVE-FTM", UpdateKeys: []string{"Nexus:3753"}, Version: "1.15.10"},
			UpdateStatus: UpdateStatus{
				State:         "update_available",
				LatestVersion: "1.15.11",
				ModPageURL:    "https://www.nexusmods.com/stardewvalley/mods/3753",
			},
		},
	}

	PropagateNexusUpdateStatus(list)

	must.Equal("update_available", list[0].UpdateStatus.State)
	must.Equal("1.15.11", list[0].UpdateStatus.LatestVersion)
	must.Equal("update_available", list[1].UpdateStatus.State)
}

func TestNexusSiblingFolderPaths(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	list := []Mod{
		{FolderPath: "Red Panda Bazaar", Manifest: Manifest{UpdateKeys: []string{"Nexus:999"}}},
		{FolderPath: "Red Panda Bazaar Code", Manifest: Manifest{UpdateKeys: []string{"Nexus:999"}}},
		{FolderPath: "Other Mod", Manifest: Manifest{UpdateKeys: []string{"Nexus:1"}}},
	}

	paths := NexusSiblingFolderPaths(list, "Red Panda Bazaar")
	must.ElementsMatch([]string{"Red Panda Bazaar", "Red Panda Bazaar Code"}, paths)
}

func TestApplyIgnoredUpdates(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	list := []Mod{
		{
			Manifest: Manifest{UpdateKeys: []string{"Nexus:42"}},
			UpdateStatus: UpdateStatus{
				State:         "update_available",
				LatestVersion: "2.0.0",
			},
		},
		{
			Manifest: Manifest{UpdateKeys: []string{"Nexus:42"}},
			UpdateStatus: UpdateStatus{
				State:         "update_available",
				LatestVersion: "2.0.0",
			},
		},
	}

	ApplyIgnoredUpdates(list, map[string]string{"42": "2.0.0"})
	must.Equal("update_ignored", list[0].UpdateStatus.State)
	must.Equal("update_ignored", list[1].UpdateStatus.State)
}
