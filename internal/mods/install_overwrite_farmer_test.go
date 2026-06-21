package mods

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func farmer20Manifest() string {
	return `{"Name":"Farmer 2.0 ESWF","Author":"Salty","Version":"1.8.4","UniqueID":"Salty.Farmer2.0","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`
}

func writeFarmer20Mod(t *testing.T, modsRoot string) Mod {
	t.Helper()
	must := require.New(t)

	modDir := filepath.Join(modsRoot, "Farmer 2.0 ESWF")
	must.NoError(os.MkdirAll(filepath.Join(modDir, "assets", "Player 1", "Desert"), 0o755))
	must.NoError(os.WriteFile(filepath.Join(modDir, "manifest.json"), []byte(farmer20Manifest()), 0o644))
	must.NoError(os.WriteFile(filepath.Join(modDir, "content.json"), []byte(`{"Format":"2.0.0","base":true}`), 0o644))
	must.NoError(os.WriteFile(filepath.Join(modDir, "assets", "background.png"), []byte("orig"), 0o644))
	must.NoError(os.WriteFile(filepath.Join(modDir, "assets", "Player 1", "Desert", "portrait.png"), []byte("orig"), 0o644))

	return Mod{
		ID:         ModID("Farmer 2.0 ESWF", "Salty.Farmer2.0"),
		FolderPath: "Farmer 2.0 ESWF",
		Manifest: Manifest{
			Name:     "Farmer 2.0 ESWF",
			UniqueID: "Salty.Farmer2.0",
		},
	}
}

func writeFarmer20UpdateZip(t *testing.T, path string) {
	t.Helper()
	writeTestZip(t, path, map[string]string{
		"Farmer 2.0 ESWF/manifest.json": farmer20Manifest(),
		"Farmer 2.0 ESWF/assets/background.png":                    "update",
		"Farmer 2.0 ESWF/assets/Player 1/Desert/portrait.png":      "update",
		"Farmer 2.0 ESWF/assets/Player 1/Desert/portrait_happy.png": "update",
	})
}

func TestPreviewInstallOverwritesFarmer20UpdateZip(t *testing.T) {
	must := require.New(t)

	modsRoot := t.TempDir()
	library := []Mod{writeFarmer20Mod(t, modsRoot)}

	archivePath := filepath.Join(t.TempDir(), "farmer-update.zip")
	writeFarmer20UpdateZip(t, archivePath)

	previews, err := PreviewInstallOverwrites([]string{archivePath}, modsRoot, library)
	must.NoError(err)
	must.Len(previews, 1)
	must.Equal("confirm", previews[0].State)
	must.Equal("Farmer 2.0 ESWF", previews[0].SuggestedTarget)
	must.NotEmpty(previews[0].Candidates)
}

func TestMergeArchiveIntoModFarmer20UpdateZip(t *testing.T) {
	must := require.New(t)

	modsRoot := t.TempDir()
	writeFarmer20Mod(t, modsRoot)

	archivePath := filepath.Join(t.TempDir(), "farmer-update.zip")
	writeFarmer20UpdateZip(t, archivePath)

	installer := NewInstaller(modsRoot)
	result, err := installer.MergeArchiveIntoMod(archivePath, "Farmer 2.0 ESWF")
	must.NoError(err)
	must.Contains(result.ModID, "Salty.Farmer2.0")

	patched, err := os.ReadFile(filepath.Join(modsRoot, "Farmer 2.0 ESWF", "assets", "background.png"))
	must.NoError(err)
	must.Equal("update", string(patched))
}

func TestPreviewInstallOverwritesFarmer20UpdateZipIntegration(t *testing.T) {
	archivePath := filepath.FromSlash("C:/Users/dylan/AppData/Roaming/JunimoHut/downloads/Farmer 2.0 ESWF-21226-1-8-4-1733088403.zip")
	if _, err := os.Stat(archivePath); err != nil {
		t.Skip("Farmer 2.0 update zip not available locally")
	}

	modsRoot := filepath.FromSlash("E:/SDV_MODS")
	if _, err := os.Stat(modsRoot); err != nil {
		t.Skip("user mods root not available")
	}

	scanner := NewScanner()
	library, err := scanner.Scan(ScanOptions{ModsRoot: modsRoot})
	require.NoError(t, err)

	previews, err := PreviewInstallOverwrites([]string{archivePath}, modsRoot, library)
	require.NoError(t, err)
	require.Len(t, previews, 1)
	require.Equal(t, "confirm", previews[0].State)
	require.Equal(t, "Farmer 2.0 ESWF", previews[0].SuggestedTarget)
}

func TestPreviewInstallOverwritesFernPresetIntegration(t *testing.T) {
	archivePath := filepath.FromSlash("C:/Users/dylan/AppData/Roaming/JunimoHut/downloads/Main File 1.1.0-39037-1-1-0-1764354745.7z")
	if _, err := os.Stat(archivePath); err != nil {
		t.Skip("Fern preset archive not available locally")
	}

	modsRoot := filepath.FromSlash("E:/SDV_MODS")
	if _, err := os.Stat(modsRoot); err != nil {
		t.Skip("user mods root not available")
	}

	scanner := NewScanner()
	library, err := scanner.Scan(ScanOptions{ModsRoot: modsRoot})
	require.NoError(t, err)

	previews, err := PreviewInstallOverwrites([]string{archivePath}, modsRoot, library)
	require.NoError(t, err)
	require.Len(t, previews, 1)
	require.Equal(t, "confirm", previews[0].State)
	require.Equal(t, "Farmer 2.0 ESWF", previews[0].SuggestedTarget)
}
