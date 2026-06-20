package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActiveModsDir(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	got := ActiveModsDir(`C:\Games\Stardew Valley`)
	want := `C:\Games\Stardew Valley\Mods`
	must.Equal(want, got)
}
