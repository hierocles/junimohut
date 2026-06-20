package mods

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func stringLightsATManifest() string {
	return `{"Name":"[AT] Shyzie's String Lights","Author":"Shyzie","Version":"2.0.1","UniqueID":"Shyzie.StringLights.AT","UpdateKeys":["Nexus:20973"],"ContentPackFor":{"UniqueID":"PeacefulEnd.AlternativeTextures","MinimumVersion":"3.3.2"}}`
}

func stringLightsCPManifest() string {
	return `{
	"Name": "[CP] Shyzie's String Lights",
	"Author": "Shyzie",
	"Version": "2.0.1",
	"UniqueID": "Shyzie.StringLights.CP",
	"UpdateKeys": ["Nexus:20973"],
	"ContentPackFor": {
		"UniqueID": "Pathoschild.ContentPatcher",
	},
	"Dependencies": []
}`
}

func writeStringLightsBundle(t *testing.T, root string) {
	t.Helper()
	must := require.New(t)

	wrapper := filepath.Join(root, "[CP][AT] Shyzie's String Lights")
	for _, sub := range []struct {
		dir, manifest string
	}{
		{"[AT] Shyzie's String Lights", stringLightsATManifest()},
		{"[CP] Shyzie's String Lights", stringLightsCPManifest()},
	} {
		dir := filepath.Join(wrapper, sub.dir)
		must.NoError(os.MkdirAll(dir, 0o755))
		must.NoError(os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(sub.manifest), 0o644))
	}
}

func TestInstallArchiveATCPVariantSplit(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "string-lights.zip")
	writeTestZip(t, archivePath, map[string]string{
		"[CP][AT] Shyzie's String Lights/[AT] Shyzie's String Lights/manifest.json": stringLightsATManifest(),
		"[CP][AT] Shyzie's String Lights/[CP] Shyzie's String Lights/manifest.json": stringLightsCPManifest(),
	})

	installer := NewInstaller(root)
	results, err := installer.InstallArchive(archivePath)
	must.NoError(err)
	must.Len(results, 2)

	names := map[string]bool{}
	for _, r := range results {
		must.Empty(r.Error)
		names[r.FolderPath] = true
	}
	must.True(names["[AT] Shyzie's String Lights"])
	must.True(names["[CP] Shyzie's String Lights"])

	_, err = os.Stat(filepath.Join(root, "[AT] Shyzie's String Lights", "manifest.json"))
	must.NoError(err)
	_, err = os.Stat(filepath.Join(root, "[CP] Shyzie's String Lights", "manifest.json"))
	must.NoError(err)
	_, err = os.Stat(filepath.Join(root, "[CP][AT] Shyzie's String Lights"))
	must.True(os.IsNotExist(err))
}

func TestUpdateModATCPVariantSplit(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	atDir := filepath.Join(root, "[AT] Shyzie's String Lights")
	must.NoError(os.MkdirAll(atDir, 0o755))
	must.NoError(os.WriteFile(filepath.Join(atDir, "manifest.json"), []byte(stringLightsATManifest()), 0o644))
	must.NoError(os.WriteFile(filepath.Join(atDir, "old.txt"), []byte("old"), 0o644))

	archivePath := filepath.Join(t.TempDir(), "update.zip")
	writeTestZip(t, archivePath, map[string]string{
		"[CP][AT] Shyzie's String Lights/[AT] Shyzie's String Lights/manifest.json": stringLightsATManifest(),
		"[CP][AT] Shyzie's String Lights/[AT] Shyzie's String Lights/new.txt":      "new",
		"[CP][AT] Shyzie's String Lights/[CP] Shyzie's String Lights/manifest.json": stringLightsCPManifest(),
	})

	installer := NewInstaller(root)
	must.NoError(installer.UpdateMod("[AT] Shyzie's String Lights", archivePath, true))

	_, err := os.Stat(filepath.Join(atDir, "old.txt"))
	must.True(os.IsNotExist(err))
	data, err := os.ReadFile(filepath.Join(atDir, "new.txt"))
	must.NoError(err)
	must.Equal("new", string(data))
}

func TestVariantSplitUnitsPreservesChestDeco(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	writeChestDecoSiblingPack(t, root)

	manifests, err := findAllManifests(root)
	must.NoError(err)
	manifests = FilterRootManifests(manifests, root)
	units, err := resolveInstallUnits(root, manifests)
	must.NoError(err)
	must.Len(units, 1)
	must.Equal("(AT) Chest Deco", units[0].destName)
}

func TestResolveInstallUnitsStringLightsSplit(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	writeStringLightsBundle(t, root)

	manifests, err := findAllManifests(root)
	must.NoError(err)
	manifests = FilterRootManifests(manifests, root)
	units, err := resolveInstallUnits(root, manifests)
	must.NoError(err)
	must.Len(units, 2)
}

func TestInstallRealStringLightsZip(t *testing.T) {
	archivePath := `C:\Users\dylan\AppData\Roaming\JunimoHut\downloads\(CP x AT) Shyzie's String Lights-20973-2-0-1-1754357872.zip`
	if _, err := os.Stat(archivePath); err != nil {
		t.Skip("real zip not available")
	}

	must := require.New(t)
	root := t.TempDir()
	installer := NewInstaller(root)
	results, err := installer.InstallArchive(archivePath)
	must.NoError(err)
	must.Len(results, 2)

	scanner := NewScanner()
	mods, err := scanner.Scan(ScanOptions{ModsRoot: root})
	must.NoError(err)
	must.Len(mods, 2)
}

func sveCodeManifest() string {
	return `{"Name":"Stardew Valley Expanded","Author":"FlashShifter","Version":"1.15.11","UniqueID":"FlashShifter.SVECode","EntryDll":"StardewValleyExpanded.dll","UpdateKeys":["Nexus:3753"]}`
}

func sveCPManifest() string {
	return `{
	"Name":"Stardew Valley Expanded",
	"UniqueID":"FlashShifter.StardewValleyExpandedCP",
	"Version":"1.15.11",
	"UpdateKeys":["Nexus:3753"],
	"ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"},
	"Dependencies":[
		{"UniqueID":"FlashShifter.SVE-FTM","IsRequired":true},
		{"UniqueID":"FlashShifter.SVECode","IsRequired":true}
	]
}`
}

func sveFTMManifest() string {
	return `{"Name":"Stardew Valley Expanded Farm Type Manager","UniqueID":"FlashShifter.SVE-FTM","Version":"1.15.11","UpdateKeys":["Nexus:3753"],"ContentPackFor":{"UniqueID":"Esca.FarmTypeManager","MinimumVersion":"1.19"}}`
}

func writeSVEBundle(t *testing.T, root string) {
	t.Helper()
	must := require.New(t)

	wrapper := filepath.Join(root, "Stardew Valley Expanded")
	for _, sub := range []struct {
		dir, manifest string
	}{
		{"Stardew Valley Expanded Code", sveCodeManifest()},
		{"[CP] Stardew Valley Expanded", sveCPManifest()},
		{"[FTM] Stardew Valley Expanded", sveFTMManifest()},
	} {
		dir := filepath.Join(wrapper, sub.dir)
		must.NoError(os.MkdirAll(dir, 0o755))
		must.NoError(os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(sub.manifest), 0o644))
	}
}

func TestVariantSplitDoesNotSplitExpansionBundle(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	writeSVEBundle(t, root)

	manifests, err := findAllManifests(root)
	must.NoError(err)
	manifests = FilterRootManifests(manifests, root)
	units, err := resolveInstallUnits(root, manifests)
	must.NoError(err)
	must.Len(units, 1)
	must.Equal("Stardew Valley Expanded", units[0].destName)
}

func TestInstallArchiveExpansionBundle(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "sve.zip")
	writeTestZip(t, archivePath, map[string]string{
		"Stardew Valley Expanded/Stardew Valley Expanded Code/manifest.json":  sveCodeManifest(),
		"Stardew Valley Expanded/[CP] Stardew Valley Expanded/manifest.json":  sveCPManifest(),
		"Stardew Valley Expanded/[FTM] Stardew Valley Expanded/manifest.json": sveFTMManifest(),
	})

	installer := NewInstaller(root)
	results, err := installer.InstallArchive(archivePath)
	must.NoError(err)
	must.Len(results, 3)

	dest := filepath.Join(root, "Stardew Valley Expanded")
	for _, sub := range []string{"Stardew Valley Expanded Code", "[CP] Stardew Valley Expanded", "[FTM] Stardew Valley Expanded"} {
		_, err := os.Stat(filepath.Join(dest, sub, "manifest.json"))
		must.NoError(err, "missing %s", sub)
	}

	scanner := NewScanner()
	mods, err := scanner.Scan(ScanOptions{ModsRoot: root})
	must.NoError(err)
	must.Len(mods, 3)
}

func TestInstallRealSVEZip(t *testing.T) {
	archivePath := `C:\Users\dylan\AppData\Roaming\JunimoHut\downloads\-Stardew Valley Expanded--3753-1-15-11-1751325459.zip`
	if _, err := os.Stat(archivePath); err != nil {
		t.Skip("real SVE zip not available")
	}

	must := require.New(t)
	root := t.TempDir()
	installer := NewInstaller(root)
	_, err := installer.InstallArchive(archivePath)
	must.NoError(err)

	dest := filepath.Join(root, "Stardew Valley Expanded")
	for _, sub := range []string{"Stardew Valley Expanded Code", "[CP] Stardew Valley Expanded", "[FTM] Stardew Valley Expanded"} {
		_, err := os.Stat(filepath.Join(dest, sub, "manifest.json"))
		must.NoError(err)
	}
}

func chestDecoSiblingManifest(name, uid string) string {
	return `{"Name":"Chest Deco : ` + name + `","Author":"guxelbit","Version":"1.3","UniqueID":"` + uid + `","UpdateKeys":["Nexus:20384"],"ContentPackFor":{"UniqueID":"PeacefulEnd.AlternativeTextures","MinimumVersion":"6.9.0"}}`
}

func writeChestDecoSiblingPack(t *testing.T, root string) {
	t.Helper()
	must := require.New(t)

	pack := filepath.Join(root, "(AT) Chest Deco")
	subs := []struct {
		folder string
		uid    string
	}{
		{"animal products", "guxelchestdeco.animal"},
		{"artisan goods", "guxelchestdeco.artisan"},
		{"fishes", "guxelchestdeco.fishes"},
		{"misc", "guxelchestdeco.misc"},
		{"resources", "guxelchestdeco.resources"},
		{"veggies", "guxelchestdeco.veggies"},
	}
	for _, sub := range subs {
		dir := filepath.Join(pack, sub.folder)
		must.NoError(os.MkdirAll(dir, 0o755))
		must.NoError(os.WriteFile(
			filepath.Join(dir, "manifest.json"),
			[]byte(chestDecoSiblingManifest(sub.folder, sub.uid)),
			0o644,
		))
	}
}

func TestInstallArchiveSiblingContentPacks(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "chest-deco.zip")
	files := map[string]string{
		"(AT) Chest Deco/animal products/manifest.json": chestDecoSiblingManifest("animal products", "guxelchestdeco.animal"),
		"(AT) Chest Deco/artisan goods/manifest.json":   chestDecoSiblingManifest("artisan goods", "guxelchestdeco.artisan"),
		"(AT) Chest Deco/fishes/manifest.json":          chestDecoSiblingManifest("fishes", "guxelchestdeco.fishes"),
		"(AT) Chest Deco/misc/manifest.json":            chestDecoSiblingManifest("misc", "guxelchestdeco.misc"),
		"(AT) Chest Deco/resources/manifest.json":       chestDecoSiblingManifest("resources", "guxelchestdeco.resources"),
		"(AT) Chest Deco/veggies/manifest.json":         chestDecoSiblingManifest("veggies", "guxelchestdeco.veggies"),
	}
	writeTestZip(t, archivePath, files)

	installer := NewInstaller(root)
	results, err := installer.InstallArchive(archivePath)
	must.NoError(err)
	must.Len(results, 1)
	must.Equal("(AT) Chest Deco", results[0].FolderPath)
	must.Equal("Chest Deco", results[0].Name)
	must.Contains(results[0].ModID, PackUniqueIDPrefix+"20384")

	dest := filepath.Join(root, "(AT) Chest Deco")
	for _, sub := range []string{"animal products", "artisan goods", "fishes", "misc", "resources", "veggies"} {
		_, err := os.Stat(filepath.Join(dest, sub, "manifest.json"))
		must.NoError(err, "missing subfolder %s", sub)
	}
}

func TestUpdateModSiblingContentPacks(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	writeChestDecoSiblingPack(t, root)

	archivePath := filepath.Join(t.TempDir(), "update.zip")
	writeTestZip(t, archivePath, map[string]string{
		"(AT) Chest Deco/animal products/manifest.json": chestDecoSiblingManifest("animal products", "guxelchestdeco.animal"),
		"(AT) Chest Deco/artisan goods/manifest.json":   chestDecoSiblingManifest("artisan goods", "guxelchestdeco.artisan"),
		"(AT) Chest Deco/fishes/manifest.json":          chestDecoSiblingManifest("fishes", "guxelchestdeco.fishes"),
		"(AT) Chest Deco/misc/manifest.json":            chestDecoSiblingManifest("misc", "guxelchestdeco.misc"),
		"(AT) Chest Deco/resources/manifest.json":       chestDecoSiblingManifest("resources", "guxelchestdeco.resources"),
		"(AT) Chest Deco/veggies/manifest.json":         chestDecoSiblingManifest("veggies", "guxelchestdeco.veggies"),
		"(AT) Chest Deco/veggies/new-file.txt":          "updated",
	})

	installer := NewInstaller(root)
	must.NoError(installer.UpdateMod("(AT) Chest Deco", archivePath, true))

	data, err := os.ReadFile(filepath.Join(root, "(AT) Chest Deco", "veggies", "new-file.txt"))
	must.NoError(err)
	must.Equal("updated", string(data))
}

func TestCollapseSiblingPacks(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	writeChestDecoSiblingPack(t, root)

	scanner := NewScanner()
	mods, err := scanner.Scan(ScanOptions{ModsRoot: root})
	must.NoError(err)
	must.Len(mods, 1)
	must.Equal("(AT) Chest Deco", mods[0].FolderPath)
	must.Equal("Chest Deco", mods[0].Manifest.Name)
	must.Equal(PackUniqueIDPrefix+"20384", mods[0].Manifest.UniqueID)
}

func TestPackEnableMigration(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	writeChestDecoSiblingPack(t, root)

	enabled := map[string]bool{
		"(AT) Chest Deco/animal products::guxelchestdeco.animal": true,
		"(AT) Chest Deco/fishes::guxelchestdeco.fishes":          false,
	}
	packID := ModID("(AT) Chest Deco", PackUniqueIDPrefix+"20384")

	mods := CollapseSiblingPacks([]Mod{
		{ID: "(AT) Chest Deco/animal products::guxelchestdeco.animal", FolderPath: "(AT) Chest Deco/animal products", Manifest: Manifest{UniqueID: "guxelchestdeco.animal", ContentPackFor: &ContentPackFor{UniqueID: "PeacefulEnd.AlternativeTextures"}, UpdateKeys: []string{"Nexus:20384"}}},
		{ID: "(AT) Chest Deco/fishes::guxelchestdeco.fishes", FolderPath: "(AT) Chest Deco/fishes", Manifest: Manifest{UniqueID: "guxelchestdeco.fishes", ContentPackFor: &ContentPackFor{UniqueID: "PeacefulEnd.AlternativeTextures"}, UpdateKeys: []string{"Nexus:20384"}}},
	}, root, enabled)

	must.Len(mods, 1)
	must.True(mods[0].Enabled)

	MigratePackEnableState(enabled, packID, false)
	must.False(enabled[packID])
	must.NotContains(enabled, "(AT) Chest Deco/animal products::guxelchestdeco.animal")
}

func TestResolveInstallUnitsSingleWrapper(t *testing.T) {
	must := require.New(t)

	root := t.TempDir()
	writeChestDecoSiblingPack(t, root)

	manifests, err := findAllManifests(root)
	must.NoError(err)
	manifests = FilterRootManifests(manifests, root)
	must.Len(manifests, 6)

	units, err := resolveInstallUnits(root, manifests)
	must.NoError(err)
	must.Len(units, 1)
	must.Equal("(AT) Chest Deco", units[0].destName)
}
