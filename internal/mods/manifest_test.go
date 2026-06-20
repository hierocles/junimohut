package mods

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseManifest(t *testing.T) {
	must := require.New(t)

	dir := t.TempDir()
	manifest := `{
		"Name": "Test Mod",
		"Author": "Author",
		"Version": "1.0.0",
		"UniqueID": "Author.TestMod",
		"UpdateKeys": ["Nexus:123"]
	}`
	must.NoError(os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644))
	m, err := ParseManifest(filepath.Join(dir, "manifest.json"))
	must.NoError(err)
	must.Equal("Author.TestMod", m.UniqueID)
}

func TestParseManifestUTF8BOM(t *testing.T) {
	must := require.New(t)

	dir := t.TempDir()
	manifest := []byte(`{
		"Name": "Farmer Portraits",
		"Author": "Author",
		"Version": "1.0.0",
		"UniqueID": "Author.FarmerPortraits"
	}`)
	withBOM := append(append([]byte(nil), utf8BOM...), manifest...)
	must.NoError(os.WriteFile(filepath.Join(dir, "manifest.json"), withBOM, 0o644))
	m, err := ParseManifest(filepath.Join(dir, "manifest.json"))
	must.NoError(err)
	must.Equal("Farmer Portraits", m.Name)
}

func TestParseManifestFarmerPortraitsDependencies(t *testing.T) {
	must := require.New(t)

	dir := t.TempDir()
	manifest := `{
		"Name": "Farmer 2.0 ESWF NPCReactionOverhaul",
		"Author": "Salty",
		"Version": "1.8.4",
		"UniqueID": "Salty.Farmer2.0NPCReactionOverhaul",
		"ContentPackFor": {
			"UniqueID": "Pathoschild.ContentPatcher",
			"MinimumVersion": "2.4.4"
		},
		"Dependencies": [
			{ "UniqueID": "nihilistzsche.FashionSenseOutfits", "IsRequired": "false" },
			{ "UniqueID": "mistyspring.aedenthornFarmerPortraits", "IsRequired": "false" }
		]
	}`
	must.NoError(os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644))
	m, err := ParseManifest(filepath.Join(dir, "manifest.json"))
	must.NoError(err)
	must.Len(m.Dependencies, 2)
	must.False(bool(*m.Dependencies[0].IsRequired))
	must.Equal("mistyspring.aedenthornFarmerPortraits", m.Dependencies[1].UniqueID)
}

func TestParseManifestWithComments(t *testing.T) {
	must := require.New(t)

	dir := t.TempDir()
	manifest := `{
		// This is a line comment
		"Name": "Generic Mod Config Menu", /* inline block comment */
		"Author": "spacechase0",
		"Version": "1.16.0",
		"UniqueID": "spacechase0.GenericModConfigMenu",
		/* multi
		   line comment */
		"UpdateKeys": ["Nexus:5098"]
	}`
	must.NoError(os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644))
	m, err := ParseManifest(filepath.Join(dir, "manifest.json"))
	must.NoError(err)
	must.Equal("spacechase0.GenericModConfigMenu", m.UniqueID)
}

func TestScannerIgnoresNestedManifests(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	writeManifest(t, filepath.Join(root, "(AT) Chest Deco"))
	writeManifest(t, filepath.Join(root, "(AT) Chest Deco", "Textures", "Barrel"))

	scanner := NewScanner()
	mods, err := scanner.Scan(ScanOptions{ModsRoot: root})
	must.NoError(err)
	must.Len(mods, 1)
	must.Equal("(AT) Chest Deco", mods[0].FolderPath)
}

func TestScannerIgnoresHiddenFolders(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	hidden := filepath.Join(root, ".hidden", "ModA")
	visible := filepath.Join(root, "ModB")
	for _, d := range []string{hidden, visible} {
		must.NoError(os.MkdirAll(d, 0o755))
		manifest := `{"Name":"M","Author":"A","Version":"1.0.0","UniqueID":"A.M"}`
		must.NoError(os.WriteFile(filepath.Join(d, "manifest.json"), []byte(manifest), 0o644))
	}
	scanner := NewScanner()
	mods, err := scanner.Scan(ScanOptions{ModsRoot: root, IgnoreHiddenFolders: true})
	must.NoError(err)
	must.Len(mods, 1)
}

func TestParseManifestTrailingComma(t *testing.T) {
	must := require.New(t)

	dir := t.TempDir()
	manifest := `{
	"Name": "[CP] Shyzie's String Lights",
	"UniqueID": "Shyzie.StringLights.CP",
	"UpdateKeys": ["Nexus:20973"],
	"ContentPackFor": {
		"UniqueID": "Pathoschild.ContentPatcher",
	},
	"Dependencies": [],
}`
	must.NoError(os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644))
	m, err := ParseManifest(filepath.Join(dir, "manifest.json"))
	must.NoError(err)
	must.Equal("Shyzie.StringLights.CP", m.UniqueID)
	must.Equal("Pathoschild.ContentPatcher", m.ContentPackFor.UniqueID)
}

func TestModID(t *testing.T) {
	must := require.New(t)

	id := ModID("folder/mod", "Author.Mod")
	must.Equal("folder/mod::Author.Mod", id)
}

func TestFilterMods(t *testing.T) {
	must := require.New(t)

	mods := []Mod{
		{Manifest: Manifest{Name: "Alpha"}, Enabled: true},
		{Manifest: Manifest{Name: "Beta"}, Enabled: false},
	}
	out := FilterMods(mods, "alpha", "none")
	must.Len(out, 1)
	out = FilterMods(mods, "", "enabled")
	must.Len(out, 1)
}
