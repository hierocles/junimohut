package modnames

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
)

type store struct {
	CustomNames map[string]string `json:"customNames"`
}

// Service manages user-defined mod display names keyed by Mod.ID.
type Service struct {
	mu          sync.RWMutex
	path        string
	customNames map[string]string
}

func NewService(path string) (*Service, error) {
	s := &Service{
		path:        path,
		customNames: map[string]string{},
	}
	err := s.load()
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *Service) Get(modID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.customNames[modID]
}

func (s *Service) Set(modID, name string) error {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	if name == "" {
		delete(s.customNames, modID)
	} else {
		s.customNames[modID] = name
	}
	return s.save()
}

func (s *Service) Delete(modID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.customNames, modID)
	return s.save()
}

func (s *Service) All() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.customNames))
	for k, v := range s.customNames {
		out[k] = v
	}
	return out
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
	if st.CustomNames == nil {
		st.CustomNames = map[string]string{}
	}
	s.customNames = st.CustomNames
	return nil
}

func (s *Service) save() error {
	st := store{CustomNames: s.customNames}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}
