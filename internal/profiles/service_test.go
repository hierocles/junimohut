package profiles

import (
	"os"
	"path/filepath"
	"testing"

	"junimohut/internal/mods"
	"junimohut/internal/platform"

	"github.com/stretchr/testify/require"
)

func TestProfileCreateAndEnable(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	dir := t.TempDir()
	svc, err := NewService(dir)
	must.NoError(err)
	p, err := svc.Create("Test")
	must.NoError(err)
	must.NoError(svc.SetActive(p.ID))
	must.NoError(svc.SetModEnabled("folder/mod::Author.Mod", false))
	enabled := svc.EnabledMods()
	must.False(enabled["folder/mod::Author.Mod"])
}

func TestAssembler(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	modsRoot := t.TempDir()
	active := filepath.Join(t.TempDir(), "Mods")
	modDir := filepath.Join(modsRoot, "TestMod")
	must.NoError(os.MkdirAll(modDir, 0o755))
	manifest := `{"Name":"T","Author":"A","Version":"1.0.0","UniqueID":"A.T"}`
	must.NoError(os.WriteFile(filepath.Join(modDir, "manifest.json"), []byte(manifest), 0o644))

	scanner := mods.NewScanner()
	list, err := scanner.Scan(mods.ScanOptions{ModsRoot: modsRoot})
	must.NoError(err)
	must.Len(list, 1)

	asm := NewAssembler(active, modsRoot)
	must.NoError(asm.Assemble(list, map[string]bool{list[0].ID: true}))
	linkPath := filepath.Join(active, "TestMod")
	must.True(platform.IsManagedModLink(linkPath, modsRoot))
}

func TestAssemblerRemovesDisabledLinks(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	modsRoot := t.TempDir()
	active := filepath.Join(t.TempDir(), "Mods")
	modDir := filepath.Join(modsRoot, "TestMod")
	must.NoError(os.MkdirAll(modDir, 0o755))
	manifest := `{"Name":"T","Author":"A","Version":"1.0.0","UniqueID":"A.T"}`
	must.NoError(os.WriteFile(filepath.Join(modDir, "manifest.json"), []byte(manifest), 0o644))

	scanner := mods.NewScanner()
	list, err := scanner.Scan(mods.ScanOptions{ModsRoot: modsRoot})
	must.NoError(err)
	must.Len(list, 1)

	asm := NewAssembler(active, modsRoot)
	must.NoError(asm.Assemble(list, map[string]bool{list[0].ID: true}))
	must.NoError(asm.Assemble(list, map[string]bool{list[0].ID: false}))
	_, err = os.Stat(filepath.Join(active, "TestMod"))
	must.True(os.IsNotExist(err))
}
