package profiles

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDesiredLinkKeyMatchesRemoveStaleLookup(t *testing.T) {
	t.Parallel()
	must := require.New(t)
	folderPath := "Downtown-Zuzu-main/[CP] Downtown Zuzu"
	key := desiredLinkKey(folderPath)
	rel := filepath.ToSlash(filepath.FromSlash(folderPath))
	must.Equal(key, rel, "desired map key must match removeStaleLinks lookup on Windows")
}
