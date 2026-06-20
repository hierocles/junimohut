package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// LinkDir creates a symlink or junction from link to target.
func LinkDir(link, target string) error {
	_ = os.RemoveAll(link)
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		return junction(link, absTarget)
	}
	return os.Symlink(absTarget, link)
}

// ClearDir removes all entries in a directory.
func ClearDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dir, 0o755)
		}
		return err
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// OpenPath opens a file or folder with the OS default handler.
func OpenPath(path string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

// RevealInFileManager opens the parent folder and selects the file (best effort).
func RevealInFileManager(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", "/select,", abs).Start()
	case "darwin":
		return exec.Command("open", "-R", abs).Start()
	default:
		return exec.Command("xdg-open", filepath.Dir(abs)).Start()
	}
}
