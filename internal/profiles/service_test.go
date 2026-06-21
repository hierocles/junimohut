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

func TestAssemblerRemovesNestedStaleLinks(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	modsRoot := t.TempDir()
	active := filepath.Join(t.TempDir(), "Mods")
	must.NoError(os.MkdirAll(active, 0o755))

	deletedTarget := filepath.Join(modsRoot, "Wallet Tools", "Wallet Tools")
	must.NoError(os.MkdirAll(deletedTarget, 0o755))
	linkPath := filepath.Join(active, "Wallet Tools", "Wallet Tools")
	must.NoError(platform.LinkDir(linkPath, deletedTarget))
	must.NoError(os.RemoveAll(filepath.Join(modsRoot, "Wallet Tools")))

	asm := NewAssembler(active, modsRoot)
	must.NoError(asm.Assemble(nil, nil))
	_, err := os.Stat(filepath.Join(active, "Wallet Tools"))
	must.True(os.IsNotExist(err))
}

func TestAssemblerPreservesUnmanagedModFolders(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	modsRoot := t.TempDir()
	active := filepath.Join(t.TempDir(), "Mods")
	unmanaged := filepath.Join(active, "ManualMod")
	must.NoError(os.MkdirAll(unmanaged, 0o755))
	manifest := `{"Name":"Manual","Author":"A","Version":"1.0.0","UniqueID":"A.Manual"}`
	must.NoError(os.WriteFile(filepath.Join(unmanaged, "manifest.json"), []byte(manifest), 0o644))

	asm := NewAssembler(active, modsRoot)
	must.NoError(asm.Assemble(nil, nil))
	_, err := os.Stat(filepath.Join(unmanaged, "manifest.json"))
	must.NoError(err)
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

func TestAssemblerMultiModContainer(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	modsRoot := t.TempDir()
	active := filepath.Join(t.TempDir(), "Mods")
	container := "Example Pack"

	writeNestedMod := func(name, uid string) mods.Mod {
		t.Helper()
		dir := filepath.Join(modsRoot, container, name)
		must.NoError(os.MkdirAll(dir, 0o755))
		manifest := `{"Name":"` + name + `","Author":"A","Version":"1.0.0","UniqueID":"` + uid + `"}`
		must.NoError(os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644))
		return mods.Mod{
			ID:           mods.ModID(container+"/"+name, uid),
			FolderPath:   container + "/" + name,
			AbsolutePath: dir,
		}
	}

	modsList := []mods.Mod{
		writeNestedMod("[CP] Example", "Author.ExampleCP"),
		writeNestedMod("[CC] Example", "Author.ExampleCC"),
		writeNestedMod("Example Code", "Author.ExampleCode"),
	}

	enabled := map[string]bool{}
	for _, m := range modsList {
		enabled[m.ID] = true
	}

	asm := NewAssembler(active, modsRoot)
	must.NoError(asm.Assemble(modsList, enabled))

	for _, m := range modsList {
		linkPath := filepath.Join(active, filepath.FromSlash(m.FolderPath))
		must.True(platform.IsManagedModLink(linkPath, modsRoot), "missing link for %s", m.FolderPath)
	}

	// No-op re-assemble must preserve all nested junctions (regression for stale-link key mismatch).
	must.NoError(asm.Assemble(modsList, enabled))
	for _, m := range modsList {
		linkPath := filepath.Join(active, filepath.FromSlash(m.FolderPath))
		must.True(platform.IsManagedModLink(linkPath, modsRoot), "link removed after re-assemble: %s", m.FolderPath)
	}
}
