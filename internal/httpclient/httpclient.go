package httpclient

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"
)

const defaultTimeout = 60 * time.Second

// Default returns a shared HTTP client with proxy-aware transport and sensible timeouts.
func Default() *http.Client {
	return &http.Client{
		Timeout:   defaultTimeout,
		Transport: defaultTransport(),
	}
}

// Download returns a client suited for large file downloads.
func Download() *http.Client {
	return &http.Client{
		Timeout:   30 * time.Minute,
		Transport: defaultTransport(),
	}
}

func defaultTransport() *http.Transport {
	return &http.Transport{
		Proxy:               proxyForRequest,
		DialContext:         (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns:        20,
		MaxIdleConnsPerHost: 4,
		IdleConnTimeout:     90 * time.Second,
	}
}

// IsTransient reports whether err is likely a temporary network/DNS failure.
func IsTransient(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "lookup") ||
		strings.Contains(msg, "getaddrinfo") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connectex") ||
		strings.Contains(msg, "connection reset")
}

// DoWithRetry executes req up to attempts times when a transient error occurs.
func DoWithRetry(client *http.Client, req *http.Request, attempts int) (*http.Response, error) {
	if client == nil {
		client = Default()
	}
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		reqClone := req.Clone(req.Context())
		resp, err := client.Do(reqClone)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !IsTransient(err) || i == attempts-1 {
			return nil, err
		}
		time.Sleep(time.Duration(250*(i+1)) * time.Millisecond)
	}
	return nil, lastErr
}

// GetWithRetry performs a GET with transient-error retries.
func GetWithRetry(client *http.Client, url string, attempts int) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return DoWithRetry(client, req, attempts)
}
