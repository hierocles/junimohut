package mods_test

import (
	"os"
	"path/filepath"
	"testing"

	"junimohut/internal/mods"
	"junimohut/internal/nexus"

	"github.com/stretchr/testify/require"
)

func writeTestMod(t *testing.T, modsRoot, folder, uniqueID string) {
	t.Helper()
	must := require.New(t)

	modDir := filepath.Join(modsRoot, folder)
	must.NoError(os.MkdirAll(modDir, 0o755))
	manifest := `{"Name":"` + folder + `","Author":"A","Version":"1.0.0","UniqueID":"` + uniqueID + `"}`
	must.NoError(os.WriteFile(filepath.Join(modDir, "manifest.json"), []byte(manifest), 0o644))
}

func modResolver(cache []mods.Mod) mods.ModResolver {
	return func(folderPath string) (mods.Mod, bool) {
		for _, mod := range cache {
			if mod.FolderPath == folderPath {
				return mod, true
			}
		}
		return mods.Mod{}, false
	}
}

func setupDeleteTest(t *testing.T) (modsRoot string, downloadsDir string, index *nexus.DownloadIndex) {
	t.Helper()
	must := require.New(t)

	dataDir := t.TempDir()
	modsRoot = filepath.Join(dataDir, "mod-library")
	downloadsDir = filepath.Join(dataDir, "downloads")
	must.NoError(os.MkdirAll(modsRoot, 0o755))

	var err error
	index, err = nexus.NewDownloadIndex(dataDir, downloadsDir)
	must.NoError(err)
	return modsRoot, downloadsDir, index
}

func TestDeleteModsBulk(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	modsRoot, _, index := setupDeleteTest(t)
	installer := mods.NewInstaller(modsRoot)

	writeTestMod(t, modsRoot, "ModA", "Author.ModA")
	writeTestMod(t, modsRoot, "ModB", "Author.ModB")

	result := mods.DeleteMods(
		installer,
		[]string{"ModA", "ModB"},
		false,
		modResolver([]mods.Mod{
			{ID: "ModA", FolderPath: "ModA", Manifest: mods.Manifest{UniqueID: "Author.ModA"}},
			{ID: "ModB", FolderPath: "ModB", Manifest: mods.Manifest{UniqueID: "Author.ModB"}},
		}),
		index,
		nexus.ModIDFromUpdateKeys,
	)
	must.Equal(2, result.DeletedCount)
	must.Empty(result.Errors)
	_, err := os.Stat(filepath.Join(modsRoot, "ModA"))
	must.True(os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(modsRoot, "ModB"))
	must.True(os.IsNotExist(err))
}

func TestDeleteModsWithArchive(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	modsRoot, downloadsDir, index := setupDeleteTest(t)
	installer := mods.NewInstaller(modsRoot)

	writeTestMod(t, modsRoot, "CoolMod", "Author.CoolMod")

	archive := filepath.Join(downloadsDir, "CoolMod.zip")
	must.NoError(os.WriteFile(archive, []byte("zip"), 0o644))
	must.NoError(index.Record(nexus.DownloadRecord{
		ArchivePath: archive,
		UniqueID:    "Author.CoolMod",
		FileName:    "CoolMod.zip",
	}))

	result := mods.DeleteMods(
		installer,
		[]string{"CoolMod"},
		true,
		modResolver([]mods.Mod{
			{ID: "CoolMod", FolderPath: "CoolMod", Manifest: mods.Manifest{UniqueID: "Author.CoolMod"}},
		}),
		index,
		nexus.ModIDFromUpdateKeys,
	)
	must.Equal(1, result.DeletedCount)
	must.Equal(1, result.ArchivesDeletedCount)
	_, err := os.Stat(archive)
	must.True(os.IsNotExist(err))
	_, ok := index.FindForMod("Author.CoolMod", 0)
	must.False(ok)
}

func TestDeleteModsLeavesArchiveWhenNotRequested(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	modsRoot, downloadsDir, index := setupDeleteTest(t)
	installer := mods.NewInstaller(modsRoot)

	writeTestMod(t, modsRoot, "KeepArchive", "Author.KeepArchive")

	archive := filepath.Join(downloadsDir, "KeepArchive.zip")
	must.NoError(os.WriteFile(archive, []byte("zip"), 0o644))
	must.NoError(index.Record(nexus.DownloadRecord{
		ArchivePath: archive,
		UniqueID:    "Author.KeepArchive",
	}))

	result := mods.DeleteMods(
		installer,
		[]string{"KeepArchive"},
		false,
		modResolver([]mods.Mod{
			{ID: "KeepArchive", FolderPath: "KeepArchive", Manifest: mods.Manifest{UniqueID: "Author.KeepArchive"}},
		}),
		index,
		nexus.ModIDFromUpdateKeys,
	)
	must.Equal(0, result.ArchivesDeletedCount)
	_, err := os.Stat(archive)
	must.NoError(err)
}

func TestDeleteModsArchiveDeleteFailure(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	modsRoot, _, index := setupDeleteTest(t)
	installer := mods.NewInstaller(modsRoot)
	dataDir := filepath.Dir(modsRoot)

	writeTestMod(t, modsRoot, "ModA", "Author.ModA")

	outsideArchive := filepath.Join(dataDir, "outside.zip")
	must.NoError(os.WriteFile(outsideArchive, []byte("zip"), 0o644))
	must.NoError(index.Record(nexus.DownloadRecord{
		ArchivePath: outsideArchive,
		UniqueID:    "Author.ModA",
	}))

	result := mods.DeleteMods(
		installer,
		[]string{"ModA"},
		true,
		modResolver([]mods.Mod{
			{ID: "ModA", FolderPath: "ModA", Manifest: mods.Manifest{UniqueID: "Author.ModA"}},
		}),
		index,
		nexus.ModIDFromUpdateKeys,
	)
	must.Equal(1, result.DeletedCount)
	must.Equal(0, result.ArchivesDeletedCount)
	must.NotEmpty(result.Errors)
	_, err := os.Stat(filepath.Join(modsRoot, "ModA"))
	must.True(os.IsNotExist(err))
}

func TestDeleteModSingle(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	modsRoot, _, index := setupDeleteTest(t)
	installer := mods.NewInstaller(modsRoot)

	writeTestMod(t, modsRoot, "Single", "Author.Single")
	must.NoError(mods.DeleteMod(
		installer,
		"Single",
		false,
		modResolver([]mods.Mod{
			{ID: "Single", FolderPath: "Single", Manifest: mods.Manifest{UniqueID: "Author.Single"}},
		}),
		index,
		nexus.ModIDFromUpdateKeys,
	))
	_, err := os.Stat(filepath.Join(modsRoot, "Single"))
	must.True(os.IsNotExist(err))
}
