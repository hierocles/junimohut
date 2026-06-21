package categories

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTagIDForNexusCategory(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"User Interface", "tag-ui"},
		{"Visuals and Graphics", "tag-visual"},
		{"Portraits", "tag-visual"},
		{"New Characters", "tag-characters"},
		{"Pets / Horses", "tag-farming"},
		{"Modding Tools", "tag-framework"},
		{"Miscellaneous", ""},
		{"", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			must := require.New(t)
			must.Equal(tc.want, TagIDForNexusCategory(tc.name))
		})
	}
}

func TestNexusCategoryDefersUntilManifest(t *testing.T) {
	must := require.New(t)
	must.True(NexusCategoryDefersUntilManifest("Clothing"))
	must.False(NexusCategoryDefersUntilManifest("Items"))
	must.False(NexusCategoryDefersUntilManifest("User Interface"))
}
