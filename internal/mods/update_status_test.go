package mods

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeUpdateState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{"update", "update_available"},
		{"ok", "current"},
		{"incompatible", "incompatible"},
		{"update_ignored", "update_ignored"},
		{"unofficial", "unofficial"},
		{"update_available", "update_available"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			must := require.New(t)
			must.Equal(tt.want, NormalizeUpdateState(tt.in))
		})
	}
}

func TestPreserveUpdateStatus(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	previous := []Mod{
		{
			ID:       "ModA::Author.ModA",
			Manifest: Manifest{Version: "1.0.0"},
			UpdateStatus: UpdateStatus{
				State:         "update_available",
				LatestVersion: "2.0.0",
			},
		},
		{
			ID:       "ModB::Author.ModB",
			Manifest: Manifest{Version: "1.0.0"},
			UpdateStatus: UpdateStatus{State: "current"},
		},
	}
	list := []Mod{
		{ID: "ModA::Author.ModA", Manifest: Manifest{Version: "1.0.0"}},
		{ID: "ModB::Author.ModB", Manifest: Manifest{Version: "2.0.0"}},
		{ID: "ModC::Author.ModC", Manifest: Manifest{Version: "1.0.0"}},
	}

	PreserveUpdateStatus(list, previous)

	must.Equal("update_available", list[0].UpdateStatus.State)
	must.Equal("2.0.0", list[0].UpdateStatus.LatestVersion)
	must.Empty(list[1].UpdateStatus.State)
	must.Empty(list[2].UpdateStatus.State)
}

func TestApplyCachedUpdateStatus(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	list := []Mod{
		{ID: "ModA::Author.ModA", Manifest: Manifest{Version: "1.0.0"}},
		{ID: "ModB::Author.ModB", Manifest: Manifest{Version: "2.0.0"}},
	}
	cached := map[string]CachedUpdate{
		"ModA::Author.ModA": {
			ManifestVersion: "1.0.0",
			State:           "update_available",
			LatestVersion:   "2.0.0",
		},
		"ModB::Author.ModB": {
			ManifestVersion: "1.0.0",
			State:           "update_available",
			LatestVersion:   "3.0.0",
		},
	}

	ApplyCachedUpdateStatus(list, cached)

	must.Equal("update_available", list[0].UpdateStatus.State)
	must.Equal("2.0.0", list[0].UpdateStatus.LatestVersion)
	must.Empty(list[1].UpdateStatus.State)
}
