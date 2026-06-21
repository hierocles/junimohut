package mods

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModJSONRoundTrip(t *testing.T) {
	must := require.New(t)
	dir := t.TempDir()
	manifest := `{
		"Name": "[AT] Pet Facelift",
		"Author": "siamece",
		"Version": "1.1.0",
		"UniqueID": "siamece.AT.PetFacelift",
		"UpdateKeys": [9097],
		"ContentPackFor": {"UniqueID": "PeacefulEnd.AlternativeTextures"}
	}`
	must.NoError(os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644))
	parsed, err := ParseManifest(filepath.Join(dir, "manifest.json"))
	must.NoError(err)

	mod := Mod{
		ID:         ModID("Pet Facelift", parsed.UniqueID),
		FolderPath: "(AT) Pet Facelift",
		Manifest:   parsed,
	}
	data, err := json.Marshal(mod)
	must.NoError(err)
	t.Log(string(data))

	var decoded map[string]any
	must.NoError(json.Unmarshal(data, &decoded))
	manifestMap := decoded["manifest"].(map[string]any)
	must.Equal("siamece.AT.PetFacelift", manifestMap["UniqueID"])
	must.Equal([]any{"Nexus:9097"}, manifestMap["UpdateKeys"])
}
