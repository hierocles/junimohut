package smapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveInstallDownloadURLFromGitHub(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v4.3.6","assets":[{"name":"SMAPI-4.3.6-installer.zip","browser_download_url":"https://example.com/smapi-github.zip"}]}`))
	}))
	t.Cleanup(github.Close)

	origGitHubURL := githubLatestReleaseAPIURL
	origClient := smapiHTTPClient
	githubLatestReleaseAPIURL = github.URL
	smapiHTTPClient = github.Client()
	t.Cleanup(func() {
		githubLatestReleaseAPIURL = origGitHubURL
		smapiHTTPClient = origClient
	})

	must.Equal("https://example.com/smapi-github.zip", resolveInstallDownloadURL())
}

func TestResolveInstallDownloadURLUsesFallbackWhenGitHubFails(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	origGitHubURL := githubLatestReleaseAPIURL
	origClient := smapiHTTPClient
	githubLatestReleaseAPIURL = "http://127.0.0.1:1/unreachable"
	smapiHTTPClient = origClient
	t.Cleanup(func() {
		githubLatestReleaseAPIURL = origGitHubURL
		smapiHTTPClient = origClient
	})

	must.Equal(fallbackInstallURL, resolveInstallDownloadURL())
}
