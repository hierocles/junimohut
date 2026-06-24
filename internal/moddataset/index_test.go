package moddataset

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDataBucket(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	must.Equal(2, DataBucket("Nexus", 2400))
	must.Equal(30, DataBucket("CurseForge", 309243))
	must.Equal(47, DataBucket("ModDrop", 470174))
}

func TestParsePageRef(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	site, id, ok := ParsePageRef("Nexus:2400")
	must.True(ok)
	must.Equal("Nexus", site)
	must.Equal(2400, id)

	_, _, ok = ParsePageRef("bad")
	must.False(ok)
}

func TestIndexSaveLoadAndLookup(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	dir := t.TempDir()
	cachePath := filepath.Join(dir, "index.json")
	idx, err := NewIndex(cachePath, dir)
	must.NoError(err)

	idx.mu.Lock()
	idx.file = IndexFile{
		FetchedAt: time.Now().Unix(),
		Pages: map[string][]string{
			"Pathoschild.ContentPatcher": {"Nexus:1915"},
		},
	}
	must.NoError(idx.saveLocked())
	idx.mu.Unlock()

	reloaded, err := NewIndex(cachePath, dir)
	must.NoError(err)
	must.Equal([]string{"Nexus:1915"}, reloaded.LookupPages("Pathoschild.ContentPatcher"))
	must.Equal(1915, reloaded.FirstNexusID("Pathoschild.ContentPatcher"))
}

func TestIndexRefresh(t *testing.T) {
	must := require.New(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string][]string{
			"Author.TestMod": {"Nexus:42"},
		})
	}))
	defer srv.Close()

	dir := t.TempDir()
	cachePath := filepath.Join(dir, "index.json")
	idx, err := NewIndex(cachePath, dir)
	must.NoError(err)

	oldURL := indexSourceURL
	t.Cleanup(func() { indexSourceURL = oldURL })
	indexSourceURL = srv.URL

	must.NoError(idx.RefreshIfStale(context.Background()))
	must.Equal([]string{"Nexus:42"}, idx.LookupPages("Author.TestMod"))
}

func TestFetchModPageCachesResult(t *testing.T) {
	must := require.New(t)

	const body = `{
		"Site": "Nexus",
		"Id": 2400,
		"Name": "Test Mod",
		"Author": "Author",
		"PageUrl": "https://example.com/mod",
		"TagLine": "A test",
		"Downloads": [
			{"Id": 1, "DisplayName": "Main", "FileName": "main.zip", "Version": "1.0", "Type": "Main", "Mods": [{}]}
		]
	}`

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	dir := t.TempDir()
	oldBase := pageDataBaseURL
	t.Cleanup(func() { pageDataBaseURL = oldBase })
	pageDataBaseURL = srv.URL

	page, err := FetchModPage(context.Background(), dir, "Nexus", 2400)
	must.NoError(err)
	must.Equal("Test Mod", page.Name)
	must.Len(page.Downloads, 1)
	must.Equal(1, page.Downloads[0].ModCount)

	page2, err := FetchModPage(context.Background(), dir, "Nexus", 2400)
	must.NoError(err)
	must.Equal(page, page2)
	must.Equal(1, calls)

	_, err = os.Stat(pageCachePath(dir, "Nexus", 2400))
	must.NoError(err)
}

func TestResolveNexusModID(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	dir := t.TempDir()
	idx, err := NewIndex(filepath.Join(dir, "index.json"), dir)
	must.NoError(err)
	idx.mu.Lock()
	idx.file.Pages = map[string][]string{
		"Author.FromIndex": {"Nexus:99"},
	}
	idx.mu.Unlock()

	must.Equal(1915, ResolveNexusModID("x", []string{"Nexus:1915"}, idx, 50, 10))
	must.Equal(50, ResolveNexusModID("x", nil, idx, 50, 10))
	must.Equal(99, ResolveNexusModID("Author.FromIndex", nil, idx, 0, 10))
	must.Equal(10, ResolveNexusModID("x", nil, idx, 0, 10))
}
