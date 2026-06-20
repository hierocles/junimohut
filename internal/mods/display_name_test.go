package mods

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultDisplayName(t *testing.T) {
	must := require.New(t)

	must.Empty(DefaultDisplayName("Lookup Anything", "Lookup Anything"))
	must.Equal(
		"[CP] Seasonal Open Windows - BIRCH",
		DefaultDisplayName("[CP] Seasonal Open Windows - BIRCH", "[CP] Seasonal Open Windows"),
	)
	must.Equal(
		"(AT) Chest Deco",
		DefaultDisplayName("(AT) Chest Deco", "Chest Deco"),
	)
	must.Empty(DefaultDisplayName("Stardew Valley Expanded/[CP] Stardew Valley Expanded", "[CP] Stardew Valley Expanded"))
}

func TestEffectiveCustomName(t *testing.T) {
	must := require.New(t)

	must.Equal("My Alias", EffectiveCustomName("My Alias", "folder", "Official", DisplayNameFolder))
	must.Equal(
		"[CP] Seasonal Open Windows - BLACK",
		EffectiveCustomName("", "[CP] Seasonal Open Windows - BLACK", "[CP] Seasonal Open Windows", DisplayNameFolder),
	)
	must.Empty(EffectiveCustomName("", "[CP] Seasonal Open Windows - BLACK", "[CP] Seasonal Open Windows", DisplayNameOfficial))
	must.Empty(EffectiveCustomName("", "Lookup Anything", "Lookup Anything", DisplayNameFolder))
}

func TestInstallResultDisplayName(t *testing.T) {
	must := require.New(t)

	must.Equal(
		"[CP] Seasonal Open Windows",
		InstallResultDisplayName("[CP] Seasonal Open Windows", "[CP] Seasonal Open Windows", false),
	)
	must.Equal(
		"[CP] Seasonal Open Windows - BIRCH",
		InstallResultDisplayName(
			"[CP] Seasonal Open Windows - BIRCH",
			"[CP] Seasonal Open Windows",
			true,
		),
	)
	must.Equal(
		"animal products",
		InstallResultDisplayName(
			"(AT) Chest Deco/animal products",
			"Chest Deco : animal products",
			true,
		),
	)
}
