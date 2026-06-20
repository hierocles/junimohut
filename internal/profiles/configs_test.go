package profiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigManagerSaveAndRestore(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	root := t.TempDir()
	profilesDir := filepath.Join(root, "profiles")
	svc, err := NewService(profilesDir)
	must.NoError(err)
	p, err := svc.Create("Gameplay")
	must.NoError(err)
	must.NoError(svc.SetActive(p.ID))

	modsRoot := filepath.Join(root, "library")
	modDir := filepath.Join(modsRoot, "SampleMod")
	must.NoError(os.MkdirAll(modDir, 0o755))
	configA := `{"Setting":"profile-a"}`
	must.NoError(os.WriteFile(filepath.Join(modDir, "config.json"), []byte(configA), 0o644))

	modID := "SampleMod::Author.Sample"
	mgr := NewConfigManager(profilesDir, svc)
	must.NoError(mgr.SaveModConfig(modsRoot, modID, "Author.Sample"))

	must.NoError(os.WriteFile(filepath.Join(modDir, "config.json"), []byte(`{"Setting":"mutated"}`), 0o644))
	must.NoError(mgr.RestoreModConfig(modsRoot, modID, "Author.Sample"))
	got, err := os.ReadFile(filepath.Join(modDir, "config.json"))
	must.NoError(err)
	must.Equal(configA, string(got))
}

func TestConfigManagerProfileSwitch(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	root := t.TempDir()
	profilesDir := filepath.Join(root, "profiles")
	svc, err := NewService(profilesDir)
	must.NoError(err)
	profileA, err := svc.Create("A")
	must.NoError(err)
	profileB, err := svc.Create("B")
	must.NoError(err)
	must.NoError(svc.SetActive(profileA.ID))

	modsRoot := filepath.Join(root, "library")
	modDir := filepath.Join(modsRoot, "SampleMod")
	must.NoError(os.MkdirAll(modDir, 0o755))

	modID := "SampleMod::Author.Sample"
	uniqueID := "Author.Sample"
	mgr := NewConfigManager(profilesDir, svc)

	writeConfig := func(value string) {
		t.Helper()
		must := require.New(t)
		must.NoError(os.WriteFile(filepath.Join(modDir, "config.json"), []byte(value), 0o644))
	}

	writeConfig(`{"Setting":"a"}`)
	must.NoError(mgr.SaveModConfig(modsRoot, modID, uniqueID))

	must.NoError(svc.SetActive(profileB.ID))
	writeConfig(`{"Setting":"b"}`)
	must.NoError(mgr.SaveModConfig(modsRoot, modID, uniqueID))

	must.NoError(svc.SetActive(profileA.ID))
	must.NoError(mgr.RestoreConfigs(modsRoot, map[string]string{modID: uniqueID}))
	got, _ := os.ReadFile(filepath.Join(modDir, "config.json"))
	must.Equal(`{"Setting":"a"}`, string(got))

	must.NoError(svc.SetActive(profileB.ID))
	must.NoError(mgr.RestoreConfigs(modsRoot, map[string]string{modID: uniqueID}))
	got, _ = os.ReadFile(filepath.Join(modDir, "config.json"))
	must.Equal(`{"Setting":"b"}`, string(got))
}

func TestConfigManagerUsesLibraryNotSymlinkPath(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	root := t.TempDir()
	profilesDir := filepath.Join(root, "profiles")
	svc, err := NewService(profilesDir)
	must.NoError(err)
	p, err := svc.Create("Default")
	must.NoError(err)
	must.NoError(svc.SetActive(p.ID))

	library := filepath.Join(root, "library")
	active := filepath.Join(root, "game", "Mods")
	modDir := filepath.Join(library, "SampleMod")
	must.NoError(os.MkdirAll(modDir, 0o755))
	must.NoError(os.MkdirAll(active, 0o755))
	config := `{"Setting":"library"}`
	must.NoError(os.WriteFile(filepath.Join(modDir, "config.json"), []byte(config), 0o644))
	if err := os.Symlink(modDir, filepath.Join(active, "SampleMod")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}

	modID := "SampleMod::Author.Sample"
	mgr := NewConfigManager(profilesDir, svc)
	must.NoError(mgr.SaveModConfig(library, modID, "Author.Sample"))

	saved, err := os.ReadFile(filepath.Join(profilesDir, p.ID, "configs", "Author.Sample", "config.json"))
	must.NoError(err)
	must.Equal(config, string(saved))
}
