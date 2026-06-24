package mods

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDedupeByUniqueIDPrefersNewerVersion(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	list := []Mod{
		{
			FolderPath:   "RedPandaBazaar_old",
			LastUpdated:  1,
			Manifest:     Manifest{UniqueID: "Author.RedPandaBazaar", Version: "1.0.0"},
			UpdateStatus: UpdateStatus{State: "update_available", LatestVersion: "2.0.0"},
		},
		{
			FolderPath:  "RedPandaBazaar",
			LastUpdated: 2,
			Manifest:    Manifest{UniqueID: "Author.RedPandaBazaar", Version: "2.0.0"},
		},
	}

	out := DedupeByUniqueID(list)
	must.Len(out, 1)
	must.Equal("2.0.0", out[0].Manifest.Version)
	must.Equal("RedPandaBazaar", out[0].FolderPath)
}

func TestClearUpdateStatusAfterModUpdate(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	list := []Mod{
		{
			FolderPath: "RedPandaBazaar",
			Manifest: Manifest{
				UniqueID:   "Author.RedPandaBazaar",
				Version:    "2.0.0",
				UpdateKeys: []string{"Nexus:123"},
			},
			UpdateStatus: UpdateStatus{State: "update_available", LatestVersion: "2.0.0"},
		},
		{
			FolderPath: "OtherMod",
			Manifest: Manifest{
				UniqueID:   "Author.Other",
				UpdateKeys: []string{"Nexus:999"},
			},
			UpdateStatus: UpdateStatus{State: "update_available", LatestVersion: "3.0.0"},
		},
	}

	ClearUpdateStatusAfterModUpdate(list, []string{"RedPandaBazaar"})

	must.Equal("current", list[0].UpdateStatus.State)
	must.Empty(list[0].UpdateStatus.LatestVersion)
	must.Equal("update_available", list[1].UpdateStatus.State)
}
