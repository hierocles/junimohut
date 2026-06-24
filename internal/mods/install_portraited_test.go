package mods

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeTestModFolder(t *testing.T, modsRoot, folder, uid, name string) {
	t.Helper()
	must := require.New(t)
	modDir := filepath.Join(modsRoot, filepath.FromSlash(folder))
	must.NoError(os.MkdirAll(modDir, 0o755))
	manifest := `{"Name":"` + name + `","Author":"A","Version":"1.0.0","UniqueID":"` + uid + `","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`
	must.NoError(os.WriteFile(filepath.Join(modDir, "manifest.json"), []byte(manifest), 0o644))
}

func TestPreviewInstallOverwritesMultiManifestBundle(t *testing.T) {
	must := require.New(t)

	modsRoot := t.TempDir()
	writeTestModFolder(t, modsRoot, "[CP] Portraited Changing Skies - Beta", "Kana.PortraitedChangingSkies", "[CP] Portraited Changing Skies - Beta")
	writeTestModFolder(t, modsRoot, "[DDF] Portraited Changing Skies - Beta", "Kana.PortraitedChangingSkies.DDF", "[DDF] Portraited Changing Skies - Beta")

	archivePath := filepath.Join(t.TempDir(), "portraited-changing-skies.zip")
	writePortraitedChangingSkiesZip(t, archivePath)

	previews, err := PreviewInstallOverwrites([]string{archivePath}, modsRoot, nil)
	must.NoError(err)
	must.Len(previews, 1)
	must.Equal(overwritePreviewStateConfirm, previews[0].State)
	must.GreaterOrEqual(len(previews[0].Candidates), 2)
}

func TestResolveInstallDestinationDetectsUniqueIDConflict(t *testing.T) {
	must := require.New(t)

	modsRoot := t.TempDir()
	writeTestModFolder(t, modsRoot, "[CP] Portraited Changing Skies - Beta", "Kana.PortraitedChangingSkies", "[CP] Portraited Changing Skies - Beta")

	archivePath := filepath.Join(t.TempDir(), "portraited-changing-skies.zip")
	writePortraitedChangingSkiesZip(t, archivePath)

	installer := NewInstaller(modsRoot)
	results, err := installer.InstallArchive(archivePath)
	must.NoError(err)
	must.NotEmpty(results)
	must.NotEmpty(results[0].Error)
	must.Contains(results[0].Error, "merge target")
}

func writePortraitedChangingSkiesZip(t *testing.T, path string) {
	t.Helper()
	cpManifest := `{"Name":"[CP] Portraited Changing Skies - Beta","Author":"A","Version":"3.0.1-beta","UniqueID":"Kana.PortraitedChangingSkies","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`
	ddfManifest := `{"Name":"[DDF] Portraited Changing Skies - Beta","Author":"A","Version":"3.0.1-beta","UniqueID":"Kana.PortraitedChangingSkies.DDF","ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`
	writeTestZip(t, path, map[string]string{
		"Portraited Changing Skies - Beta/[CP] Portraited Changing Skies/manifest.json":  cpManifest,
		"Portraited Changing Skies - Beta/[CP] Portraited Changing Skies/content.json":   `{}`,
		"Portraited Changing Skies - Beta/[DDF] Portraited Changing Skies/manifest.json": ddfManifest,
		"Portraited Changing Skies - Beta/[DDF] Portraited Changing Skies/content.json":  `{}`,
	})
}
