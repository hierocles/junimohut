package smapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckSMAPIUpdateFromGitHub(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v4.3.6","assets":[{"name":"SMAPI-4.3.6-installer.zip","browser_download_url":"https://example.com/install.zip"}]}`))
	}))
	t.Cleanup(srv.Close)

	origURL := githubLatestReleaseAPIURL
	origClient := smapiHTTPClient
	githubLatestReleaseAPIURL = srv.URL
	smapiHTTPClient = srv.Client()
	t.Cleanup(func() {
		githubLatestReleaseAPIURL = origURL
		smapiHTTPClient = origClient
	})

	info, err := CheckSMAPIUpdate("4.3.5")
	must.NoError(err)
	must.True(info.UpdateAvailable)
	must.Equal("4.3.6", info.LatestVersion)
	must.Equal("https://example.com/install.zip", info.DownloadURL)
}

func TestCheckSMAPIUpdateTransientNetworkReturnsNoError(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	origURL := githubLatestReleaseAPIURL
	origClient := smapiHTTPClient
	githubLatestReleaseAPIURL = "http://127.0.0.1:1/unreachable"
	smapiHTTPClient = origClient
	t.Cleanup(func() {
		githubLatestReleaseAPIURL = origURL
		smapiHTTPClient = origClient
	})

	info, err := CheckSMAPIUpdate("4.3.5")
	must.NoError(err)
	must.Equal("4.3.5", info.CurrentVersion)
	must.False(info.UpdateAvailable)
}
