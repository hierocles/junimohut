package mods

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeManifest(t *testing.T, dir string) {
	t.Helper()
	must := require.New(t)

	must.NoError(os.MkdirAll(dir, 0o755))
	manifest := `{"Name":"M","Author":"A","Version":"1.0.0","UniqueID":"A.M"}`
	must.NoError(os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644))
}

func TestIsRootModManifestNestedPack(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	pack := filepath.Join(root, "(AT) Chest Deco")
	writeManifest(t, pack)
	for _, sub := range []string{"Textures/Barrel", "Textures/Crate", "Textures/Rack"} {
		writeManifest(t, filepath.Join(pack, sub))
	}

	all, err := findAllManifests(root)
	must.NoError(err)
	must.Len(all, 4)

	roots := FilterRootManifests(all, root)
	must.Len(roots, 1)
	must.Equal("(AT) Chest Deco", filepath.Base(filepath.Dir(roots[0])))
}

func TestIsRootModManifestMultiModZip(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	writeManifest(t, filepath.Join(root, "ModA"))
	writeManifest(t, filepath.Join(root, "ModB"))

	all, err := findAllManifests(root)
	must.NoError(err)
	roots := FilterRootManifests(all, root)
	must.Len(roots, 2)
}

func TestIsRootModManifestFrameworkExamples(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	writeManifest(t, filepath.Join(root, "Framework"))
	writeManifest(t, filepath.Join(root, "Framework", "Examples", "Child"))

	all, err := findAllManifests(root)
	must.NoError(err)
	roots := FilterRootManifests(all, root)
	must.Len(roots, 1)
	must.Equal("Framework", filepath.Base(filepath.Dir(roots[0])))
}
