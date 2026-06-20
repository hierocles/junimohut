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

func TestParseNXMURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		modID   int
		fileID  int
		wantKey string
		wantExp int64
		wantErr bool
	}{
		{
			name:   "basic",
			raw:    "nxm://stardewvalley/mods/2400/files/12345",
			modID:  2400,
			fileID: 12345,
		},
		{
			name:    "free tier auth",
			raw:     "nxm://stardewvalley/mods/2400/files/12345?key=abc%2Bdef&expires=1700000000&user_id=42",
			modID:   2400,
			fileID:  12345,
			wantKey: "abc+def",
			wantExp: 1700000000,
		},
		{
			name:    "invalid",
			raw:     "https://www.nexusmods.com/stardewvalley/mods/2400",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			must := require.New(t)

			got, err := ParseNXMURL(tc.raw)
			if tc.wantErr {
				must.Error(err)
				return
			}
			must.NoError(err)
			must.Equal(tc.modID, got.ModID)
			must.Equal(tc.fileID, got.FileID)
			if tc.wantKey == "" {
				must.Nil(got.Auth)
				return
			}
			must.NotNil(got.Auth)
			must.Equal(tc.wantKey, got.Auth.Key)
			must.Equal(tc.wantExp, got.Auth.Expires)
		})
	}
}

func TestDownloadFileUsesNXMAuth(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	const modID = 2400
	const fileID = 12345
	payload := []byte("free download")

	var srvURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/games/stardewvalley/mods/2400/files.json":
			_ = json.NewEncoder(w).Encode(modFilesResponse{
				Files: []ModFile{
					{FileID: fileID, CategoryName: "MAIN", FileName: "FreeMod.zip"},
				},
			})
		case "/v1/games/stardewvalley/mods/2400/files/12345/download_link.json":
			if r.URL.Query().Get("key") != "free-key" || r.URL.Query().Get("expires") != "1700000000" {
				http.Error(w, "missing auth", http.StatusForbidden)
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]string{
				{"URI": srvURL + "/cdn/mod.zip"},
			})
		case "/cdn/mod.zip":
			_, _ = w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	srvURL = srv.URL

	client := &Client{apiKey: "test-key", apiBase: srvURL}
	dm := NewDownloadManager(t.TempDir(), nil)
	auth := &DownloadAuth{Key: "free-key", Expires: 1700000000}

	path, err := dm.DownloadFile(client, modID, fileID, "Free Mod", auth)
	must.NoError(err)
	got, err := os.ReadFile(path)
	must.NoError(err)
	must.Equal(string(payload), string(got))
	must.Equal("FreeMod.zip", filepath.Base(path))
}
