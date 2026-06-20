package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlatformDataDir_WindowsUsesAppData(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows only")
	}
	must := require.New(t)

	t.Setenv("APPDATA", filepath.Join(t.TempDir(), "Roaming"))
	dir, err := platformDataDir()
	must.NoError(err)
	want := filepath.Join(os.Getenv("APPDATA"), appName)
	must.Equal(want, dir)
}
