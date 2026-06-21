package categories

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeInstallSuggestedTags(t *testing.T) {
	known := map[string]bool{
		TagItems:        true,
		TagFashionSense: true,
		"tag-ui":        true,
	}

	tests := []struct {
		name         string
		nexusTagIDs  []string
		fashionSense bool
		want         []string
	}{
		{
			name:         "clothing nexus without fs",
			nexusTagIDs:  []string{TagItems},
			fashionSense: false,
			want:         []string{TagItems},
		},
		{
			name:         "clothing nexus with fs",
			nexusTagIDs:  []string{TagItems},
			fashionSense: true,
			want:         []string{TagFashionSense},
		},
		{
			name:         "fs manifest only",
			nexusTagIDs:  nil,
			fashionSense: true,
			want:         []string{TagFashionSense},
		},
		{
			name:         "fs with other nexus tags",
			nexusTagIDs:  []string{TagItems, "tag-ui"},
			fashionSense: true,
			want:         []string{"tag-ui", TagFashionSense},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			must := require.New(t)
			must.Equal(tc.want, MergeInstallSuggestedTags(tc.nexusTagIDs, tc.fashionSense, known))
		})
	}
}
