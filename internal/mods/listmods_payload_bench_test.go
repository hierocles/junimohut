package mods

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestUserLibraryListModsPayloadBench(t *testing.T) {
	root := os.Getenv("SDV_MODS")
	if root == "" {
		t.Skip("set SDV_MODS")
	}
	list, err := NewScanner().Scan(ScanOptions{
		ModsRoot:            root,
		IgnoreHiddenFolders: true,
		Grouping:            "folder",
	})
	if err != nil {
		t.Fatal(err)
	}
	list = ResolveDependencies(list)

	start := time.Now()
	data, err := json.Marshal(list)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("json marshal: %d mods, %.1f MB in %v", len(list), float64(len(data))/1e6, time.Since(start))

	start = time.Now()
	var out []Mod
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	t.Logf("json unmarshal: %v", time.Since(start))
}
