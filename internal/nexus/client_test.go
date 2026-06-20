package nexus

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDownloadFileResolvesLatestMain(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	const modID = 2400
	const fileID = 522942
	payload := []byte("mod archive bytes")

	var srvURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/games/stardewvalley/mods/2400/files.json":
			_ = json.NewEncoder(w).Encode(modFilesResponse{
				Files: []ModFile{
					{FileID: 100, CategoryName: "MAIN", UploadedTimestamp: 100, FileName: "Older.zip"},
					{FileID: fileID, CategoryName: "MAIN", UploadedTimestamp: 200, FileName: "CoolMod.zip"},
				},
			})
		case "/v1/games/stardewvalley/mods/2400/files/522942/download_link.json":
			_ = json.NewEncoder(w).Encode([]map[string]string{
				{"URI": srvURL + "/cdn/mod.zip"},
			})
		case "/cdn/mod.zip":
			w.Header().Set("Content-Length", "17")
			_, _ = w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	srvURL = srv.URL

	client := &Client{apiKey: "test-key", apiBase: srvURL}
	dir := t.TempDir()
	dm := NewDownloadManager(dir, nil)

	path, err := dm.DownloadFile(client, modID, 0, "Test Mod", nil)
	must.NoError(err)

	data, err := os.ReadFile(path)
	must.NoError(err)
	must.Equal(string(payload), string(data))

	entries := dm.List()
	must.Len(entries, 1)
	must.Equal("complete", entries[0].Status)
	must.Equal(100, entries[0].Progress)
	must.Equal(path, entries[0].FilePath)
	must.Equal("CoolMod.zip", filepath.Base(path))
}

func TestDownloadFileWithExplicitFileID(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	payload := []byte("explicit file")
	var srvURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/games/stardewvalley/mods/99/files.json":
			_ = json.NewEncoder(w).Encode(modFilesResponse{
				Files: []ModFile{
					{FileID: 42, CategoryName: "MAIN", FileName: "ExplicitFile.7z"},
				},
			})
		case "/v1/games/stardewvalley/mods/99/files/42/download_link.json":
			_ = json.NewEncoder(w).Encode([]map[string]string{
				{"URI": srvURL + "/file.zip"},
			})
		case "/file.zip":
			_, _ = w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	srvURL = srv.URL

	client := &Client{apiKey: "test-key", apiBase: srvURL}
	dm := NewDownloadManager(t.TempDir(), nil)

	path, err := dm.DownloadFile(client, 99, 42, "Explicit", nil)
	must.NoError(err)
	got, err := os.ReadFile(path)
	must.NoError(err)
	must.Equal(string(payload), string(got))
	must.Equal("ExplicitFile.7z", filepath.Base(path))
}
