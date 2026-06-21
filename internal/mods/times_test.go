package mods

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestManifestModTime(t *testing.T) {
	must := require.New(t)

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")
	must.NoError(os.WriteFile(manifestPath, []byte(`{"Name":"Test"}`), 0o644))
	must.NoError(os.Chtimes(manifestPath, time.Unix(1_704_067_200, 0), time.Unix(1_704_067_200, 0)))

	must.Equal(int64(1_704_067_200), ManifestModTime(dir))
	must.Zero(ManifestModTime(filepath.Join(dir, "missing")))
}
