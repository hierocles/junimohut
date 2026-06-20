package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DetectionResult holds auto-detected paths.
type DetectionResult struct {
	GamePath  string `json:"gamePath"`
	SMAPIPath string `json:"smapiPath"`
	ModsRoot  string `json:"modsRoot"`
}

// DetectPaths attempts to find Stardew Valley, SMAPI, and the default mod library.
func DetectPaths(modLibrary string) DetectionResult {
	result := DetectionResult{}
	candidates := gamePathCandidates()
	for _, p := range candidates {
		if isValidGamePath(p) {
			result.GamePath = p
			break
		}
	}
	if result.GamePath != "" {
		result.SMAPIPath = detectSMAPI(result.GamePath)
		activeMods := filepath.Join(result.GamePath, "Mods")
		if _, err := os.Stat(activeMods); os.IsNotExist(err) {
			_ = os.MkdirAll(activeMods, 0o755)
		}
		if modLibrary != "" {
			result.ModsRoot = modLibrary
			_ = os.MkdirAll(result.ModsRoot, 0o755)
		}
	}
	return result
}

func isValidGamePath(p string) bool {
	if p == "" {
		return false
	}
	stardew := filepath.Join(p, "Stardew Valley.exe")
	if runtime.GOOS == "windows" {
		if _, err := os.Stat(stardew); err == nil {
			return true
		}
	}
	// Unix / macOS
	for _, name := range []string{"StardewModdingAPI", "Stardew Valley.app"} {
		if _, err := os.Stat(filepath.Join(p, name)); err == nil {
			return true
		}
	}
	if _, err := os.Stat(filepath.Join(p, "Stardew Valley")); err == nil {
		return true
	}
	return false
}

// IsSMAPIExe reports whether path looks like the SMAPI executable for the current OS.
func IsSMAPIExe(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	switch runtime.GOOS {
	case "windows":
		return name == "stardewmoddingapi.exe"
	default:
		return name == "stardewmoddingapi"
	}
}

// DetectSMAPI returns the SMAPI executable path within gamePath, or "" if not found.
func DetectSMAPI(gamePath string) string {
	return detectSMAPI(gamePath)
}

func detectSMAPI(gamePath string) string {
	switch runtime.GOOS {
	case "windows":
		p := filepath.Join(gamePath, "StardewModdingAPI.exe")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	case "darwin":
		p := filepath.Join(gamePath, "Contents", "MacOS", "StardewModdingAPI")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	default:
		p := filepath.Join(gamePath, "StardewModdingAPI")
		if _, err := os.Stat(p); err == nil {
			return p
		}
		if _, err := os.Stat(filepath.Join(gamePath, "StardewModdingAPI.dll")); err == nil {
			return filepath.Join(gamePath, "StardewModdingAPI")
		}
	}
	return ""
}

func gamePathCandidates() []string {
	var paths []string
	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "windows":
		steam := filepath.Join(os.Getenv("ProgramFiles(x86)"), "Steam", "steamapps", "common", "Stardew Valley")
		paths = append(paths, steam)
		if alt := os.Getenv("ProgramFiles"); alt != "" {
			paths = append(paths, filepath.Join(alt, "Steam", "steamapps", "common", "Stardew Valley"))
		}
		for _, drive := range []string{"C", "D", "E", "F"} {
			paths = append(paths, filepath.Join(drive+":", "Games", "Stardew Valley"))
			paths = append(paths, filepath.Join(drive+":", "GOG Games", "Stardew Valley"))
		}
	case "darwin":
		paths = append(paths,
			filepath.Join(home, "Library", "Application Support", "Steam", "steamapps", "common", "Stardew Valley", "Contents", "Resources", "app"),
			filepath.Join(home, "Library", "Application Support", "Steam", "steamapps", "common", "Stardew Valley"),
		)
	default:
		paths = append(paths,
			filepath.Join(home, ".steam", "steam", "steamapps", "common", "Stardew Valley"),
			filepath.Join(home, ".local", "share", "Steam", "steamapps", "common", "Stardew Valley"),
		)
	}
	return paths
}

// ValidateGamePath checks if a user-provided path is valid.
func ValidateGamePath(p string) bool {
	return isValidGamePath(strings.TrimSpace(p))
}

// ValidateSMAPIPath checks if SMAPI executable exists.
func ValidateSMAPIPath(p string) bool {
	p = strings.TrimSpace(p)
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}
