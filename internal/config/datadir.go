package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const portableMarker = "portable.txt"

// ResolveDataDir returns the application data directory.
// Portable mode is enabled when portable.txt exists next to the executable (data/ subfolder).
// Otherwise platform defaults apply: Windows APPDATA, macOS Application Support, Linux XDG data home.
func ResolveDataDir() (string, error) {
	if dir, ok := portableDataDir(); ok {
		return dir, nil
	}
	return platformDataDir()
}

func portableDataDir() (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", false
	}
	exeDir := filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(exeDir, portableMarker)); err != nil {
		return "", false
	}
	return filepath.Join(exeDir, "data"), true
}

func platformDataDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA is not set")
		}
		return filepath.Join(appData, appName), nil
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", appName), nil
	default:
		if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
			return filepath.Join(dataHome, appName), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", appName), nil
	}
}
