package mods

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadWriteModConfig(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	folder := "Author.TestMod"
	modDir := filepath.Join(root, folder)
	require.NoError(t, os.MkdirAll(modDir, 0o755))

	_, err := ReadModConfig(root, folder)
	require.ErrorIs(t, err, ErrModConfigNotFound)

	content := `{"Setting": true}`
	require.NoError(t, WriteModConfig(root, folder, content))

	got, err := ReadModConfig(root, folder)
	require.NoError(t, err)
	require.Equal(t, content, got)

	err = WriteModConfig(root, folder, `{invalid`)
	require.ErrorIs(t, err, ErrModConfigInvalidJSON)
}

func TestListJsonFilesAndTree(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	modDir := filepath.Join(root, "SampleMod")
	require.NoError(t, os.MkdirAll(filepath.Join(modDir, "i18n"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "config.json"), []byte(`{}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "i18n", "default.json"), []byte(`{}`), 0o644))

	paths, err := ListJsonFileRelPaths(modDir)
	require.NoError(t, err)
	require.Equal(t, []string{"config.json", "i18n/default.json"}, paths)
	require.Equal(t, "config.json", DefaultJsonRelPath(paths))

	tree := BuildJsonFileTree(paths)
	require.Len(t, tree, 2)
	require.False(t, tree[0].IsDir)
	require.Equal(t, "config.json", tree[0].RelPath)
	require.True(t, tree[1].IsDir)
	require.Equal(t, "i18n", tree[1].Name)
	require.Len(t, tree[1].Children, 1)
}

func TestResolveModJSONPathRejectsTraversal(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	folder := "SampleMod"
	require.NoError(t, os.MkdirAll(filepath.Join(root, folder), 0o755))
	_, err := ResolveModJSONPath(root, folder, "../secret.json")
	require.ErrorIs(t, err, ErrModConfigInvalidPath)
}

func TestReadWriteNestedJsonFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	folder := "SampleMod"
	modDir := filepath.Join(root, folder)
	require.NoError(t, os.MkdirAll(filepath.Join(modDir, "assets"), 0o755))

	content := `{"x": 1}`
	require.NoError(t, WriteModJsonFile(root, folder, "assets/data.json", content))
	got, err := ReadModJsonFile(root, folder, "assets/data.json")
	require.NoError(t, err)
	require.Equal(t, content, got)
}
