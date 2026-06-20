package nexus

import (
	"archive/zip"
	"os"
	"path/filepath"
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

	idx, err := NewDownloadIndex(dataDir, downloadsDir)
	must.NoError(err)

	archive := filepath.Join(downloadsDir, "CoolMod.zip")
	must.NoError(os.WriteFile(archive, []byte("zip"), 0o644))

	must.NoError(idx.Record(DownloadRecord{
		ArchivePath:  archive,
		NexusModID:   2400,
		UniqueID:     "Author.CoolMod",
		FileName:     "CoolMod.zip",
		DownloadedAt: 100,
	}))

	reloaded, err := NewDownloadIndex(dataDir, downloadsDir)
	must.NoError(err)
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
	idx, err := NewDownloadIndex(dataDir, downloadsDir)
	must.NoError(err)
	must.NoError(idx.Record(DownloadRecord{
		ArchivePath:  missing,
		UniqueID:     "Author.Gone",
		DownloadedAt: time.Now().Unix(),
	}))

	reloaded, err := NewDownloadIndex(dataDir, downloadsDir)
	must.NoError(err)
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

	idx, err := NewDownloadIndex(dataDir, downloadsDir)
	must.NoError(err)
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

	idx, err := NewDownloadIndex(dataDir, downloadsDir)
	must.NoError(err)
	path, ok := idx.FindForMod("Author.Scanned", 0)
	must.True(ok)
	must.Equal(archive, path)
}

func TestDownloadIndexInDownloadsDir(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	dataDir := t.TempDir()
	downloadsDir := filepath.Join(dataDir, "downloads")
	idx, err := NewDownloadIndex(dataDir, downloadsDir)
	must.NoError(err)
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

	idx, err := NewDownloadIndex(dataDir, downloadsDir)
	must.NoError(err)
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
	_, err = os.Stat(newer)
	must.True(os.IsNotExist(err))
	list = idx.List()
	must.Len(list, 1)
	must.Equal(older, list[0].ArchivePath)
}
