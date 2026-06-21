package nexus

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsTransientNetworkError(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	must.True(IsTransientNetworkError(errors.New("validate Nexus API key: could not resolve Nexus server (DNS/network error)")))
	must.False(IsTransientNetworkError(errors.New("validate API key failed: HTTP 401")))
}
