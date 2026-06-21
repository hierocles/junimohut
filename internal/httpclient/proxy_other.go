//go:build !windows

package httpclient

import (
	"net/http"
	"net/url"
)

func proxyForRequest(req *http.Request) (*url.URL, error) {
	return http.ProxyFromEnvironment(req)
}
