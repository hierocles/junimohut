package modtimes

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecordInstall(t *testing.T) {
	must := require.New(t)

	path := filepath.Join(t.TempDir(), "mod-times.json")
	svc, err := NewService(path)
	must.NoError(err)

	must.NoError(svc.RecordInstall("folder::Author.Mod"))
	rec, ok := svc.Get("folder::Author.Mod")
	must.True(ok)
	must.Greater(rec.InstallTime, int64(0))
	must.Equal(rec.InstallTime, rec.LastUpdated)
}

func TestRecordInstallAndUpdate(t *testing.T) {
	must := require.New(t)

	path := filepath.Join(t.TempDir(), "mod-times.json")
	svc, err := NewService(path)
	must.NoError(err)

	must.NoError(svc.SeedBatch(map[string]int64{"folder::Author.Mod": 1_000}))
	rec, ok := svc.Get("folder::Author.Mod")
	must.True(ok)
	must.Equal(int64(1_000), rec.InstallTime)
	must.Equal(int64(1_000), rec.LastUpdated)

	must.NoError(svc.RecordUpdate("folder::Author.Mod", 0))
	updated, ok := svc.Get("folder::Author.Mod")
	must.True(ok)
	must.Equal(rec.InstallTime, updated.InstallTime)
	must.Greater(updated.LastUpdated, updated.InstallTime)
}

func TestRecordUpdateSeedsFromFallback(t *testing.T) {
	must := require.New(t)

	path := filepath.Join(t.TempDir(), "mod-times.json")
	svc, err := NewService(path)
	must.NoError(err)

	fallback := int64(1_704_067_200)
	must.NoError(svc.RecordUpdate("folder::Author.Mod", fallback))
	rec, ok := svc.Get("folder::Author.Mod")
	must.True(ok)
	must.Equal(fallback, rec.InstallTime)
	must.Greater(rec.LastUpdated, fallback)
}

func TestSeedBatchAndDelete(t *testing.T) {
	must := require.New(t)

	path := filepath.Join(t.TempDir(), "mod-times.json")
	svc, err := NewService(path)
	must.NoError(err)

	must.NoError(svc.RecordInstall("existing::Author.Mod"))
	must.NoError(svc.SeedBatch(map[string]int64{
		"existing::Author.Mod": 100,
		"new::Author.Mod":      200,
	}))

	existing, ok := svc.Get("existing::Author.Mod")
	must.True(ok)
	must.NotEqual(int64(100), existing.InstallTime)

	seeded, ok := svc.Get("new::Author.Mod")
	must.True(ok)
	must.Equal(int64(200), seeded.InstallTime)
	must.Equal(int64(200), seeded.LastUpdated)

	must.NoError(svc.Delete("new::Author.Mod"))
	_, ok = svc.Get("new::Author.Mod")
	must.False(ok)
}

func TestPersistenceRoundTrip(t *testing.T) {
	must := require.New(t)

	path := filepath.Join(t.TempDir(), "mod-times.json")
	svc, err := NewService(path)
	must.NoError(err)
	must.NoError(svc.RecordInstall("folder::Author.Mod"))

	reloaded, err := NewService(path)
	must.NoError(err)
	rec, ok := reloaded.Get("folder::Author.Mod")
	must.True(ok)
	must.Greater(rec.InstallTime, int64(0))

	data, err := os.ReadFile(path)
	must.NoError(err)
	var st store
	must.NoError(json.Unmarshal(data, &st))
	must.Equal(rec, st.Mods["folder::Author.Mod"])
}
