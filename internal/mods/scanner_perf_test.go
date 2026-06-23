package mods

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestScannerSkipsModAssetTrees(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	root := t.TempDir()
	modDir := filepath.Join(root, "BigMod")
	writeManifest(t, modDir)

	assets := filepath.Join(modDir, "assets", "textures")
	must.NoError(os.MkdirAll(assets, 0o755))
	for i := range 2000 {
		name := filepath.Join(assets, fmt.Sprintf("tile_%04d.png", i))
		must.NoError(os.WriteFile(name, []byte("x"), 0o644))
	}

	start := time.Now()
	list, err := NewScanner().Scan(ScanOptions{ModsRoot: root})
	elapsed := time.Since(start)

	must.NoError(err)
	must.Len(list, 1)
	must.True(elapsed < 2*time.Second, "scan took %v", elapsed)
}

func TestScannerFindsSiblingContentPacks(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	root := t.TempDir()
	container := filepath.Join(root, "Example Pack")
	writeContentPackManifest(t, filepath.Join(container, "PartA"), "Author.ModA", 123)
	writeContentPackManifest(t, filepath.Join(container, "PartB"), "Author.ModB", 123)

	list, err := NewScanner().Scan(ScanOptions{
		ModsRoot:  root,
	})
	must.NoError(err)
	must.Len(list, 1)
	must.Equal("pack:nexus:123", list[0].Manifest.UniqueID)
}

func writeContentPackManifest(t *testing.T, dir, uid string, nexusID int) {
	t.Helper()
	must := require.New(t)
	must.NoError(os.MkdirAll(dir, 0o755))
	manifest := fmt.Sprintf(
		`{"Name":"P","Author":"A","Version":"1.0.0","UniqueID":"%s","ContentPackFor":{"UniqueID":"Target.Mod"},"UpdateKeys":["Nexus:%d"]}`,
		uid,
		nexusID,
	)
	must.NoError(os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644))
}
