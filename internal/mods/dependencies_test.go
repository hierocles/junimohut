package mods

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func requiredBool(b bool) *flexBool {
	v := flexBool(b)
	return &v
}

func modWithDeps(uniqueID string, deps []ModDependency, contentPack *ContentPackFor) Mod {
	return Mod{
		ID: "folder/" + uniqueID,
		Manifest: Manifest{
			UniqueID:       uniqueID,
			Name:           uniqueID,
			Version:        "1.0.0",
			Dependencies:   deps,
			ContentPackFor: contentPack,
		},
	}
}

func providerMod(uniqueID, version string) Mod {
	return Mod{
		ID: "folder/" + uniqueID,
		Manifest: Manifest{
			UniqueID: uniqueID,
			Name:     uniqueID,
			Version:  version,
		},
		Enabled: true,
	}
}

func TestResolveDependenciesRequiredInstalled(t *testing.T) {
	must := require.New(t)

	mods := []Mod{
		providerMod("Author.Dependency", "2.0.0"),
		modWithDeps("Author.Mod", []ModDependency{
			{UniqueID: "Author.Dependency", MinimumVersion: "1.0.0", IsRequired: requiredBool(true)},
		}, nil),
	}
	out := ResolveDependencies(mods)
	must.Empty(out[1].DependencyIssues)
}

func TestResolveDependenciesRequiredMissing(t *testing.T) {
	must := require.New(t)

	mods := []Mod{
		modWithDeps("Author.Mod", []ModDependency{
			{UniqueID: "Author.Missing", MinimumVersion: "", IsRequired: requiredBool(true)},
		}, nil),
	}
	out := ResolveDependencies(mods)
	must.Len(out[0].DependencyIssues, 1)
	issue := out[0].DependencyIssues[0]
	must.Equal(DependencyMissing, issue.State)
	must.True(issue.IsRequired)
}

func TestCollectDependencyEntriesIncludesOptional(t *testing.T) {
	must := require.New(t)

	mod := modWithDeps("Author.Mod", []ModDependency{
		{UniqueID: "Author.Optional", IsRequired: requiredBool(false)},
		{UniqueID: "Author.Required", IsRequired: requiredBool(true)},
	}, nil)
	entries := collectDependencyEntries(mod)
	must.Len(entries, 2)
	optionalFound := false
	for _, e := range entries {
		if e.UniqueID == "Author.Optional" {
			optionalFound = true
			must.False(e.IsRequired)
		}
	}
	must.True(optionalFound)
}

func TestResolveDependenciesOptionalMissing(t *testing.T) {
	must := require.New(t)

	mods := []Mod{
		modWithDeps("Author.Mod", []ModDependency{
			{UniqueID: "Author.Optional", IsRequired: requiredBool(false)},
		}, nil),
	}
	out := ResolveDependencies(mods)
	must.Empty(out[0].DependencyIssues)
}

func TestResolveDependenciesOmittedIsRequired(t *testing.T) {
	must := require.New(t)

	mods := []Mod{
		modWithDeps("Author.Mod", []ModDependency{
			{UniqueID: "Author.Missing"},
		}, nil),
	}
	out := ResolveDependencies(mods)
	must.Len(out[0].DependencyIssues, 1)
	must.True(out[0].DependencyIssues[0].IsRequired)
}

func TestResolveDependenciesAlternativeTexturesOptionalIntegrations(t *testing.T) {
	must := require.New(t)

	mods := []Mod{
		modWithDeps("PeacefulEnd.AlternativeTextures", []ModDependency{
			{UniqueID: "spacechase0.MoreGiantCrops", IsRequired: requiredBool(false)},
			{UniqueID: "spacechase0.DynamicGameAssets", IsRequired: requiredBool(false)},
		}, nil),
	}
	out := ResolveDependencies(mods)
	must.Empty(out[0].DependencyIssues)
}

func TestResolveDependenciesVersionTooLow(t *testing.T) {
	must := require.New(t)

	mods := []Mod{
		providerMod("Author.Dependency", "1.0.0"),
		modWithDeps("Author.Mod", []ModDependency{
			{UniqueID: "Author.Dependency", MinimumVersion: "2.0.0", IsRequired: requiredBool(true)},
		}, nil),
	}
	out := ResolveDependencies(mods)
	must.Len(out[1].DependencyIssues, 1)
	issue := out[1].DependencyIssues[0]
	must.Equal(DependencyVersionTooLow, issue.State)
	must.Equal("1.0.0", issue.InstalledVersion)
}

func TestResolveDependenciesContentPackMissing(t *testing.T) {
	must := require.New(t)

	mods := []Mod{
		modWithDeps("Author.Pack", nil, &ContentPackFor{
			UniqueID:       "Pathoschild.ContentPatcher",
			MinimumVersion: "2.0.0",
		}),
	}
	out := ResolveDependencies(mods)
	must.Len(out[0].DependencyIssues, 1)
	issue := out[0].DependencyIssues[0]
	must.True(issue.IsContentPack)
	must.Equal(DependencyMissing, issue.State)
}

func TestResolveDependenciesEmptyMinimumVersion(t *testing.T) {
	must := require.New(t)

	mods := []Mod{
		providerMod("Author.Dependency", "0.1.0"),
		modWithDeps("Author.Mod", []ModDependency{
			{UniqueID: "Author.Dependency", IsRequired: requiredBool(true)},
		}, nil),
	}
	out := ResolveDependencies(mods)
	must.Empty(out[1].DependencyIssues)
}

func TestResolveDependenciesIgnoresSelfReference(t *testing.T) {
	must := require.New(t)

	mods := []Mod{
		modWithDeps("Author.Mod", []ModDependency{
			{UniqueID: "Author.Mod", IsRequired: requiredBool(true)},
		}, nil),
	}
	out := ResolveDependencies(mods)
	must.Empty(out[0].DependencyIssues)
}

func TestResolveDependenciesDedupesPreferRequired(t *testing.T) {
	must := require.New(t)

	mods := []Mod{
		modWithDeps("Author.Mod", []ModDependency{
			{UniqueID: "Author.Dependency", IsRequired: requiredBool(false)},
			{UniqueID: "Author.Dependency", MinimumVersion: "1.0.0", IsRequired: requiredBool(true)},
		}, nil),
	}
	out := ResolveDependencies(mods)
	must.Len(out[0].DependencyIssues, 1)
	must.True(out[0].DependencyIssues[0].IsRequired)
}

func TestResolveDependenciesDisabledProvider(t *testing.T) {
	must := require.New(t)

	provider := providerMod("Author.Dependency", "2.0.0")
	provider.Enabled = false
	mods := []Mod{
		provider,
		modWithDeps("Author.Mod", []ModDependency{
			{UniqueID: "Author.Dependency", MinimumVersion: "1.0.0", IsRequired: requiredBool(true)},
		}, nil),
	}
	out := ResolveDependencies(mods)
	must.Len(out[1].DependencyIssues, 1)
	issue := out[1].DependencyIssues[0]
	must.Equal(DependencyDisabled, issue.State)
	must.NotEmpty(issue.ProviderModID)
}
