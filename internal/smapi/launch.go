package smapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"junimohut/internal/config"
)

// Launcher starts SMAPI with the game's Mods folder (symlinked enabled mods).
type Launcher struct {
	GamePath  string
	SMAPIPath string
}

func NewLauncher(gamePath, smapiPath string) *Launcher {
	return &Launcher{GamePath: gamePath, SMAPIPath: smapiPath}
}

// Launch starts SMAPI. Enabled mods are expected as symlinks in {GamePath}/Mods.
func (l *Launcher) Launch() error {
	if l.SMAPIPath == "" {
		return fmt.Errorf("SMAPI path not configured")
	}
	if _, err := os.Stat(l.SMAPIPath); err != nil {
		return fmt.Errorf("SMAPI not found at %s", l.SMAPIPath)
	}

	if !config.IsSMAPIExe(l.SMAPIPath) {
		if detected := config.DetectSMAPI(l.gameDir()); detected != "" {
			l.SMAPIPath = detected
		} else {
			return fmt.Errorf("path %q does not look like StardewModdingAPI.exe and no SMAPI was found in %s", l.SMAPIPath, l.gameDir())
		}
	}

	return launchProcess(l.SMAPIPath, l.gameDir())
}

func (l *Launcher) gameDir() string {
	if l.GamePath != "" {
		return l.GamePath
	}
	return filepath.Dir(l.SMAPIPath)
}

// Version reads SMAPI version from the install folder.
func (l *Launcher) Version() string {
	dir := l.gameDir()
	for _, name := range []string{"smapi-internal", filepath.Join("Mods", "smapi-internal")} {
		p := filepath.Join(dir, name, "config.json")
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var cfg struct {
			Version string `json:"Version"`
		}
		if json.Unmarshal(data, &cfg) == nil && cfg.Version != "" {
			return cfg.Version
		}
	}
	return "unknown"
}

// UpdateInfo describes available SMAPI update.
type UpdateInfo struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	DownloadURL     string `json:"downloadUrl"`
}

// CheckSMAPIUpdate queries smapi.io for latest version.
func CheckSMAPIUpdate(currentVersion string) (UpdateInfo, error) {
	info := UpdateInfo{CurrentVersion: currentVersion}
	resp, err := smapiHTTPClient.Get(smapiVersionAPIURL)
	if err != nil {
		return info, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var data struct {
		Version     string `json:"version"`
		DownloadURL string `json:"downloadUrl"`
	}
	if json.Unmarshal(body, &data) != nil {
		// fallback: try alternate format
		var alt struct {
			Stable string `json:"stable"`
		}
		_ = json.Unmarshal(body, &alt)
		data.Version = alt.Stable
	}
	info.LatestVersion = data.Version
	info.DownloadURL = data.DownloadURL
	if currentVersion != "" && data.Version != "" {
		cur, err1 := semver.NewVersion(strings.TrimPrefix(currentVersion, "v"))
		latest, err2 := semver.NewVersion(strings.TrimPrefix(data.Version, "v"))
		if err1 == nil && err2 == nil {
			info.UpdateAvailable = latest.GreaterThan(cur)
		}
	}
	return info, nil
}

// ModUpdateResult from SMAPI mod update API.
type ModUpdateResult struct {
	UniqueID       string `json:"uniqueId"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	Status         string `json:"status"` // update, ok, incompatible
	ModPageURL     string `json:"modPageUrl"`
	Message        string `json:"message"`
}

// CheckModUpdates queries SMAPI's mod update service.
func CheckModUpdates(mods []ModUpdateRequest) ([]ModUpdateResult, error) {
	if len(mods) == 0 {
		return nil, nil
	}
	params := url.Values{}
	for _, m := range mods {
		params.Add("mods", fmt.Sprintf("%s@%s", m.UniqueID, m.Version))
		for _, k := range m.UpdateKeys {
			params.Add("keys", k)
		}
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://smapi.io/api/mods/updates?" + params.Encode())
	if err != nil {
		return checkModUpdatesFallback(mods)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var results []ModUpdateResult
	if json.Unmarshal(body, &results) != nil {
		return checkModUpdatesFallback(mods)
	}
	return results, nil
}

type ModUpdateRequest struct {
	UniqueID   string
	Version    string
	UpdateKeys []string
}

func checkModUpdatesFallback(mods []ModUpdateRequest) ([]ModUpdateResult, error) {
	results := make([]ModUpdateResult, len(mods))
	for i, m := range mods {
		results[i] = ModUpdateResult{
			UniqueID:       m.UniqueID,
			CurrentVersion: m.Version,
			Status:         "ok",
		}
	}
	return results, nil
}

// InstallSMAPI downloads and runs the SMAPI installer (Windows).
func InstallSMAPI(downloadURL, gamePath string) error {
	if downloadURL == "" {
		downloadURL = resolveInstallDownloadURL()
	}

	tmp, err := os.MkdirTemp("", "smapi-install-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	resp, err := smapiHTTPClient.Get(downloadURL)
	if err != nil {
		return downloadInstallError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("could not download SMAPI installer: HTTP %d. Download it manually from https://smapi.io", resp.StatusCode)
	}
	zipPath := filepath.Join(tmp, "smapi.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		return err
	}

	// On Windows, user typically runs installer manually; open folder
	if runtime.GOOS == "windows" {
		return exec.Command("explorer", tmp).Start()
	}
	return fmt.Errorf("automatic SMAPI install not supported on this platform; download from smapi.io")
}
