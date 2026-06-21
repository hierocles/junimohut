package mods

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScannerCountsEditableJSONFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	modDir := filepath.Join(root, "SampleMod")
	require.NoError(t, os.MkdirAll(modDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "manifest.json"), []byte(`{"Name":"Sample","Author":"A","Version":"1.0.0","UniqueID":"A.Sample"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "config.json"), []byte(`{}`), 0o644))

	list, err := NewScanner().Scan(ScanOptions{ModsRoot: root})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.True(t, list[0].HasJsonFiles)
	require.Equal(t, 1, list[0].JsonFileCount)
	require.True(t, list[0].HasConfig)
}

func TestIsEditableJSONFileExcludesManifest(t *testing.T) {
	t.Parallel()
	require.False(t, IsEditableJSONFile("manifest.json"))
	require.True(t, IsEditableJSONFile("config.json"))
}
