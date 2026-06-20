//go:build windows

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func junction(link, target string) error {
	// Directory junction (no admin required). Hide the console so GUI actions
	// like install/delete do not flash a cmd window.
	cmd := exec.Command("cmd", "/c", "mklink", "/J", link, target)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		// fallback to symlink
		if symErr := os.Symlink(target, link); symErr != nil {
			return fmt.Errorf("mklink: %s: %w", string(out), err)
		}
	}
	return nil
}
