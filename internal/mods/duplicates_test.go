package mods

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectDuplicateMods(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	plain := writeSimpleModWithAsset(t, root, "ConvenientInventory", "Author.ConvenientInventory", "Convenient Inventory", "Mod.dll")
	dup := writeSimpleModWithAsset(t, root, "ConvenientInventory_20260620_165612", "Author.ConvenientInventory", "Convenient Inventory", "Mod.dll")
	other := writeSimpleModWithAsset(t, root, "OtherMod", "Author.Other", "Other Mod", "Mod.dll")

	groups := DetectDuplicateMods([]Mod{plain, dup, other})
	must.Len(groups, 1)
	must.Equal("Author.ConvenientInventory", groups[0].UniqueID)
	must.Equal("ConvenientInventory", groups[0].Canonical)
	must.ElementsMatch([]string{"ConvenientInventory", "ConvenientInventory_20260620_165612"}, groups[0].Folders)
}

func TestHasTimestampInstallSuffix(t *testing.T) {
	must := require.New(t)

	must.True(HasTimestampInstallSuffix("ConvenientInventory_20260620_165612"))
	must.False(HasTimestampInstallSuffix("ConvenientInventory"))
	must.False(HasTimestampInstallSuffix("[CP] Seasonal Open Windows"))
}

func TestPickCanonicalDuplicateFolderPrefersPlainName(t *testing.T) {
	must := require.New(t)

	canonical := pickCanonicalDuplicateFolder([]string{
		"ConvenientInventory_20260620_190945",
		"ConvenientInventory",
	})
	must.Equal("ConvenientInventory", canonical)
}

func TestDuplicateGroupForFolder(t *testing.T) {
	must := require.New(t)

	groups := []DuplicateModGroup{{
		UniqueID:  "Author.Mod",
		Folders:   []string{"ModA", "ModA_20260620_120000"},
		Canonical: "ModA",
	}}
	group, ok := DuplicateGroupForFolder(groups, "ModA_20260620_120000")
	must.True(ok)
	must.Equal("ModA", group.Canonical)
}

func TestPreviewManifestArchiveMergeFindsDiskMatch(t *testing.T) {
	must := require.New(t)

	modsRoot := t.TempDir()
	writeSimpleModWithAsset(t, modsRoot, "ConvenientInventory", "Author.ConvenientInventory", "Convenient Inventory", "Mod.dll")

	archivePath := filepath.Join(t.TempDir(), "convenient-inventory.zip")
	manifest := `{"Name":"Convenient Inventory","Author":"A","Version":"2.0.0","UniqueID":"Author.ConvenientInventory","EntryDll":"Mod.dll"}`
	writeTestZip(t, archivePath, map[string]string{
		"ConvenientInventory/manifest.json": manifest,
		"ConvenientInventory/Mod.dll":        "v2",
	})

	targets, err := ResolveInstallMergeTargets(archivePath, nil, modsRoot, nil)
	must.NoError(err)
	must.Empty(targets)

	targets, err = ResolveInstallMergeTargets(archivePath, map[string][]string{
		archivePath: {"ConvenientInventory"},
	}, modsRoot, nil)
	must.NoError(err)
	must.Equal([]string{"ConvenientInventory"}, targets)
}
