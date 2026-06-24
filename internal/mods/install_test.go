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

func TestUpdateModBumpsManifestVersion(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	modDir := filepath.Join(root, "TestMod")
	must.NoError(os.MkdirAll(modDir, 0o755))
	oldManifest := `{"Name":"Test Mod","Author":"A","Version":"1.0.0","UniqueID":"A.TestMod"}`
	must.NoError(os.WriteFile(filepath.Join(modDir, "manifest.json"), []byte(oldManifest), 0o644))

	archivePath := filepath.Join(t.TempDir(), "update.zip")
	newManifest := `{"Name":"Test Mod","Author":"A","Version":"2.0.0","UniqueID":"A.TestMod"}`
	writeTestZip(t, archivePath, map[string]string{
		"manifest.json": newManifest,
		"new.dll":       "new content",
	})

	installer := NewInstaller(root)
	must.NoError(installer.UpdateMod("TestMod", archivePath, true))

	onDisk, err := os.ReadFile(filepath.Join(modDir, "manifest.json"))
	must.NoError(err)
	parsed, err := ParseManifest(filepath.Join(modDir, "manifest.json"))
	must.NoError(err)
	must.Contains(string(onDisk), `"Version":"2.0.0"`)
	must.Equal("2.0.0", parsed.Version)
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

func TestInstallArchiveReinstallSameUniqueID(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	manifest := `{"Name":"Convenient Inventory","Author":"A","Version":"1.0.0","UniqueID":"Author.ConvenientInventory","EntryDll":"Mod.dll"}`
	modDir := filepath.Join(root, "ConvenientInventory")
	must.NoError(os.MkdirAll(modDir, 0o755))
	must.NoError(os.WriteFile(filepath.Join(modDir, "manifest.json"), []byte(manifest), 0o644))
	must.NoError(os.WriteFile(filepath.Join(modDir, "Mod.dll"), []byte("v1"), 0o644))

	archivePath := filepath.Join(t.TempDir(), "convenient-inventory.zip")
	writeTestZip(t, archivePath, map[string]string{
		"ConvenientInventory/manifest.json": manifest,
		"ConvenientInventory/Mod.dll":        "v2",
	})

	installer := NewInstaller(root)
	results, err := installer.InstallArchive(archivePath)
	must.NoError(err)
	must.Len(results, 1)
	must.NotEmpty(results[0].Error)
	must.Contains(results[0].Error, "merge target")

	entries, err := os.ReadDir(root)
	must.NoError(err)
	must.Len(entries, 1)

	data, err := os.ReadFile(filepath.Join(root, "ConvenientInventory", "Mod.dll"))
	must.NoError(err)
	must.Equal("v1", string(data))
}

func TestInstallArchiveFolderCollisionDifferentUniqueID(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	existing := `{"Name":"Shared Name","Author":"A","Version":"1.0.0","UniqueID":"Author.ExistingMod"}`
	modDir := filepath.Join(root, "Shared Name")
	must.NoError(os.MkdirAll(modDir, 0o755))
	must.NoError(os.WriteFile(filepath.Join(modDir, "manifest.json"), []byte(existing), 0o644))

	incoming := `{"Name":"Shared Name","Author":"B","Version":"1.0.0","UniqueID":"Author.OtherMod"}`
	archivePath := filepath.Join(t.TempDir(), "other-mod.zip")
	writeTestZip(t, archivePath, map[string]string{
		"Shared Name/manifest.json": incoming,
	})

	installer := NewInstaller(root)
	results, err := installer.InstallArchive(archivePath)
	must.NoError(err)
	must.Len(results, 1)
	must.NotEmpty(results[0].Error)
	must.Contains(results[0].Error, "Shared Name")

	entries, err := os.ReadDir(root)
	must.NoError(err)
	must.Len(entries, 1)
}

func TestFindInstalledModsByUniqueID(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	writeSimpleModWithAsset(t, root, "ConvenientInventory", "Author.ConvenientInventory", "Convenient Inventory", "Mod.dll")
	writeSimpleModWithAsset(t, root, "ConvenientInventory_20260620_165612", "Author.ConvenientInventory", "Convenient Inventory", "Mod.dll")

	found, err := FindInstalledModsByUniqueID(root, "Author.ConvenientInventory")
	must.NoError(err)
	must.Len(found, 2)
}

func TestInstallArchiveNestedWrapperFolder(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	manifest := `{"Name":"Fruit Tree Bug Fix","Author":"moonslime","Version":"1.0.0","UniqueID":"moonslime.FruitTreeBugFix","EntryDll":"FruitTreeBugFix.dll"}`
	archivePath := filepath.Join(t.TempDir(), "fruit-tree-bug-fix.zip")
	writeTestZip(t, archivePath, map[string]string{
		"FruitTreeBugFix/FruitTreeBugFix/manifest.json": manifest,
		"FruitTreeBugFix/FruitTreeBugFix/FruitTreeBugFix.dll": "dll",
	})

	previews, err := PreviewInstallNames([]string{archivePath})
	must.NoError(err)
	must.Len(previews, 1)
	must.Len(previews[0].Mods, 1)
	must.Equal("Fruit Tree Bug Fix", previews[0].Mods[0].OfficialName)
	must.Equal("Fruit Tree Bug Fix", previews[0].Mods[0].FolderLabel)
	must.Equal("moonslime.FruitTreeBugFix", previews[0].Mods[0].UniqueID)

	installer := NewInstaller(root)
	results, err := installer.InstallArchive(archivePath)
	must.NoError(err)
	must.Len(results, 1)
	must.Empty(results[0].Error)
	must.Equal("Fruit Tree Bug Fix", results[0].FolderPath)

	_, err = os.Stat(filepath.Join(root, "Fruit Tree Bug Fix", "manifest.json"))
	must.NoError(err)
}
