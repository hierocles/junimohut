package modnames

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetGetClear(t *testing.T) {
	must := require.New(t)

	path := filepath.Join(t.TempDir(), "mod-names.json")
	svc, err := NewService(path)
	must.NoError(err)
	must.Empty(svc.Get("mod-1"))

	must.NoError(svc.Set("mod-1", "  My Label  "))
	must.Equal("My Label", svc.Get("mod-1"))

	must.NoError(svc.Set("mod-1", ""))
	must.Empty(svc.Get("mod-1"))
}

func TestDelete(t *testing.T) {
	must := require.New(t)

	path := filepath.Join(t.TempDir(), "mod-names.json")
	svc, err := NewService(path)
	must.NoError(err)
	must.NoError(svc.Set("mod-1", "Alias"))
	must.NoError(svc.Delete("mod-1"))
	must.Empty(svc.Get("mod-1"))
}

func TestPersistenceRoundTrip(t *testing.T) {
	must := require.New(t)

	path := filepath.Join(t.TempDir(), "mod-names.json")
	svc, err := NewService(path)
	must.NoError(err)
	must.NoError(svc.Set("folder::Author.Mod", "Short Name"))

	reloaded, err := NewService(path)
	must.NoError(err)
	must.Equal("Short Name", reloaded.Get("folder::Author.Mod"))
	must.Equal(map[string]string{"folder::Author.Mod": "Short Name"}, reloaded.All())

	data, err := os.ReadFile(path)
	must.NoError(err)
	var st store
	must.NoError(json.Unmarshal(data, &st))
	must.Equal("Short Name", st.CustomNames["folder::Author.Mod"])
}

func TestAllReturnsCopy(t *testing.T) {
	must := require.New(t)

	path := filepath.Join(t.TempDir(), "mod-names.json")
	svc, err := NewService(path)
	must.NoError(err)
	must.NoError(svc.Set("mod-1", "A"))

	all := svc.All()
	all["mod-1"] = "mutated"
	must.Equal("A", svc.Get("mod-1"))
}
