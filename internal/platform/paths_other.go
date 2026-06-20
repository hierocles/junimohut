//go:build !windows

package platform

import "errors"

func junction(link, target string) error {
	return errors.New("platform: junction is only supported on Windows")
}
