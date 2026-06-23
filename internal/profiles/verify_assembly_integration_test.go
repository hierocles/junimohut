package profiles

import (
	"os"
	"path/filepath"
	"testing"

	"junimohut/internal/config"
	"junimohut/internal/mods"
	"junimohut/internal/platform"

	"github.com/stretchr/testify/require"
)

// TestVerifyAssemblyIntegration runs assembly against the local Junimo Hut config.
// Set JUNIMOHUT_VERIFY=1 to run (uses real game/library paths from AppData config).
func TestVerifyAssemblyIntegration(t *testing.T) {
	if os.Getenv("JUNIMOHUT_VERIFY") != "1" {
		t.Skip("set JUNIMOHUT_VERIFY=1 to run against local config")
	}
	must := require.New(t)

	store, err := config.NewStore()
	must.NoError(err)
	settings := store.Get()
	must.NotEmpty(settings.ModsRoot)
	must.NotEmpty(settings.GamePath)

	profSvc, err := NewService(store.ProfilesDir())
	must.NoError(err)
	enabled := profSvc.EnabledMods()

	list, err := mods.NewScanner().Scan(mods.ScanOptions{
		ModsRoot:            settings.ModsRoot,
		IgnoreHiddenFolders: settings.IgnoreHiddenFolders,
	})
	must.NoError(err)
	list = mods.DedupeByUniqueID(mods.DedupeByID(list))

	active := config.ActiveModsDir(settings.GamePath)
	asm := NewAssembler(active, settings.ModsRoot)
	must.NoError(asm.Assemble(list, enabled))

	checks := []string{
		"Downtown-Zuzu-main/[BL] Downtown Zuzu",
		"Downtown-Zuzu-main/[CC] Downtown Zuzu",
		"Downtown-Zuzu-main/[CP] Downtown Zuzu",
		"Downtown-Zuzu-main/[DLL] Downtown Zuzu",
		"Downtown-Zuzu-main/[FTM] Downtown Zuzu",
		"Downtown-Zuzu-main/[MFM] Downtown Zuzu",
		"Downtown-Zuzu-main/[TS] Downtown Zuzu",
		"Stardew Valley Expanded/[CP] Stardew Valley Expanded",
		"Stardew Valley Expanded/[FTM] Stardew Valley Expanded",
		"Stardew Valley Expanded/Stardew Valley Expanded Code",
		"Sunberry Village/[CP] Sunberry Village",
		"Sunberry Village/[CC] Sunberry Village",
		"Sunberry Village/[C#] Sunberry Village",
		"Sunberry Village/[FTM] Sunberry Village",
	}
	for _, rel := range checks {
		linkPath := filepath.Join(active, filepath.FromSlash(rel))
		must.True(platform.IsManagedModLink(linkPath, settings.ModsRoot), "missing junction: %s", rel)
	}

	// Re-assemble must not tear down valid nested junctions.
	must.NoError(asm.Assemble(list, enabled))
	for _, rel := range checks {
		linkPath := filepath.Join(active, filepath.FromSlash(rel))
		must.True(platform.IsManagedModLink(linkPath, settings.ModsRoot), "junction removed after re-assemble: %s", rel)
	}
}
