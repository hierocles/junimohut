package nexus

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"junimohut/internal/archive"

	"github.com/stretchr/testify/require"
)

func testArchiveEnricher(rec *DownloadRecord) {
	if rec == nil || rec.ArchivePath == "" {
		return
	}
	tmpDir, err := os.MkdirTemp("", "nexus-enrich-test-*")
	if err != nil {
		return
	}
	defer os.RemoveAll(tmpDir)

	if err := archive.Extract(rec.ArchivePath, tmpDir); err != nil {
		return
	}

	manifestPath, err := findFirstManifest(tmpDir)
	if err != nil {
		return
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return
	}

	var stub struct {
		Name       string          `json:"Name"`
		UniqueID   string          `json:"UniqueID"`
		UpdateKeys json.RawMessage `json:"UpdateKeys"`
	}
	if err := json.Unmarshal(data, &stub); err != nil {
		return
	}
	if rec.UniqueID == "" && stub.UniqueID != "" {
		rec.UniqueID = stub.UniqueID
	}
	if rec.ModName == "" && stub.Name != "" {
		rec.ModName = stub.Name
	}
	if rec.NexusModID == 0 && len(stub.UpdateKeys) > 0 {
		rec.NexusModID = nexusIDFromManifestUpdateKeys(stub.UpdateKeys)
	}
}

func findFirstManifest(root string) (string, error) {
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.EqualFold(d.Name(), "manifest.json") {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if found == "" {
		return "", os.ErrNotExist
	}
	return found, nil
}

func nexusIDFromManifestUpdateKeys(raw json.RawMessage) int {
	var keys []string
	if err := json.Unmarshal(raw, &keys); err == nil {
		return ModIDFromUpdateKeys(keys)
	}
	var numbers []json.Number
	if err := json.Unmarshal(raw, &numbers); err == nil && len(numbers) > 0 {
		id, err := numbers[0].Int64()
		if err == nil && id > 0 {
			return int(id)
		}
	}
	return 0
}

func newTestDownloadIndex(t *testing.T, dataDir, downloadsDir string) *DownloadIndex {
	t.Helper()
	must := require.New(t)
	idx, err := NewDownloadIndex(dataDir, downloadsDir)
	must.NoError(err)
	idx.SetArchiveEnricher(testArchiveEnricher)
	return idx
}
