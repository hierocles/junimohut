package smapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	smapiVersionAPIURL        = "https://smapi.io/api/version?game=stardewvalley&platform=windows"
	githubLatestReleaseAPIURL = "https://api.github.com/repos/Pathoschild/SMAPI/releases/latest"
	fallbackInstallURL        = "https://github.com/Pathoschild/SMAPI/releases/latest/download/SMAPI-4.3.6-installer.zip"
)

var smapiHTTPClient = &http.Client{Timeout: 30 * time.Second}

func fetchSMAPIIOInstallURL() (string, error) {
	resp, err := smapiHTTPClient.Get(smapiVersionAPIURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("smapi.io version API returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var data struct {
		DownloadURL string `json:"downloadUrl"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	return strings.TrimSpace(data.DownloadURL), nil
}

func fetchGitHubInstallURL() (string, error) {
	req, err := http.NewRequest(http.MethodGet, githubLatestReleaseAPIURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "JunimoHut")

	resp, err := smapiHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("GitHub releases API returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return "", err
	}
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, "installer") && strings.HasSuffix(name, ".zip") {
			return strings.TrimSpace(asset.BrowserDownloadURL), nil
		}
	}
	return "", fmt.Errorf("no SMAPI installer asset found in latest GitHub release")
}

func resolveInstallDownloadURL() string {
	if url, err := fetchSMAPIIOInstallURL(); err == nil && url != "" {
		return url
	}
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
