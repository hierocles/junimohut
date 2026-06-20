package mods

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreviewInstallNamesCollidingCPVariants(t *testing.T) {
	must := require.New(t)

	archivePath := filepath.Join(t.TempDir(), "seasonal.zip")
	variants := []struct {
		folder string
		uid    string
	}{
		{"[CP] Seasonal Open Windows", "OB7.SOWindows"},
		{"[CP] Seasonal Open Windows - BIRCH", "OB7.SOWindows.birch"},
	}
	files := map[string]string{}
	for _, v := range variants {
		files[v.folder+"/manifest.json"] = seasonalOpenWindowsManifest(v.uid)
	}
	writeTestZip(t, archivePath, files)

	previews, err := PreviewInstallNames([]string{archivePath})
	must.NoError(err)
	must.Len(previews, 1)
	must.Len(previews[0].Mods, 2)
	must.True(InstallNameChoiceDiffers(
		previews[0].Mods[1].OfficialName,
		previews[0].Mods[1].FolderLabel,
	))
	must.False(InstallNameChoiceDiffers(
		previews[0].Mods[0].OfficialName,
		previews[0].Mods[0].FolderLabel,
	))
	must.True(previews[0].NeedsDisplayNameChoice)
}

func TestPreviewInstallNamesSingleModFolderMismatch(t *testing.T) {
	must := require.New(t)

	archivePath := filepath.Join(t.TempDir(), "expanded-storage.zip")
	manifest := `{"Name":"Expanded Storage","Author":"A","Version":"3.3.0","UniqueID":"furyx639.ExpandedStorage","EntryDll":"ExpandedStorage.dll"}`
	writeTestZip(t, archivePath, map[string]string{
		"ExpandedStorage/manifest.json": manifest,
	})

	previews, err := PreviewInstallNames([]string{archivePath})
	must.NoError(err)
	must.Len(previews, 1)
	must.Len(previews[0].Mods, 1)
	must.Equal("Expanded Storage", previews[0].Mods[0].OfficialName)
	must.Equal("ExpandedStorage", previews[0].Mods[0].FolderLabel)
	must.True(InstallNameChoiceDiffers(
		previews[0].Mods[0].OfficialName,
		previews[0].Mods[0].FolderLabel,
	))
	must.False(previews[0].NeedsDisplayNameChoice)
}

func TestInstallNameChoiceDiffers(t *testing.T) {
	must := require.New(t)
	must.True(InstallNameChoiceDiffers(
		"[CP] Seasonal Open Windows",
		"[CP] Seasonal Open Windows - BIRCH",
	))
	must.False(InstallNameChoiceDiffers("Lookup Anything", "Lookup Anything"))
}
