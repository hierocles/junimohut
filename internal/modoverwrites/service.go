package modoverwrites

import (
	"encoding/json"
	"os"
	"sync"
)

type store struct {
	ModIDs []string `json:"modIds"`
}

// Service tracks mod IDs that received merged overwrite patches via Junimo Hut.
type Service struct {
	mu     sync.RWMutex
	path   string
	modIDs map[string]struct{}
}

func NewService(path string) (*Service, error) {
	s := &Service{
		path:   path,
		modIDs: map[string]struct{}{},
	}
	err := s.load()
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *Service) ContainsOverwrites(modID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.modIDs[modID]
	return ok
}

func (s *Service) RecordMerge(modID string) error {
	if modID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.modIDs[modID] = struct{}{}
	return s.save()
}

func (s *Service) Delete(modID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.modIDs, modID)
	return s.save()
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
	s.modIDs = map[string]struct{}{}
	for _, id := range st.ModIDs {
		if id != "" {
			s.modIDs[id] = struct{}{}
		}
	}
	return nil
}

func (s *Service) save() error {
	ids := make([]string, 0, len(s.modIDs))
	for id := range s.modIDs {
		ids = append(ids, id)
	}
	st := store{ModIDs: ids}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}
