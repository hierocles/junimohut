package smapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildModSearchRequest(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	req := buildModSearchRequest([]ModUpdateRequest{
		{UniqueID: "Pathoschild.ContentPatcher", Version: "1.9.2", UpdateKeys: []string{"Nexus:1915"}},
	}, "4.5.1")

	must.Len(req.Mods, 1)
	must.Equal("Pathoschild.ContentPatcher", req.Mods[0].ID)
	must.Equal("1.9.2", req.Mods[0].InstalledVersion)
	must.Equal([]string{"Nexus:1915"}, req.Mods[0].UpdateKeys)
	must.Equal("4.5.1", req.APIVersion)
	must.False(req.IncludeExtendedMetadata)
}

func TestAPIPathVersion(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	must.Equal("v3.0", apiPathVersion("unknown"))
	must.Equal("v4.5.1", apiPathVersion("4.5.1"))
	must.Equal("v4.5.1", apiPathVersion("v4.5.1"))
}

func TestMapModUpdateResults(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	mods := []ModUpdateRequest{
		{UniqueID: "Pathoschild.ContentPatcher", Version: "1.9.2"},
		{UniqueID: "Author.OtherMod", Version: "2.0.0"},
	}
	entries := []modSearchResponseEntry{
		{
			ID: "Pathoschild.ContentPatcher",
			SuggestedUpdate: &struct {
				Version string `json:"version"`
				URL     string `json:"url"`
			}{Version: "1.10.0", URL: "https://www.nexusmods.com/stardewvalley/mods/1915"},
		},
		{ID: "Author.OtherMod", Errors: []string{"page missing"}},
	}

	results := mapModUpdateResults(mods, entries)
	must.Len(results, 2)
	must.Equal("update", results[0].Status)
	must.Equal("1.10.0", results[0].LatestVersion)
	must.Equal("ok", results[1].Status)
	must.Equal("page missing", results[1].Message)
}

func TestPostModUpdates(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		must.Equal(http.MethodPost, r.Method)
		must.Equal("/api/v4.5.1/mods", r.URL.Path)
		must.Equal("JunimoHut", r.Header.Get("Application-Name"))

		var body modSearchRequest
		must.NoError(json.NewDecoder(r.Body).Decode(&body))
		must.Len(body.Mods, 1)

		_ = json.NewEncoder(w).Encode([]modSearchResponseEntry{
			{
				ID: "Pathoschild.ContentPatcher",
				SuggestedUpdate: &struct {
					Version string `json:"version"`
					URL     string `json:"url"`
				}{Version: "1.10.0", URL: "https://example.com/mod"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	origBase := smapiModUpdateBaseURL
	origClient := smapiHTTPClient
	smapiModUpdateBaseURL = srv.URL + "/api"
	smapiHTTPClient = srv.Client()
	t.Cleanup(func() {
		smapiModUpdateBaseURL = origBase
		smapiHTTPClient = origClient
	})

	results, err := postModUpdates([]ModUpdateRequest{
		{UniqueID: "Pathoschild.ContentPatcher", Version: "1.9.2", UpdateKeys: []string{"Nexus:1915"}},
	}, "4.5.1")
	must.NoError(err)
	must.Len(results, 1)
	must.Equal("update", results[0].Status)
	must.Equal("1.10.0", results[0].LatestVersion)
}
