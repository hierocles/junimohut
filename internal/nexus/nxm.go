package nexus

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// DownloadAuth carries a short-lived Nexus download key from an nxm:// link.
// Free accounts must pass these values when requesting download_link.json.
type DownloadAuth struct {
	Key     string
	Expires int64
}

// NXMURL is a parsed nxm:// mod download link.
type NXMURL struct {
	ModID   int
	FileID  int
	Auth    *DownloadAuth
}

// ParseNXMURL parses an nxm:// link, including free-tier key and expiry query params.
func ParseNXMURL(raw string) (NXMURL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "nxm" {
		return NXMURL{}, fmt.Errorf("invalid NXM URL")
	}

	path := strings.TrimPrefix(u.EscapedPath(), "/")
	parts := strings.Split(path, "/")
	var out NXMURL
	for i, p := range parts {
		if p == "mods" && i+1 < len(parts) {
			fmt.Sscanf(parts[i+1], "%d", &out.ModID)
		}
		if p == "files" && i+1 < len(parts) {
			fmt.Sscanf(parts[i+1], "%d", &out.FileID)
		}
	}
	if out.ModID == 0 {
		return NXMURL{}, fmt.Errorf("invalid NXM URL")
	}

	key := strings.TrimSpace(u.Query().Get("key"))
	expiresRaw := strings.TrimSpace(u.Query().Get("expires"))
	if key != "" && expiresRaw != "" {
		expires, err := strconv.ParseInt(expiresRaw, 10, 64)
		if err != nil || expires <= 0 {
			return NXMURL{}, fmt.Errorf("invalid NXM URL: bad expires parameter")
		}
		out.Auth = &DownloadAuth{Key: key, Expires: expires}
	}
	return out, nil
}

func downloadLinkURL(client *Client, modID, fileID int, auth *DownloadAuth) string {
	base := fmt.Sprintf("%s/v1/games/%s/mods/%d/files/%d/download_link.json",
		client.apiBaseURL(), gameDomain, modID, fileID)
	if auth == nil || auth.Key == "" || auth.Expires <= 0 {
		return base
	}
	q := url.Values{}
	q.Set("key", auth.Key)
	q.Set("expires", strconv.FormatInt(auth.Expires, 10))
	return base + "?" + q.Encode()
}
