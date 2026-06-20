//go:build !windows

package main

import (
	"fmt"
	"runtime"
)

func registerNXMProtocol() error {
	return fmt.Errorf("NXM protocol registration is supported on Windows only (current OS: %s)", runtime.GOOS)
}
