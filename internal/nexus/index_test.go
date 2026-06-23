package nexus

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

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

func TestDownloadIndexRecordAndLoad(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	dataDir := t.TempDir()
	downloadsDir := filepath.Join(dataDir, "downloads")
	must.NoError(os.MkdirAll(downloadsDir, 0o755))

	idx := newTestDownloadIndex(t, dataDir, downloadsDir)

	archive := filepath.Join(downloadsDir, "CoolMod.zip")
	must.NoError(os.WriteFile(archive, []byte("zip"), 0o644))

	must.NoError(idx.Record(DownloadRecord{
		ArchivePath:  archive,
		NexusModID:   2400,
		UniqueID:     "Author.CoolMod",
		FileName:     "CoolMod.zip",
		DownloadedAt: 100,
	}))

	reloaded := newTestDownloadIndex(t, dataDir, downloadsDir)
	path, ok := reloaded.FindForMod("Author.CoolMod", 0)
	must.True(ok)
	must.Equal(archive, path)
}

func TestDownloadIndexPrunesMissingFiles(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	dataDir := t.TempDir()
	downloadsDir := filepath.Join(dataDir, "downloads")
	must.NoError(os.MkdirAll(downloadsDir, 0o755))

	missing := filepath.Join(downloadsDir, "gone.zip")
	idx := newTestDownloadIndex(t, dataDir, downloadsDir)
	must.NoError(idx.Record(DownloadRecord{
		ArchivePath:  missing,
		UniqueID:     "Author.Gone",
		DownloadedAt: time.Now().Unix(),
	}))

	reloaded := newTestDownloadIndex(t, dataDir, downloadsDir)
	reloaded.Reconcile()
	_, ok := reloaded.FindForMod("Author.Gone", 0)
	must.False(ok)
}

func TestDownloadIndexFindPrefersUniqueID(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	dataDir := t.TempDir()
	downloadsDir := filepath.Join(dataDir, "downloads")
	must.NoError(os.MkdirAll(downloadsDir, 0o755))

	older := filepath.Join(downloadsDir, "older.zip")
	newer := filepath.Join(downloadsDir, "newer.zip")
	for _, path := range []string{older, newer} {
		must.NoError(os.WriteFile(path, []byte("zip"), 0o644))
	}

	idx := newTestDownloadIndex(t, dataDir, downloadsDir)
	must.NoError(idx.Record(DownloadRecord{
		ArchivePath: older, NexusModID: 99, UniqueID: "Author.Mod", DownloadedAt: 100,
	}))
	must.NoError(idx.Record(DownloadRecord{
		ArchivePath: newer, NexusModID: 99, DownloadedAt: 200,
	}))

	path, ok := idx.FindForMod("Author.Mod", 99)
	must.True(ok)
	must.Equal(older, path)
}

func TestDownloadIndexReconcileIndexesUnlistedArchive(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	dataDir := t.TempDir()
	downloadsDir := filepath.Join(dataDir, "downloads")
	must.NoError(os.MkdirAll(downloadsDir, 0o755))

	archive := filepath.Join(downloadsDir, "ScannedMod.zip")
	writeTestZip(t, archive, map[string]string{
		"manifest.json": `{"Name":"Scanned","Author":"A","Version":"1.0.0","UniqueID":"Author.Scanned"}`,
	})

	idx := newTestDownloadIndex(t, dataDir, downloadsDir)
	idx.Reconcile()
	path, ok := idx.FindForMod("Author.Scanned", 0)
	must.True(ok)
	must.Equal(archive, path)

	list := idx.List()
	must.Len(list, 1)
	must.Equal("Scanned", list[0].ModName)
}

func TestDownloadIndexEnrichesNumericUpdateKeys(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	dataDir := t.TempDir()
	downloadsDir := filepath.Join(dataDir, "downloads")
	must.NoError(os.MkdirAll(downloadsDir, 0o755))

	archive := filepath.Join(downloadsDir, "PetFacelift.zip")
	writeTestZip(t, archive, map[string]string{
		"(AT) Pet Facelift/manifest.json": `{
			"Name":"[AT] Pet Facelift",
			"Author":"siamece",
			"Version":"1.1.0",
			"UniqueID":"siamece.AT.PetFacelift",
			"UpdateKeys":[9097],
			"ContentPackFor":{"UniqueID":"PeacefulEnd.AlternativeTextures"}
		}`,
	})

	idx := newTestDownloadIndex(t, dataDir, downloadsDir)
	idx.Reconcile()
	list := idx.List()
	must.Len(list, 1)
	must.Equal("siamece.AT.PetFacelift", list[0].UniqueID)
	must.Equal("[AT] Pet Facelift", list[0].ModName)
	must.Equal(9097, list[0].NexusModID)

	// Second reconcile should not rewrite the index when metadata is complete.
	info, err := os.Stat(filepath.Join(dataDir, "downloads.json"))
	must.NoError(err)
	firstMod := info.ModTime()
	time.Sleep(10 * time.Millisecond)
	idx.Reconcile()
	info, err = os.Stat(filepath.Join(dataDir, "downloads.json"))
	must.NoError(err)
	must.Equal(firstMod, info.ModTime())
}

func TestDownloadIndexReconcileBackfillsMissingUniqueID(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	dataDir := t.TempDir()
	downloadsDir := filepath.Join(dataDir, "downloads")
	must.NoError(os.MkdirAll(downloadsDir, 0o755))

	archive := filepath.Join(downloadsDir, "KnownMod.zip")
	writeTestZip(t, archive, map[string]string{
		"manifest.json": `{"Name":"Known Mod","Author":"A","Version":"1.0.0","UniqueID":"Author.KnownMod"}`,
	})

	must.NoError(os.WriteFile(filepath.Join(dataDir, "downloads.json"), []byte(`{
		"records":[{"archivePath":`+strconv.Quote(filepath.ToSlash(archive))+`,"fileName":"KnownMod.zip","downloadedAt":100}]
	}`), 0o644))

	idx := newTestDownloadIndex(t, dataDir, downloadsDir)
	idx.Reconcile()
	list := idx.List()
	must.Len(list, 1)
	must.Equal("Author.KnownMod", list[0].UniqueID)
	must.Equal("Known Mod", list[0].ModName)
}

func TestDownloadIndexInDownloadsDir(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	dataDir := t.TempDir()
	downloadsDir := filepath.Join(dataDir, "downloads")
	idx := newTestDownloadIndex(t, dataDir, downloadsDir)
	inside := filepath.Join(downloadsDir, "mod.zip")
	outside := filepath.Join(dataDir, "other.zip")
	must.True(idx.InDownloadsDir(inside))
	must.False(idx.InDownloadsDir(outside))
}

func TestDownloadIndexListAndDelete(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	dataDir := t.TempDir()
	downloadsDir := filepath.Join(dataDir, "downloads")
	must.NoError(os.MkdirAll(downloadsDir, 0o755))

	older := filepath.Join(downloadsDir, "older.zip")
	newer := filepath.Join(downloadsDir, "newer.zip")
	for _, path := range []string{older, newer} {
		must.NoError(os.WriteFile(path, []byte("zip"), 0o644))
	}

	idx := newTestDownloadIndex(t, dataDir, downloadsDir)
	must.NoError(idx.Record(DownloadRecord{
		ArchivePath: older, FileName: "older.zip", DownloadedAt: 100,
	}))
	must.NoError(idx.Record(DownloadRecord{
		ArchivePath: newer, FileName: "newer.zip", DownloadedAt: 200,
	}))

	list := idx.List()
	must.Len(list, 2)
	must.Equal(newer, list[0].ArchivePath)

	must.NoError(idx.Delete(newer))
	_, err := os.Stat(newer)
	must.True(os.IsNotExist(err))
	list = idx.List()
	must.Len(list, 1)
	must.Equal(older, list[0].ArchivePath)
}
