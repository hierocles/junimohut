package nexus

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCategoryNameForMod(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	ResetGameCategoriesCache()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/games/stardewvalley.json":
			_ = json.NewEncoder(w).Encode(gameInfoResponse{
				Categories: []gameCategory{
					{CategoryID: 7, Name: "User Interface"},
				},
			})
		case "/v1/games/stardewvalley/mods/509.json":
			_ = json.NewEncoder(w).Encode(modInfoResponse{
				ModInfo: ModInfo{ModID: 509, CategoryID: 7, Name: "Lookup Anything"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := &Client{apiKey: "test-key", apiBase: srv.URL}
	name, err := client.CategoryNameForMod(509)
	must.NoError(err)
	must.Equal("User Interface", name)
}
