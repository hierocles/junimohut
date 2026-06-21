package mods

import (
	"encoding/json"
	"fmt"

	"github.com/tidwall/jsonc"
)

// ValidJSONC reports whether content is valid JSON with comments (JSONC).
func ValidJSONC(content string) error {
	if !json.Valid(jsonc.ToJSON([]byte(content))) {
		return fmt.Errorf("%w", ErrModConfigInvalidJSON)
	}
	return nil
}
