package profiles

import (
	"os"
	"testing"
	"time"

	"junimohut/internal/mods"
)

// Manual: SDV_MODS=E:\SDV_MODS ACTIVE_MODS="C:\Steam\steamapps\common\Stardew Valley\Mods" go test ./internal/profiles -run TestUserLibraryAssembleBench -v
func TestUserLibraryAssembleBench(t *testing.T) {
	modsRoot := os.Getenv("SDV_MODS")
	active := os.Getenv("ACTIVE_MODS")
	if modsRoot == "" || active == "" {
		t.Skip("set SDV_MODS and ACTIVE_MODS")
	}
	list, err := mods.NewScanner().Scan(mods.ScanOptions{
		ModsRoot:            modsRoot,
		IgnoreHiddenFolders: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	enabled := map[string]bool{}
	for _, m := range list {
		enabled[m.ID] = m.Enabled
	}
	asm := NewAssembler(active, modsRoot)
	start := time.Now()
	if err := asm.Assemble(list, enabled); err != nil {
		t.Fatal(err)
	}
	t.Logf("assemble %d mods: %v", len(list), time.Since(start))
}
