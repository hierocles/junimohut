package nexus

import (
	"net/http"

	"junimohut/internal/httpclient"
)

const userAgent = "JunimoHut/0.1 (compatible; Nexus Mod Manager)"

var (
	apiHTTPClient      = httpclient.Default()
	downloadHTTPClient = httpclient.Download()
)

func setUserAgent(req *http.Request) {
	req.Header.Set("User-Agent", userAgent)
}
