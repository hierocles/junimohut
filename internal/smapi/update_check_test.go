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
		{"checked 12 hours ago", now.Add(-12 * time.Hour).Unix(), true},
		{"checked 25 hours ago", now.Add(-25 * time.Hour).Unix(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			must := require.New(t)
			must.Equal(tt.want, UpdateCheckRateLimited(tt.lastCheck, now))
		})
	}
}

func TestUpdateCheckRetryAfter(t *testing.T) {
	t.Parallel()
	must := require.New(t)
	now := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)

	must.Equal(time.Duration(0), UpdateCheckRetryAfter(0, now))
	must.Equal(12*time.Hour, UpdateCheckRetryAfter(now.Add(-12*time.Hour).Unix(), now))
	must.Equal(time.Duration(0), UpdateCheckRetryAfter(now.Add(-25*time.Hour).Unix(), now))
}

func TestFormatUpdateCheckRetryAfter(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	must.Equal("soon", FormatUpdateCheckRetryAfter(0))
	must.Equal("31 minutes", FormatUpdateCheckRetryAfter(30*time.Minute))
	must.Equal("2 hours", FormatUpdateCheckRetryAfter(1*time.Hour))
	must.Equal("18 hours", FormatUpdateCheckRetryAfter(17*time.Hour))
}
