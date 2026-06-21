package httpclient

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsTransient(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	must.True(IsTransient(errors.New("lookup smapi.io: no such host")))
	must.True(IsTransient(errors.New("dial tcp: connection refused")))
	must.False(IsTransient(errors.New("HTTP 401")))
}

type flakyTransport struct {
	attempts int
	inner    http.RoundTripper
}

func (f *flakyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	f.attempts++
	if f.attempts == 1 {
		return nil, errors.New("lookup example.com: no such host")
	}
	if f.inner == nil {
		f.inner = http.DefaultTransport
	}
	return f.inner.RoundTrip(req)
}

func TestDoWithRetryRecoversFromTransientFailure(t *testing.T) {
	t.Parallel()
	must := require.New(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	transport := &flakyTransport{}
	client := &http.Client{Transport: transport}
	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	must.NoError(err)

	resp, err := DoWithRetry(client, req, 3)
	must.NoError(err)
	must.Equal(http.StatusOK, resp.StatusCode)
	resp.Body.Close()
	must.Equal(2, transport.attempts)
}
