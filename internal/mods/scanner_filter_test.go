package mods

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterByCategoriesShowsAllWhenEveryTagVisible(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	mods := []Mod{
		{ID: "a::A.A", CategoryIDs: []string{"tag1"}},
		{ID: "b::B.B", CategoryIDs: []string{}},
	}
	categories := []CategoryVisibility{
		{ID: "tag1", Visible: true},
		{ID: "tag2", Visible: true},
	}

	got := FilterByCategories(mods, categories)
	must.Len(got, 2)
}

func TestFilterByCategoriesNarrowsToVisibleTags(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	mods := []Mod{
		{ID: "a::A.A", CategoryIDs: []string{"tag1"}},
		{ID: "b::B.B", CategoryIDs: []string{"tag2"}},
		{ID: "c::C.C", CategoryIDs: []string{"tag1", "tag2"}},
		{ID: "d::D.D", CategoryIDs: []string{}},
	}
	categories := []CategoryVisibility{
		{ID: "tag1", Visible: true},
		{ID: "tag2", Visible: false},
	}

	got := FilterByCategories(mods, categories)
	must.Len(got, 2)
	must.Equal("a::A.A", got[0].ID)
	must.Equal("c::C.C", got[1].ID)
}

func TestFilterByCategoriesDoesNotDuplicateMultiTagMods(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	mods := []Mod{
		{ID: "cp::Pathoschild.ContentPatcher", CategoryIDs: []string{"frameworks"}},
		{ID: "f2::Author.F2", CategoryIDs: []string{"ui"}},
		{ID: "fp::aedenthorn.FarmerPortraits", CategoryIDs: []string{"frameworks", "ui"}},
	}
	categories := []CategoryVisibility{
		{ID: "frameworks", Visible: false},
		{ID: "ui", Visible: true},
	}

	got := FilterByCategories(mods, categories)
	must.Len(got, 2)
	must.Equal("f2::Author.F2", got[0].ID)
	must.Equal("fp::aedenthorn.FarmerPortraits", got[1].ID)
}

func TestFilterByCategoriesShowsAllWhenNoTagsVisible(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	mods := []Mod{{ID: "a::A.A", CategoryIDs: []string{"tag1"}}}
	categories := []CategoryVisibility{{ID: "tag1", Visible: false}}

	got := FilterByCategories(mods, categories)
	must.Len(got, 1)
}
