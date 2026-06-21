package profiles

import (
	"os"
	"path/filepath"
	"testing"

	"junimohut/internal/platform"

	"github.com/stretchr/testify/require"
)

func TestScanUnmanagedModsExcludesManagedLinks(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	modsRoot := t.TempDir()
	active := filepath.Join(t.TempDir(), "Mods")
	modDir := filepath.Join(modsRoot, "TestMod")
	must.NoError(os.MkdirAll(modDir, 0o755))
	manifest := `{"Name":"Test","Author":"A","Version":"1.0.0","UniqueID":"A.Test"}`
	must.NoError(os.WriteFile(filepath.Join(modDir, "manifest.json"), []byte(manifest), 0o644))
	must.NoError(platform.LinkDir(filepath.Join(active, "TestMod"), modDir))

	list, err := ScanUnmanagedMods(active, modsRoot)
	must.NoError(err)
	must.Empty(list)
}

func TestScanUnmanagedModsReportsManualFolder(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	modsRoot := t.TempDir()
	active := filepath.Join(t.TempDir(), "Mods")
	manual := filepath.Join(active, "ManualMod")
	must.NoError(os.MkdirAll(manual, 0o755))
	manifest := `{"Name":"Manual","Author":"A","Version":"1.0.0","UniqueID":"A.Manual"}`
	must.NoError(os.WriteFile(filepath.Join(manual, "manifest.json"), []byte(manifest), 0o644))

	list, err := ScanUnmanagedMods(active, modsRoot)
	must.NoError(err)
	must.Len(list, 1)
	must.Equal("ManualMod", list[0].FolderName)
	must.Equal("Manual", list[0].Name)
	must.Equal("A.Manual", list[0].UniqueID)
}

func TestScanUnmanagedModsExcludesCoreMods(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	modsRoot := t.TempDir()
	active := filepath.Join(t.TempDir(), "Mods")
	core := filepath.Join(active, "ConsoleCommands")
	must.NoError(os.MkdirAll(core, 0o755))
	manifest := `{"Name":"Console Commands","Author":"Pathoschild","Version":"1.0.0","UniqueID":"Pathoschild.ConsoleCommands"}`
	must.NoError(os.WriteFile(filepath.Join(core, "manifest.json"), []byte(manifest), 0o644))

	list, err := ScanUnmanagedMods(active, modsRoot)
	must.NoError(err)
	must.Empty(list)
}

func TestScanUnmanagedModsExcludesManagedOnlyContainer(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	modsRoot := t.TempDir()
	active := filepath.Join(t.TempDir(), "Mods")
	target := filepath.Join(modsRoot, "Pack", "InnerMod")
	must.NoError(os.MkdirAll(target, 0o755))
	manifest := `{"Name":"Inner","Author":"A","Version":"1.0.0","UniqueID":"A.Inner"}`
	must.NoError(os.WriteFile(filepath.Join(target, "manifest.json"), []byte(manifest), 0o644))
	must.NoError(platform.LinkDir(filepath.Join(active, "Pack", "InnerMod"), target))

	list, err := ScanUnmanagedMods(active, modsRoot)
	must.NoError(err)
	must.Empty(list)
}

func TestScanUnmanagedModsReportsExternalSymlink(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	modsRoot := t.TempDir()
	active := filepath.Join(t.TempDir(), "Mods")
	external := t.TempDir()
	externalMod := filepath.Join(external, "ExternalMod")
	must.NoError(os.MkdirAll(externalMod, 0o755))
	must.NoError(platform.LinkDir(filepath.Join(active, "ExternalMod"), externalMod))

	list, err := ScanUnmanagedMods(active, modsRoot)
	must.NoError(err)
	must.Len(list, 1)
	must.Equal("ExternalMod", list[0].FolderName)
}
