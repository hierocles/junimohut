package smapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"junimohut/internal/httpclient"
)

var (
	githubLatestReleaseAPIURL = "https://api.github.com/repos/Pathoschild/SMAPI/releases/latest"
	fallbackInstallURL        = "https://github.com/Pathoschild/SMAPI/releases/latest/download/SMAPI-4.3.6-installer.zip"
)

var smapiHTTPClient = httpclient.Default()

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func fetchGitHubRelease() (githubRelease, error) {
	req, err := http.NewRequest(http.MethodGet, githubLatestReleaseAPIURL, nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", appName)

	resp, err := httpclient.DoWithRetry(smapiHTTPClient, req, 3)
	if err != nil {
		return githubRelease{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return githubRelease{}, fmt.Errorf("GitHub releases API returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return githubRelease{}, err
	}
	var release githubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return githubRelease{}, err
	}
	return release, nil
}

func installerURLFromRelease(release githubRelease) string {
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, "installer") && strings.HasSuffix(name, ".zip") {
			return strings.TrimSpace(asset.BrowserDownloadURL)
		}
	}
	return ""
}

func fetchGitHubInstallURL() (string, error) {
	release, err := fetchGitHubRelease()
	if err != nil {
		return "", err
	}
	url := installerURLFromRelease(release)
	if url == "" {
		return "", fmt.Errorf("no SMAPI installer asset found in latest GitHub release")
	}
	return url, nil
}

func resolveInstallDownloadURL() string {
	if url, err := fetchGitHubInstallURL(); err == nil && url != "" {
		return url
	}
	return fallbackInstallURL
}

func downloadInstallError(err error) error {
	msg := err.Error()
	if strings.Contains(msg, "lookup") || strings.Contains(msg, "getaddrinfo") || strings.Contains(msg, "no such host") {
		return fmt.Errorf("could not reach SMAPI download servers (DNS/network error). Check your internet connection, or download the installer manually from https://smapi.io")
	}
	return fmt.Errorf("could not download SMAPI installer: %w. You can download it manually from https://smapi.io", err)
}
