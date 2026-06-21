package mods

import (
	"os"
	"testing"
	"time"
)

// Manual benchmark: SDV_MODS=E:\SDV_MODS go test ./internal/mods -run TestUserLibraryScanBench -v
func TestUserLibraryScanBench(t *testing.T) {
	root := os.Getenv("SDV_MODS")
	if root == "" {
		t.Skip("set SDV_MODS to run")
	}
	s := NewScanner()
	start := time.Now()
	list, err := s.Scan(ScanOptions{
		ModsRoot:            root,
		IgnoreHiddenFolders: true,
		Grouping:            "folder",
	})
	if err != nil {
		t.Fatal(err)
	}
	scanDur := time.Since(start)

	start = time.Now()
	list = ResolveDependencies(list)
	resolveDur := time.Since(start)

	t.Logf("scan: %d mods in %v", len(list), scanDur)
	t.Logf("resolve: %v", resolveDur)
}
