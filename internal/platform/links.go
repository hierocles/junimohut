package platform

import (
	"os"
	"path/filepath"
	"strings"
)

// IsModLink reports whether path is a symlink or directory junction.
func IsModLink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return true
	}
	target, err := os.Readlink(path)
	return err == nil && target != ""
}

// LinkTarget returns the absolute target of a symlink or junction.
func LinkTarget(path string) (string, error) {
	target, err := os.Readlink(path)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(path), target)
	}
	return filepath.Clean(target), nil
}

// IsManagedModLink reports whether path is a link owned by SDVM pointing into libraryRoot.
func IsManagedModLink(path, libraryRoot string) bool {
	if !IsModLink(path) {
		return false
	}
	target, err := LinkTarget(path)
	if err != nil {
		return false
	}
	absLib, err := filepath.Abs(libraryRoot)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	if absTarget == absLib {
		return true
	}
	sep := string(filepath.Separator)
	return strings.HasPrefix(absTarget, absLib+sep)
}
