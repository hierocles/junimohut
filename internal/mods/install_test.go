package mods

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeTestZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	must := require.New(t)

	f, err := os.Create(path)
	must.NoError(err)
	defer f.Close()
	w := zip.NewWriter(f)
	for name, content := range files {
		fw, err := w.Create(name)
		must.NoError(err)
		_, err = fw.Write([]byte(content))
		must.NoError(err)
	}
	must.NoError(w.Close())
}

func TestInstallArchiveNestedManifests(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	manifest := func(uid string) string {
		return `{"Name":"` + uid + `","Author":"A","Version":"1.0.0","UniqueID":"` + uid + `","ContentPackFor":{"UniqueID":"PeacefulEnd.AlternativeTextures"}}`
	}
	archivePath := filepath.Join(t.TempDir(), "chest-deco.zip")
	writeTestZip(t, archivePath, map[string]string{
		"(AT) Chest Deco/manifest.json":                 manifest("Author.ChestDeco"),
		"(AT) Chest Deco/Textures/Barrel/manifest.json": manifest("Author.ChestDeco.Barrel"),
		"(AT) Chest Deco/Textures/Crate/manifest.json":  manifest("Author.ChestDeco.Crate"),
		"(AT) Chest Deco/Textures/Barrel/texture.json":  `{}`,
	})

	installer := NewInstaller(root)
	results, err := installer.InstallArchive(archivePath)
	must.NoError(err)
	must.Len(results, 1)
	must.Empty(results[0].Error)
	dest := filepath.Join(root, results[0].FolderPath)
	_, err = os.Stat(filepath.Join(dest, "Textures", "Barrel", "texture.json"))
	must.NoError(err)
	_, err = os.Stat(filepath.Join(dest, "Textures", "Barrel", "manifest.json"))
	must.NoError(err)
}

func TestUpdateModWithDeleteOld(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	modDir := filepath.Join(root, "TestMod")
	must.NoError(os.MkdirAll(modDir, 0o755))
	manifest := `{"Name":"Test Mod","Author":"A","Version":"1.0.0","UniqueID":"A.TestMod"}`
	must.NoError(os.WriteFile(filepath.Join(modDir, "manifest.json"), []byte(manifest), 0o644))
	must.NoError(os.WriteFile(filepath.Join(modDir, "old.dll"), []byte("old"), 0o644))
	must.NoError(os.WriteFile(filepath.Join(modDir, "config.json"), []byte(`{"key":"keep"}`), 0o644))

	archivePath := filepath.Join(t.TempDir(), "update.zip")
	writeTestZip(t, archivePath, map[string]string{
		"manifest.json": manifest,
		"new.dll":       "new content",
	})

	installer := NewInstaller(root)
	must.NoError(installer.UpdateMod("TestMod", archivePath, true))

	_, err := os.Stat(filepath.Join(modDir, "old.dll"))
	must.True(os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(modDir, "new.dll"))
	must.NoError(err)
	config, err := os.ReadFile(filepath.Join(modDir, "config.json"))
	must.NoError(err)
	must.True(strings.Contains(string(config), "keep"))
}

func TestUpdateModWithoutDeleteOld(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	modDir := filepath.Join(root, "TestMod")
	must.NoError(os.MkdirAll(modDir, 0o755))
	manifest := `{"Name":"Test Mod","Author":"A","Version":"1.0.0","UniqueID":"A.TestMod"}`
	must.NoError(os.WriteFile(filepath.Join(modDir, "manifest.json"), []byte(manifest), 0o644))
	must.NoError(os.WriteFile(filepath.Join(modDir, "old.dll"), []byte("old"), 0o644))

	archivePath := filepath.Join(t.TempDir(), "update.zip")
	writeTestZip(t, archivePath, map[string]string{
		"manifest.json": manifest,
		"new.dll":       "new content",
	})

	installer := NewInstaller(root)
	must.NoError(installer.UpdateMod("TestMod", archivePath, false))

	_, err := os.Stat(filepath.Join(modDir, "old.dll"))
	must.NoError(err)
	_, err = os.Stat(filepath.Join(modDir, "new.dll"))
	must.NoError(err)
}

func seasonalOpenWindowsManifest(uid string) string {
	return `{"Name":"[CP] Seasonal Open Windows","Author":"orangeblossom","Version":"1.2.1","UniqueID":"` + uid + `","UpdateKeys":["Nexus:20298"],"ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`
}

func TestInstallArchiveCollidingCPVariantNames(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "seasonal-open-windows.zip")
	variants := []struct {
		folder string
		uid    string
	}{
		{"[CP] Seasonal Open Windows", "OB7.SOWindows"},
		{"[CP] Seasonal Open Windows - BIRCH", "OB7.SOWindows.birch"},
		{"[CP] Seasonal Open Windows - BLACK", "OB7.SOWindows.black"},
		{"[CP] Seasonal Open Windows - BROWN", "OB7.SOWindows.brown"},
		{"[CP] Seasonal Open Windows - DARK BROWN", "OB7.SOWindows.darkbrown"},
	}
	files := map[string]string{}
	for _, v := range variants {
		files[v.folder+"/manifest.json"] = seasonalOpenWindowsManifest(v.uid)
		files[v.folder+"/content.json"] = `{}`
	}
	writeTestZip(t, archivePath, files)

	installer := NewInstaller(root)
	results, err := installer.InstallArchive(archivePath)
	must.NoError(err)
	must.Len(results, len(variants))

	seen := map[string]bool{}
	for _, r := range results {
		must.Empty(r.Error)
		seen[r.FolderPath] = true
		_, err := os.Stat(filepath.Join(root, r.FolderPath, "manifest.json"))
		must.NoError(err)
	}
	for _, v := range variants {
		must.True(seen[v.folder], "missing install folder %s", v.folder)
	}
	_, err = os.Stat(filepath.Join(root, "[CP] Seasonal Open Windows", "[CP] Seasonal Open Windows - BIRCH"))
	must.True(os.IsNotExist(err))
}

func TestInstallArchiveCollidingCPVariantNamesWrapped(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "seasonal-open-windows-wrapped.zip")
	files := map[string]string{
		"Seasonal Open Windows All-in-One/[CP] Seasonal Open Windows/manifest.json":             seasonalOpenWindowsManifest("OB7.SOWindows"),
		"Seasonal Open Windows All-in-One/[CP] Seasonal Open Windows - BIRCH/manifest.json":     seasonalOpenWindowsManifest("OB7.SOWindows.birch"),
		"Seasonal Open Windows All-in-One/[CP] Seasonal Open Windows - BLACK/manifest.json":     seasonalOpenWindowsManifest("OB7.SOWindows.black"),
	}
	writeTestZip(t, archivePath, files)

	installer := NewInstaller(root)
	results, err := installer.InstallArchive(archivePath)
	must.NoError(err)
	must.Len(results, 3)

	names := map[string]bool{}
	for _, r := range results {
		must.Empty(r.Error)
		names[r.FolderPath] = true
	}
	must.True(names["[CP] Seasonal Open Windows"])
	must.True(names["[CP] Seasonal Open Windows - BIRCH"])
	must.True(names["[CP] Seasonal Open Windows - BLACK"])
	_, err = os.Stat(filepath.Join(root, "Seasonal Open Windows All-in-One"))
	must.True(os.IsNotExist(err))
}
