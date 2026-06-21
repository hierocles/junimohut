package mods

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidJSONC(t *testing.T) {
	t.Parallel()

	require.NoError(t, ValidJSONC(`{"a": 1}`))
	require.NoError(t, ValidJSONC(`{
		// line comment
		"a": 1,
		/* block comment */
		"b": 2
	}`))
	require.ErrorIs(t, ValidJSONC(`{invalid`), ErrModConfigInvalidJSON)
}

func TestWriteModJsonFilePreservesComments(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	folder := "SampleMod"
	modDir := filepath.Join(root, folder)
	require.NoError(t, os.MkdirAll(filepath.Join(modDir, "assets"), 0o755))

	content := "{\n  // keep me\n  \"x\": 1\n}"
	require.NoError(t, WriteModJsonFile(root, folder, "assets/data.json", content))
	got, err := ReadModJsonFile(root, folder, "assets/data.json")
	require.NoError(t, err)
	require.Equal(t, content, got)
}
