package mods

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const petFaceliftArchive = `C:/Users/dylan/AppData/Roaming/JunimoHut/downloads/(AT) Pet Facelift-9097-1-1-0-1711053648.zip`

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	must := require.New(t)
	in, err := os.Open(src)
	must.NoError(err)
	defer in.Close()
	out, err := os.Create(dst)
	must.NoError(err)
	defer out.Close()
	_, err = io.Copy(out, in)
	must.NoError(err)
}

func TestInstallArchivePetFacelift(t *testing.T) {
	if _, err := os.Stat(petFaceliftArchive); err != nil {
		t.Skip("Pet Facelift archive not available locally")
	}

	must := require.New(t)
	modsRoot := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "pet-facelift.zip")
	copyFile(t, petFaceliftArchive, archivePath)

	installer := NewInstaller(modsRoot)
	results, err := installer.InstallArchive(archivePath)
	must.NoError(err)
	must.Len(results, 1)
	must.Empty(results[0].Error, results[0].Error)
	must.Contains(results[0].FolderPath, "Pet Facelift")

	dest := filepath.Join(modsRoot, results[0].FolderPath)
	_, err = os.Stat(filepath.Join(dest, "manifest.json"))
	must.NoError(err)
	_, err = os.Stat(filepath.Join(dest, "Textures", "Cats", "texture_1.png"))
	must.NoError(err)
}

func TestPreviewInstallPetFacelift(t *testing.T) {
	if _, err := os.Stat(petFaceliftArchive); err != nil {
		t.Skip("Pet Facelift archive not available locally")
	}

	must := require.New(t)
	archivePath := petFaceliftArchive

	names, err := PreviewInstallNames([]string{archivePath})
	must.NoError(err)
	must.Len(names, 1)

	deps, err := PreviewInstallDependencies([]string{archivePath}, nil)
	must.NoError(err)
	if len(deps) > 0 {
		t.Logf("dependency previews (Alternative Textures may be missing): %+v", deps)
	}

	overwrites, err := PreviewInstallOverwrites([]string{archivePath}, t.TempDir(), nil)
	must.NoError(err)
	must.Empty(overwrites)
}
