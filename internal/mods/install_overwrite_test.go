package mods

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func fashionSenseManifest() string {
	return `{"Name":"Fashion Sense","Author":"PeacefulEnd","Version":"7.0.0","UniqueID":"PeacefulEnd.FashionSense","EntryDll":"FashionSense.dll"}`
}

func writeFashionSenseMod(t *testing.T, modsRoot, folderName string) Mod {
	t.Helper()
	must := require.New(t)

	modDir := filepath.Join(modsRoot, folderName)
	uiDir := filepath.Join(modDir, "FashionSense", "Framework", "Assets", "UI")
	must.NoError(os.MkdirAll(uiDir, 0o755))
	must.NoError(os.WriteFile(filepath.Join(modDir, "FashionSense", "manifest.json"), []byte(fashionSenseManifest()), 0o644))
	must.NoError(os.WriteFile(filepath.Join(uiDir, "HairButton.png"), []byte("orig"), 0o644))
	must.NoError(os.WriteFile(filepath.Join(uiDir, "HatButton.png"), []byte("orig"), 0o644))

	return Mod{
		ID:         ModID(folderName, "PeacefulEnd.FashionSense"),
		FolderPath: filepath.Join(folderName, "FashionSense"),
		Manifest: Manifest{
			Name:     "Fashion Sense",
			UniqueID: "PeacefulEnd.FashionSense",
		},
	}
}

func writeFlatFashionSenseMod(t *testing.T, modsRoot string) Mod {
	t.Helper()
	must := require.New(t)

	modDir := filepath.Join(modsRoot, "FashionSense")
	uiDir := filepath.Join(modDir, "Framework", "Assets", "UI")
	must.NoError(os.MkdirAll(uiDir, 0o755))
	must.NoError(os.WriteFile(filepath.Join(modDir, "manifest.json"), []byte(fashionSenseManifest()), 0o644))
	must.NoError(os.WriteFile(filepath.Join(uiDir, "HairButton.png"), []byte("orig"), 0o644))
	must.NoError(os.WriteFile(filepath.Join(uiDir, "HatButton.png"), []byte("orig"), 0o644))

	return Mod{
		ID:         ModID("FashionSense", "PeacefulEnd.FashionSense"),
		FolderPath: "FashionSense",
		Manifest: Manifest{
			Name:     "Fashion Sense",
			UniqueID: "PeacefulEnd.FashionSense",
		},
	}
}

func writeFashionSenseUIPatchZip(t *testing.T, path string) {
	t.Helper()
	writeTestZip(t, path, map[string]string{
		"FashionSense/Framework/Assets/UI/HairButton.png":  "patch",
		"FashionSense/Framework/Assets/UI/HatButton.png":   "patch",
		"FashionSense/Framework/Assets/UI/ShirtButton.png": "patch",
	})
}

func TestPreviewInstallOverwritesFashionSenseUI(t *testing.T) {
	must := require.New(t)

	modsRoot := t.TempDir()
	library := []Mod{writeFashionSenseMod(t, modsRoot, "Fashion Sense")}

	archivePath := filepath.Join(t.TempDir(), "fs-ui.zip")
	writeFashionSenseUIPatchZip(t, archivePath)

	previews, err := PreviewInstallOverwrites([]string{archivePath}, modsRoot, library)
	must.NoError(err)
	must.Len(previews, 1)
	must.Equal("confirm", previews[0].State)
	must.Equal("Fashion Sense", previews[0].SuggestedTarget)
	must.Len(previews[0].Candidates, 1)
	must.Contains(previews[0].Candidates[0].FolderPath, "Fashion Sense")
	must.Equal(2, previews[0].Candidates[0].MatchedFiles)
}

func TestPreviewInstallOverwritesFlatFashionSenseFolder(t *testing.T) {
	must := require.New(t)

	modsRoot := t.TempDir()
	library := []Mod{writeFlatFashionSenseMod(t, modsRoot)}

	archivePath := filepath.Join(t.TempDir(), "fs-ui.zip")
	writeFashionSenseUIPatchZip(t, archivePath)

	previews, err := PreviewInstallOverwrites([]string{archivePath}, modsRoot, library)
	must.NoError(err)
	must.Len(previews, 1)
	must.Equal("confirm", previews[0].State)
	must.Equal("FashionSense", previews[0].SuggestedTarget)
	must.Equal(2, previews[0].Candidates[0].MatchedFiles)
}

func TestMergeArchiveIntoModFlatFashionSenseFolder(t *testing.T) {
	must := require.New(t)

	modsRoot := t.TempDir()
	writeFlatFashionSenseMod(t, modsRoot)

	archivePath := filepath.Join(t.TempDir(), "fs-ui.zip")
	writeFashionSenseUIPatchZip(t, archivePath)

	installer := NewInstaller(modsRoot)
	result, err := installer.MergeArchiveIntoMod(archivePath, "FashionSense")
	must.NoError(err)
	must.Contains(result.ModID, "PeacefulEnd.FashionSense")

	patched, err := os.ReadFile(filepath.Join(modsRoot, "FashionSense", "Framework", "Assets", "UI", "HairButton.png"))
	must.NoError(err)
	must.Equal("patch", string(patched))
}

func TestPreviewInstallOverwritesBlockedWithoutTarget(t *testing.T) {
	must := require.New(t)

	modsRoot := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "fs-ui.zip")
	writeFashionSenseUIPatchZip(t, archivePath)

	previews, err := PreviewInstallOverwrites([]string{archivePath}, modsRoot, nil)
	must.NoError(err)
	must.Len(previews, 1)
	must.Equal("blocked", previews[0].State)
	must.NotEmpty(previews[0].BlockReason)
	must.Contains(previews[0].BlockReason, "Fashion Sense")
}

func TestPreviewInstallOverwritesSkipsManifestArchive(t *testing.T) {
	must := require.New(t)

	modsRoot := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "mod.zip")
	writeTestZip(t, archivePath, map[string]string{
		"manifest.json": `{"Name":"Test","Author":"A","Version":"1.0.0","UniqueID":"A.Test"}`,
	})

	previews, err := PreviewInstallOverwrites([]string{archivePath}, modsRoot, nil)
	must.NoError(err)
	must.Empty(previews)
}

func TestPreviewInstallOverwritesMultipleCandidates(t *testing.T) {
	must := require.New(t)

	modsRoot := t.TempDir()
	library := []Mod{
		writeFashionSenseMod(t, modsRoot, "Fashion Sense A"),
		writeFashionSenseMod(t, modsRoot, "Fashion Sense B"),
	}

	archivePath := filepath.Join(t.TempDir(), "fs-ui.zip")
	writeFashionSenseUIPatchZip(t, archivePath)

	previews, err := PreviewInstallOverwrites([]string{archivePath}, modsRoot, library)
	must.NoError(err)
	must.Len(previews, 1)
	must.Equal("confirm", previews[0].State)
	must.Len(previews[0].Candidates, 2)
	must.NotEmpty(previews[0].SuggestedTarget)
}

func TestMergeArchiveIntoModFashionSenseUI(t *testing.T) {
	must := require.New(t)

	modsRoot := t.TempDir()
	writeFashionSenseMod(t, modsRoot, "Fashion Sense")

	archivePath := filepath.Join(t.TempDir(), "fs-ui.zip")
	writeFashionSenseUIPatchZip(t, archivePath)

	installer := NewInstaller(modsRoot)
	result, err := installer.MergeArchiveIntoMod(archivePath, "Fashion Sense")
	must.NoError(err)
	must.Empty(result.Error)
	must.Contains(result.ModID, "PeacefulEnd.FashionSense")
	must.Equal("Fashion Sense/FashionSense", result.FolderPath)

	patched, err := os.ReadFile(filepath.Join(modsRoot, "Fashion Sense", "FashionSense", "Framework", "Assets", "UI", "HairButton.png"))
	must.NoError(err)
	must.Equal("patch", string(patched))
}

func TestPreviewInstallDependenciesSkipsOverwritePatch(t *testing.T) {
	must := require.New(t)

	modsRoot := t.TempDir()
	library := []Mod{writeFashionSenseMod(t, modsRoot, "Fashion Sense")}
	archivePath := filepath.Join(t.TempDir(), "fs-ui.zip")
	writeFashionSenseUIPatchZip(t, archivePath)

	previews, err := PreviewInstallDependencies([]string{archivePath}, library)
	must.NoError(err)
	must.Empty(previews)
}

func TestPreviewInstallNamesSkipsOverwritePatch(t *testing.T) {
	must := require.New(t)

	archivePath := filepath.Join(t.TempDir(), "fs-ui.zip")
	writeFashionSenseUIPatchZip(t, archivePath)

	previews, err := PreviewInstallNames([]string{archivePath})
	must.NoError(err)
	must.Empty(previews)
}

func writeFarmer20PresetZip(t *testing.T, path string) {
	t.Helper()
	writeTestZip(t, path, map[string]string{
		"FernPreset/content.json":                          `{"Format":"2.0.0"}`,
		"FernPreset/assets/Fern/Desert/portrait.png":       "preset",
		"FernPreset/assets/Fern/Desert/portrait_happy.png":   "preset",
	})
}

func TestPreviewInstallOverwritesFarmer20PresetPack(t *testing.T) {
	must := require.New(t)

	modsRoot := t.TempDir()
	library := []Mod{writeFarmer20Mod(t, modsRoot)}

	archivePath := filepath.Join(t.TempDir(), "fern-preset.zip")
	writeFarmer20PresetZip(t, archivePath)

	previews, err := PreviewInstallOverwrites([]string{archivePath}, modsRoot, library)
	must.NoError(err)
	must.Len(previews, 1)
	must.Equal("confirm", previews[0].State)
	must.Equal("Farmer 2.0 ESWF", previews[0].SuggestedTarget)
}

func TestMergeArchiveIntoModFarmer20PresetPack(t *testing.T) {
	must := require.New(t)

	modsRoot := t.TempDir()
	writeFarmer20Mod(t, modsRoot)

	archivePath := filepath.Join(t.TempDir(), "fern-preset.zip")
	writeFarmer20PresetZip(t, archivePath)

	installer := NewInstaller(modsRoot)
	result, err := installer.MergeArchiveIntoMod(archivePath, "Farmer 2.0 ESWF")
	must.NoError(err)
	must.Contains(result.ModID, "Salty.Farmer2.0")

	content, err := os.ReadFile(filepath.Join(modsRoot, "Farmer 2.0 ESWF", "content.json"))
	must.NoError(err)
	must.Equal(`{"Format":"2.0.0"}`, string(content))

	patched, err := os.ReadFile(filepath.Join(modsRoot, "Farmer 2.0 ESWF", "assets", "Fern", "Desert", "portrait.png"))
	must.NoError(err)
	must.Equal("preset", string(patched))

	_, err = os.Stat(filepath.Join(modsRoot, "Farmer 2.0 ESWF", "FernPreset"))
	must.True(os.IsNotExist(err))
}
