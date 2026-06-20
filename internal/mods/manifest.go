package mods

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// flexBool accepts JSON booleans encoded as bool, string, or number.
type flexBool bool

func (b *flexBool) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		*b = false
		return nil
	}
	var asBool bool
	if err := json.Unmarshal(data, &asBool); err == nil {
		*b = flexBool(asBool)
		return nil
	}
	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		switch strings.ToLower(strings.TrimSpace(asString)) {
		case "true", "1", "yes":
			*b = true
		default:
			*b = false
		}
		return nil
	}
	var asNumber json.Number
	if err := json.Unmarshal(data, &asNumber); err == nil {
		i, err := asNumber.Int64()
		if err != nil {
			return fmt.Errorf("invalid boolean number %q", asNumber)
		}
		*b = flexBool(i != 0)
		return nil
	}
	return fmt.Errorf("invalid boolean value %s", string(data))
}

func (b flexBool) MarshalJSON() ([]byte, error) {
	return json.Marshal(bool(b))
}

func decodeJSONManifest(data []byte, m *Manifest) error {
	data = bytes.TrimPrefix(data, utf8BOM)
	data = stripJSONComments(data)
	return json.Unmarshal(data, m)
}

// stripJSONComments removes // line comments and /* block comments */ from JSON-like data.
// SMAPI manifest files commonly include comments that standard JSON parsers reject.
func stripJSONComments(data []byte) []byte {
	var out []byte
	i := 0
	inString := false
	for i < len(data) {
		if inString {
			if data[i] == '\\' && i+1 < len(data) {
				out = append(out, data[i], data[i+1])
				i += 2
				continue
			}
			if data[i] == '"' {
				inString = false
			}
			out = append(out, data[i])
			i++
			continue
		}
		if data[i] == '"' {
			inString = true
			out = append(out, data[i])
			i++
			continue
		}
		// Line comment
		if i+1 < len(data) && data[i] == '/' && data[i+1] == '/' {
			for i < len(data) && data[i] != '\n' {
				i++
			}
			continue
		}
		// Block comment
		if i+1 < len(data) && data[i] == '/' && data[i+1] == '*' {
			i += 2
			for i+1 < len(data) && !(data[i] == '*' && data[i+1] == '/') {
				i++
			}
			i += 2 // consume */
			continue
		}
		out = append(out, data[i])
		i++
	}
	return out
}

// FindManifestPath locates manifest.json in a directory (case-insensitive).
func FindManifestPath(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(e.Name(), "manifest.json") {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("manifest.json not found in %s", dir)
}

// ParseManifest reads and parses a manifest.json file.
func ParseManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	if err := decodeJSONManifest(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest: %w", err)
	}
	return m, nil
}

// ModID generates a stable ID from folder path and unique ID.
func ModID(folderPath, uniqueID string) string {
	return fmt.Sprintf("%s::%s", folderPath, uniqueID)
}
