package modupdates

import (
	"encoding/json"
	"os"
	"sync"

	"junimohut/internal/mods"
)

// Entry is a persisted update check result for one mod at a specific installed version.
type Entry struct {
	ManifestVersion      string `json:"manifestVersion"`
	State                string `json:"state"`
	LatestVersion        string `json:"latestVersion,omitempty"`
	ModPageURL           string `json:"modPageUrl,omitempty"`
	Message              string `json:"message,omitempty"`
	CompatibilityStatus  string `json:"compatibilityStatus,omitempty"`
	CompatibilitySummary string `json:"compatibilitySummary,omitempty"`
}

type store struct {
	Mods map[string]Entry `json:"mods"`
}

// Service persists mod update check results keyed by mods.Mod.ID.
type Service struct {
	mu   sync.RWMutex
	path string
	mods map[string]Entry
}

func NewService(path string) (*Service, error) {
	s := &Service{
		path: path,
		mods: map[string]Entry{},
	}
	err := s.load()
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *Service) All() map[string]Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]Entry, len(s.mods))
	for id, e := range s.mods {
		out[id] = e
	}
	return out
}

// SyncFromMods writes non-default update statuses for the current mod list and prunes stale entries.
func (s *Service) SyncFromMods(list []mods.Mod) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	active := make(map[string]bool, len(list))
	dirty := false
	for _, m := range list {
		active[m.ID] = true
		if prev, ok := s.mods[m.ID]; ok && prev.ManifestVersion != m.Manifest.Version {
			delete(s.mods, m.ID)
			dirty = true
		}
		if isDefaultCurrent(m.UpdateStatus) {
			if _, had := s.mods[m.ID]; had {
				delete(s.mods, m.ID)
				dirty = true
			}
			continue
		}
		next := entryFromMod(m)
		if prev, ok := s.mods[m.ID]; !ok || prev != next {
			s.mods[m.ID] = next
			dirty = true
		}
	}
	for id := range s.mods {
		if !active[id] {
			delete(s.mods, id)
			dirty = true
		}
	}
	if !dirty {
		return nil
	}
	return s.save()
}

func entryFromMod(m mods.Mod) Entry {
	st := m.UpdateStatus
	return Entry{
		ManifestVersion:      m.Manifest.Version,
		State:                st.State,
		LatestVersion:        st.LatestVersion,
		ModPageURL:           st.ModPageURL,
		Message:              st.Message,
		CompatibilityStatus:  st.CompatibilityStatus,
		CompatibilitySummary: st.CompatibilitySummary,
	}
}

func isDefaultCurrent(st mods.UpdateStatus) bool {
	return st.State == "" || st.State == "current"
}

func (s *Service) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	var st store
	if err := json.Unmarshal(data, &st); err != nil {
		return err
	}
	if st.Mods == nil {
		st.Mods = map[string]Entry{}
	}
	s.mods = st.Mods
	return nil
}

func (s *Service) save() error {
	st := store{Mods: s.mods}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}
