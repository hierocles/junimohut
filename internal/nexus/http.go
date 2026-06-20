package nexus

import (
	"net/http"
	"time"
)

const userAgent = "JunimoHut/0.1 (compatible; Nexus Mod Manager)"

var (
	apiHTTPClient      = &http.Client{Timeout: 60 * time.Second}
	downloadHTTPClient = &http.Client{Timeout: 30 * time.Minute}
)

func setUserAgent(req *http.Request) {
	req.Header.Set("User-Agent", userAgent)
}
