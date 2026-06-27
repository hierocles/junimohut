//go:build windows

package app

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

func registerNXMProtocol() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe = filepath.Clean(exe)
	cmd := fmt.Sprintf(`"%s" "%%1"`, exe)

	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\nxm`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if err := k.SetStringValue("", "URL:Nexus Mod Manager Link"); err != nil {
		return err
	}
	if err := k.SetStringValue("URL Protocol", ""); err != nil {
		return err
	}

	iconKey, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\nxm\DefaultIcon`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer iconKey.Close()
	if err := iconKey.SetStringValue("", exe+",1"); err != nil {
		return err
	}

	cmdKey, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\nxm\shell\open\command`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer cmdKey.Close()
	return cmdKey.SetStringValue("", cmd)
}
