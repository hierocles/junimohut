package mods

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeAssortedAssetsZip(t *testing.T, path string) {
	t.Helper()
	writeTestZip(t, path, map[string]string{
		"GenericModConfigMenu/assets/config-button.png":  "gmc",
		"GenericModConfigMenu/assets/keybinds-button.png": "gmc",
		"EventLookup/assets/button.png":                  "el",
		"EventLookup/assets/statusCover.png":             "el",
		"FerngillSimpleEconomy/FerngillSimpleEconomy/assets/stock-menu.png": "fse",
		"BetterJukebox/assets/BetterJukeboxGraphics.png": "bj",
	})
}

func writeSimpleModWithAsset(t *testing.T, modsRoot, folder, uid, name, assetRel string) Mod {
	t.Helper()
	must := require.New(t)

	modDir := filepath.Join(modsRoot, filepath.FromSlash(folder))
	assetPath := filepath.Join(modDir, filepath.FromSlash(assetRel))
	must.NoError(os.MkdirAll(filepath.Dir(assetPath), 0o755))
	must.NoError(os.WriteFile(assetPath, []byte("orig"), 0o644))
	manifestPath := filepath.Join(modDir, "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		manifest := `{"Name":"` + name + `","Author":"A","Version":"1.0.0","UniqueID":"` + uid + `"}`
		must.NoError(os.WriteFile(manifestPath, []byte(manifest), 0o644))
	}

	return Mod{
		ID:         ModID(folder, uid),
		FolderPath: folder,
		Manifest: Manifest{
			Name:     name,
			UniqueID: uid,
		},
	}
}

func TestPreviewInstallOverwritesAssortedAssetsPack(t *testing.T) {
	must := require.New(t)

	modsRoot := t.TempDir()
	gmc := writeSimpleModWithAsset(t, modsRoot, "GenericModConfigMenu", "spacechase0.GenericModConfigMenu", "Generic Mod Config Menu", "assets/config-button.png")
	must.NoError(os.WriteFile(
		filepath.Join(modsRoot, "GenericModConfigMenu", "assets", "keybinds-button.png"),
		[]byte("orig"),
		0o644,
	))
	library := []Mod{
		gmc,
		writeSimpleModWithAsset(t, modsRoot, "EventLookup", "shekurika.EventLookup", "Event Lookup", "assets/button.png"),
		writeSimpleModWithAsset(t, modsRoot, "EventLookup", "shekurika.EventLookup", "Event Lookup", "assets/statusCover.png"),
		writeSimpleModWithAsset(t, modsRoot, "FerngillSimpleEconomy/FerngillSimpleEconomy", "furyx639.FerngillSimpleEconomy", "Ferngill Simple Economy", "assets/stock-menu.png"),
	}

	archivePath := filepath.Join(t.TempDir(), "assorted.zip")
	writeAssortedAssetsZip(t, archivePath)

	previews, err := PreviewInstallOverwrites([]string{archivePath}, modsRoot, library)
	must.NoError(err)
	must.Len(previews, 1)
	must.Equal("confirm", previews[0].State)
	must.GreaterOrEqual(len(previews[0].Candidates), 3)

	names := map[string]bool{}
	for _, c := range previews[0].Candidates {
		names[c.ModName] = true
	}
	must.True(names["Generic Mod Config Menu"])
	must.True(names["Event Lookup"])
	must.True(names["Ferngill Simple Economy"])
}

func TestMergeArchiveIntoModAssortedAssetsPackSingleTarget(t *testing.T) {
	must := require.New(t)

	modsRoot := t.TempDir()
	writeSimpleModWithAsset(t, modsRoot, "GenericModConfigMenu", "spacechase0.GenericModConfigMenu", "Generic Mod Config Menu", "assets/config-button.png")
	writeSimpleModWithAsset(t, modsRoot, "EventLookup", "shekurika.EventLookup", "Event Lookup", "assets/button.png")

	archivePath := filepath.Join(t.TempDir(), "assorted.zip")
	writeAssortedAssetsZip(t, archivePath)

	installer := NewInstaller(modsRoot)
	_, err := installer.MergeArchiveIntoMod(archivePath, "GenericModConfigMenu")
	must.NoError(err)

	patched, err := os.ReadFile(filepath.Join(modsRoot, "GenericModConfigMenu", "assets", "config-button.png"))
	must.NoError(err)
	must.Equal("gmc", string(patched))

	untouched, err := os.ReadFile(filepath.Join(modsRoot, "EventLookup", "assets", "button.png"))
	must.NoError(err)
	must.Equal("orig", string(untouched))
}
