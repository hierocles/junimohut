package mods

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsFashionSenseRelated(t *testing.T) {
	falseVal := flexBool(false)

	tests := []struct {
		name string
		m    Manifest
		want bool
	}{
		{
			name: "content pack",
			m: Manifest{
				UniqueID: "Author.Outfits",
				ContentPackFor: &ContentPackFor{
					UniqueID: FashionSenseFrameworkUID,
				},
			},
			want: true,
		},
		{
			name: "required dependency",
			m: Manifest{
				UniqueID: "Author.Outfits",
				Dependencies: []ModDependency{{
					UniqueID: FashionSenseFrameworkUID,
				}},
			},
			want: true,
		},
		{
			name: "optional dependency",
			m: Manifest{
				UniqueID: "Author.Outfits",
				Dependencies: []ModDependency{{
					UniqueID:   FashionSenseFrameworkUID,
					IsRequired: &falseVal,
				}},
			},
			want: true,
		},
		{
			name: "framework mod itself",
			m: Manifest{
				UniqueID: FashionSenseFrameworkUID,
			},
			want: false,
		},
		{
			name: "unrelated mod",
			m: Manifest{
				UniqueID: "Author.OtherMod",
				ContentPackFor: &ContentPackFor{
					UniqueID: "Pathoschild.ContentPatcher",
				},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			must := require.New(t)
			must.Equal(tc.want, IsFashionSenseRelated(tc.m))
		})
	}
}
