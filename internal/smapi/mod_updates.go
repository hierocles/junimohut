package smapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"

	"junimohut/internal/httpclient"
)

var smapiModUpdateBaseURL = "https://smapi.io/api"

const (
	appName    = "JunimoHut"
	appVersion = "0.1.0"
)

type modSearchEntry struct {
	ID               string   `json:"id"`
	UpdateKeys       []string `json:"updateKeys,omitempty"`
	InstalledVersion string   `json:"installedVersion,omitempty"`
}

type modSearchRequest struct {
	Mods                    []modSearchEntry `json:"mods"`
	APIVersion              string           `json:"apiVersion,omitempty"`
	GameVersion             string           `json:"gameVersion,omitempty"`
	Platform                string           `json:"platform,omitempty"`
	IncludeExtendedMetadata bool             `json:"includeExtendedMetadata"`
}

type modSearchResponseEntry struct {
	ID              string `json:"id"`
	SuggestedUpdate *struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	} `json:"suggestedUpdate"`
	Errors []string `json:"errors"`
}

func apiPathVersion(smapiVersion string) string {
	v := strings.TrimSpace(strings.TrimPrefix(smapiVersion, "v"))
	if v == "" || strings.EqualFold(v, "unknown") {
		return "v3.0"
	}
	return "v" + v
}

func platformName() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows"
	case "darwin":
		return "Mac"
	case "linux":
		return "Linux"
	default:
		return runtime.GOOS
	}
}

func modUpdateURL(smapiVersion string) string {
	return fmt.Sprintf("%s/%s/mods", smapiModUpdateBaseURL, apiPathVersion(smapiVersion))
}

func buildModSearchRequest(mods []ModUpdateRequest, smapiVersion string) modSearchRequest {
	entries := make([]modSearchEntry, len(mods))
	for i, m := range mods {
		entries[i] = modSearchEntry{
			ID:               m.UniqueID,
			UpdateKeys:       m.UpdateKeys,
			InstalledVersion: m.Version,
		}
	}
	apiVer := strings.TrimPrefix(strings.TrimSpace(smapiVersion), "v")
	if apiVer == "" || strings.EqualFold(apiVer, "unknown") {
		apiVer = "3.0"
	}
	return modSearchRequest{
		Mods:                    entries,
		APIVersion:              apiVer,
		Platform:                platformName(),
		IncludeExtendedMetadata: false,
	}
}

func mapModUpdateResults(mods []ModUpdateRequest, entries []modSearchResponseEntry) []ModUpdateResult {
	byID := map[string]modSearchResponseEntry{}
	for _, e := range entries {
		byID[e.ID] = e
	}
	results := make([]ModUpdateResult, len(mods))
	for i, m := range mods {
		r := ModUpdateResult{
			UniqueID:       m.UniqueID,
			CurrentVersion: m.Version,
			Status:         "ok",
		}
		if e, ok := byID[m.UniqueID]; ok {
			if e.SuggestedUpdate != nil {
				r.LatestVersion = e.SuggestedUpdate.Version
				r.ModPageURL = e.SuggestedUpdate.URL
				r.Status = "update"
			}
			if len(e.Errors) > 0 {
				r.Message = strings.Join(e.Errors, "; ")
			}
		}
		results[i] = r
	}
	return results
}

func postModUpdates(mods []ModUpdateRequest, smapiVersion string) ([]ModUpdateResult, error) {
	body, err := json.Marshal(buildModSearchRequest(mods, smapiVersion))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, modUpdateURL(smapiVersion), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Application-Name", appName)
	req.Header.Set("Application-Version", appVersion)
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s %s", appName, appVersion, runtime.GOOS))

	resp, err := httpclient.DoWithRetry(smapiHTTPClient, req, 3)
	if err != nil {
		return nil, fmt.Errorf("could not reach SMAPI update service: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read SMAPI update response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("SMAPI update check failed: HTTP %d", resp.StatusCode)
	}
	var entries []modSearchResponseEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("SMAPI update check returned invalid data: %w", err)
	}
	return mapModUpdateResults(mods, entries), nil
}
