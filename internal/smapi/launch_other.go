//go:build !windows

package smapi

import "os/exec"

func launchProcess(path, dir string) error {
	cmd := exec.Command(path)
	cmd.Dir = dir
	return cmd.Start()
}
