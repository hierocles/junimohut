package smapi

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUpdateCheckRateLimited(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		lastCheck int64
		want      bool
	}{
		{"never checked", 0, false},
		{"checked 30 minutes ago", now.Add(-30 * time.Minute).Unix(), true},
		{"checked 61 minutes ago", now.Add(-61 * time.Minute).Unix(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			must := require.New(t)
			must.Equal(tt.want, UpdateCheckRateLimited(tt.lastCheck, now))
		})
	}
}
