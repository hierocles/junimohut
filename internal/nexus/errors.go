package nexus

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"junimohut/internal/httpclient"
)

// Nexus public API: 20,000 requests per 24-hour period per API key.
const dailyRequestLimit = 20_000

const freeDownloadHint = "Free Nexus accounts must start the download from the mod page — click \"Mod Manager Download\" or \"Slow download\" so Junimo Hut receives an authorized nxm:// link. Premium members can download updates directly from Junimo Hut."

// ErrNoAPIKeyConfigured is returned when a Nexus operation requires an API key.
var ErrNoAPIKeyConfigured = errors.New("No Nexus API key configured. Add one in Settings.")

// apiError formats Nexus HTTP failures; 429 gets a quota-specific message.
func apiError(status int, body []byte, context string) error {
	msg := strings.TrimSpace(string(body))
	if status == http.StatusTooManyRequests {
		return fmt.Errorf("Nexus API rate limit reached (%d requests per 24 hours); try again later", dailyRequestLimit)
	}
	if status == http.StatusForbidden && context == "get download link" &&
		strings.Contains(strings.ToLower(msg), "premium") {
		return fmt.Errorf("%s. %s", msg, freeDownloadHint)
	}
	if msg == "" {
		return fmt.Errorf("%s failed: HTTP %d", context, status)
	}
	return fmt.Errorf("%s failed: %s", context, msg)
}

func requestError(context string, err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "no such host"), strings.Contains(msg, "lookup"):
		return fmt.Errorf("%s: could not resolve Nexus server (DNS/network error)", context)
	case strings.Contains(msg, "dial tcp"), strings.Contains(msg, "connectex"),
		strings.Contains(msg, "connection refused"), strings.Contains(msg, "i/o timeout"):
		return fmt.Errorf("%s: could not connect to Nexus (firewall, proxy, or network issue)", context)
	default:
		return fmt.Errorf("%s: %w", context, err)
	}
}

func readAPIError(resp *http.Response, context string) error {
	body, _ := io.ReadAll(resp.Body)
	return apiError(resp.StatusCode, body, context)
}

// IsTransientNetworkError reports DNS/connectivity failures that may be temporary.
func IsTransientNetworkError(err error) bool {
	if httpclient.IsTransient(err) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "could not resolve Nexus server") ||
		strings.Contains(msg, "could not connect to Nexus")
}
