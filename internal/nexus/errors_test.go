package nexus

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIError429(t *testing.T) {
	must := require.New(t)

	err := apiError(http.StatusTooManyRequests, []byte("quota exceeded"), "list mod files")
	must.Error(err)
	must.ErrorContains(err, "20")
	must.ErrorContains(err, "24 hours")
}
