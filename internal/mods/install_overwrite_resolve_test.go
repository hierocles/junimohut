package mods

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLookupOverwriteTargetsNormalizesPaths(t *testing.T) {
	must := require.New(t)

	archive := `C:\Users\dylan\Downloads\patch.7z`
	targets := map[string][]string{
		`C:/Users/dylan/Downloads/patch.7z`: {"GenericModConfigMenu"},
	}

	got := LookupOverwriteTargets(targets, archive)
	must.Equal([]string{"GenericModConfigMenu"}, got)
}

func TestResolveInstallMergeTargetsRequiresExplicitSelection(t *testing.T) {
	must := require.New(t)

	modsRoot := t.TempDir()
	gmc := writeSimpleModWithAsset(t, modsRoot, "GenericModConfigMenu", "spacechase0.GenericModConfigMenu", "Generic Mod Config Menu", "assets/config-button.png")
	must.NoError(writeSimpleModAsset(t, modsRoot, "GenericModConfigMenu", "assets/keybinds-button.png"))
	library := []Mod{
		gmc,
		writeSimpleModWithAsset(t, modsRoot, "EventLookup", "shekurika.EventLookup", "Event Lookup", "assets/button.png"),
	}

	archivePath := filepath.Join(t.TempDir(), "assorted.zip")
	writeAssortedAssetsZip(t, archivePath)

	targets, err := ResolveInstallMergeTargets(archivePath, nil, modsRoot, library)
	must.NoError(err)
	must.Empty(targets)

	targets, err = ResolveInstallMergeTargets(archivePath, map[string][]string{
		archivePath: {"GenericModConfigMenu", "EventLookup"},
	}, modsRoot, library)
	must.NoError(err)
	must.Equal([]string{"GenericModConfigMenu", "EventLookup"}, targets)
}

func writeSimpleModAsset(t *testing.T, modsRoot, folder, assetRel string) error {
	t.Helper()
	must := require.New(t)
	assetPath := filepath.Join(modsRoot, filepath.FromSlash(folder), filepath.FromSlash(assetRel))
	must.NoError(os.MkdirAll(filepath.Dir(assetPath), 0o755))
	return os.WriteFile(assetPath, []byte("orig"), 0o644)
}
