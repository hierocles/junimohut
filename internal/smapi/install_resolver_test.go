package smapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveInstallDownloadURLPrefersSMAPIIO(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"downloadUrl":"https://example.com/smapi-from-api.zip"}`))
	}))
	t.Cleanup(srv.Close)

	origURL := smapiVersionAPIURL
	origClient := smapiHTTPClient
	smapiVersionAPIURL = srv.URL
	smapiHTTPClient = srv.Client()
	t.Cleanup(func() {
		smapiVersionAPIURL = origURL
		smapiHTTPClient = origClient
	})

	must.Equal("https://example.com/smapi-from-api.zip", resolveInstallDownloadURL())
}

func TestResolveInstallDownloadURLFallsBackToGitHub(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"assets":[{"name":"SMAPI-9.9.9-installer.zip","browser_download_url":"https://example.com/smapi-github.zip"}]}`))
	}))
	t.Cleanup(github.Close)

	origSMAPIURL := smapiVersionAPIURL
	origGitHubURL := githubLatestReleaseAPIURL
	origClient := smapiHTTPClient
	smapiVersionAPIURL = "http://127.0.0.1:1/unreachable"
	githubLatestReleaseAPIURL = github.URL
	smapiHTTPClient = github.Client()
	t.Cleanup(func() {
		smapiVersionAPIURL = origSMAPIURL
		githubLatestReleaseAPIURL = origGitHubURL
		smapiHTTPClient = origClient
	})

	must.Equal("https://example.com/smapi-github.zip", resolveInstallDownloadURL())
}
