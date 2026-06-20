package mods

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeUpdateState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{"update", "update_available"},
		{"ok", "current"},
		{"incompatible", "incompatible"},
		{"unofficial", "unofficial"},
		{"update_available", "update_available"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			must := require.New(t)
			must.Equal(tt.want, NormalizeUpdateState(tt.in))
		})
	}
}
