package smapi

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	"junimohut/internal/config"
	"junimohut/internal/httpclient"
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

// CheckSMAPIUpdate compares the installed SMAPI version against the latest GitHub release.
func CheckSMAPIUpdate(currentVersion string) (UpdateInfo, error) {
	info := UpdateInfo{CurrentVersion: currentVersion}
	release, err := fetchGitHubRelease()
	if err != nil {
		if httpclient.IsTransient(err) {
			return info, nil
		}
		return info, err
	}
	latest := strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
	info.LatestVersion = latest
	info.DownloadURL = installerURLFromRelease(release)
	if info.DownloadURL == "" {
		info.DownloadURL = resolveInstallDownloadURL()
	}
	if currentVersion != "" && latest != "" && !strings.EqualFold(currentVersion, "unknown") {
		cur, err1 := semver.NewVersion(strings.TrimPrefix(currentVersion, "v"))
		latestVer, err2 := semver.NewVersion(latest)
		if err1 == nil && err2 == nil {
			info.UpdateAvailable = latestVer.GreaterThan(cur)
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

// CheckModUpdates queries SMAPI's mod update service via POST /api/v{version}/mods.
func CheckModUpdates(mods []ModUpdateRequest, smapiVersion string) ([]ModUpdateResult, error) {
	if len(mods) == 0 {
		return nil, nil
	}
	return postModUpdates(mods, smapiVersion)
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
