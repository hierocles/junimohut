package categories

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewService_SeedsDefaultsWhenMissing(t *testing.T) {
	must := require.New(t)

	path := filepath.Join(t.TempDir(), "categories.json")
	svc, err := NewService(path)
	must.NoError(err)
	cats := svc.List()
	must.Len(cats, 13)
	must.Equal("tag-qol", cats[0].ID)
	must.Equal("Quality of Life", cats[0].Name)
	must.Equal("tag-fashion-sense", cats[12].ID)
	must.Equal("Fashion Sense", cats[12].Name)
	_, err = os.Stat(path)
	must.NoError(err)
}

func TestNewService_MergesMissingDefaults(t *testing.T) {
	must := require.New(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "categories.json")
	legacy := store{
		Categories: DefaultCategories()[:12],
	}
	data, err := json.MarshalIndent(legacy, "", "  ")
	must.NoError(err)
	must.NoError(os.WriteFile(path, data, 0o644))

	svc, err := NewService(path)
	must.NoError(err)
	cats := svc.List()
	must.Len(cats, 13)
	var found bool
	for _, c := range cats {
		if c.ID == "tag-fashion-sense" {
			found = true
			must.Equal("Fashion Sense", c.Name)
		}
	}
	must.True(found)
}

func TestNewService_DoesNotReseedEmptyFile(t *testing.T) {
	must := require.New(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "categories.json")
	must.NoError(os.WriteFile(path, []byte(`{"categories":[]}`), 0o644))
	svc, err := NewService(path)
	must.NoError(err)
	must.Empty(svc.List())
}

func TestAssignModCategoryIDs(t *testing.T) {
	must := require.New(t)

	path := filepath.Join(t.TempDir(), "categories.json")
	svc, err := NewService(path)
	must.NoError(err)
	c, err := svc.Create("QoL", "#ff0000")
	must.NoError(err)
	must.NoError(svc.AssignMod(c.ID, "mod-1"))
	ids := svc.ModCategoryIDs("mod-1")
	must.Len(ids, 1)
	must.Equal(c.ID, ids[0])
}

func TestCategoryCRUD(t *testing.T) {
	must := require.New(t)

	path := filepath.Join(t.TempDir(), "categories.json")
	svc, err := NewService(path)
	must.NoError(err)
	c, err := svc.Create("QoL", "#ff0000")
	must.NoError(err)
	must.Equal("QoL", c.Name)
	must.NoError(svc.AssignMod(c.ID, "mod-1"))
	must.NoError(svc.SetVisibility(c.ID, false))
	hidden := svc.HiddenCategoryIDs()
	must.True(hidden[c.ID])
}
