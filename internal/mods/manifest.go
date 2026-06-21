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
	data = stripJSONTrailingCommas(data)

	type manifestAlias Manifest
	aux := &struct {
		UpdateKeys json.RawMessage `json:"UpdateKeys"`
		*manifestAlias
	}{
		manifestAlias: (*manifestAlias)(m),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if len(aux.UpdateKeys) > 0 && !bytes.Equal(aux.UpdateKeys, []byte("null")) {
		keys, err := parseFlexibleUpdateKeys(aux.UpdateKeys)
		if err != nil {
			return err
		}
		m.UpdateKeys = keys
	}
	return nil
}

// parseFlexibleUpdateKeys accepts SMAPI UpdateKeys arrays containing strings and/or numeric Nexus IDs.
func parseFlexibleUpdateKeys(data []byte) ([]string, error) {
	var stringsOnly []string
	if err := json.Unmarshal(data, &stringsOnly); err == nil {
		out := make([]string, 0, len(stringsOnly))
		for _, key := range stringsOnly {
			out = append(out, normalizeUpdateKey(key))
		}
		return out, nil
	}

	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		var key string
		if err := json.Unmarshal(item, &key); err == nil {
			out = append(out, normalizeUpdateKey(key))
			continue
		}
		var number json.Number
		if err := json.Unmarshal(item, &number); err == nil {
			id, err := number.Int64()
			if err != nil || id <= 0 {
				return nil, fmt.Errorf("invalid UpdateKeys number %q", number)
			}
			out = append(out, fmt.Sprintf("Nexus:%d", id))
			continue
		}
		return nil, fmt.Errorf("invalid UpdateKeys entry %s", string(item))
	}
	return out, nil
}

func normalizeUpdateKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return key
	}
	if strings.Contains(key, ":") {
		return key
	}
	var id int
	if _, err := fmt.Sscanf(key, "%d", &id); err == nil && id > 0 {
		return fmt.Sprintf("Nexus:%d", id)
	}
	return key
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

// stripJSONTrailingCommas removes trailing commas before } or ] (SMAPI-tolerant JSON).
func stripJSONTrailingCommas(data []byte) []byte {
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
		if data[i] == ',' {
			j := i + 1
			for j < len(data) && (data[j] == ' ' || data[j] == '\t' || data[j] == '\r' || data[j] == '\n') {
				j++
			}
			if j < len(data) && (data[j] == '}' || data[j] == ']') {
				i++
				continue
			}
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

// CanonicalUniqueID returns the normalized form used for SMAPI UniqueID lookup.
// SMAPI treats UniqueIDs as case-insensitive.
func CanonicalUniqueID(uid string) string {
	return strings.ToLower(uid)
}

// UniqueIDsEqual reports whether two SMAPI UniqueIDs refer to the same mod.
func UniqueIDsEqual(a, b string) bool {
	return strings.EqualFold(a, b)
}
