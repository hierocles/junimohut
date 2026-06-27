package app

import (
	"fmt"
	"net/url"
	"strings"
)

func configEditorWindowTitle(modName, fileName string) string {
	name := strings.TrimSpace(modName)
	if name == "" {
		name = "Mod"
	}
	file := strings.TrimSpace(fileName)
	if file == "" {
		file = "config.json"
	}
	return fmt.Sprintf("%s — %s", name, file)
}

func configEditorURL(modID, relPath string) string {
	u := "/config-editor.html?modId=" + url.QueryEscape(modID)
	if strings.TrimSpace(relPath) != "" {
		u += "&file=" + url.QueryEscape(relPath)
	}
	return u
}
