package modtimes

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

type ModTimes struct {
	InstallTime int64 `json:"installTime"`
	LastUpdated int64 `json:"lastUpdated"`
}

type store struct {
	Mods map[string]ModTimes `json:"mods"`
}

// Service manages persisted install and last-updated timestamps keyed by Mod.ID.
type Service struct {
	mu   sync.RWMutex
	path string
	mods map[string]ModTimes
}

func NewService(path string) (*Service, error) {
	s := &Service{
		path: path,
		mods: map[string]ModTimes{},
	}
	err := s.load()
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *Service) Get(modID string) (ModTimes, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.mods[modID]
	return rec, ok
}

// RecordInstall sets both timestamps to now for a newly installed mod.
func (s *Service) RecordInstall(modID string) error {
	if modID == "" {
		return nil
	}
	now := time.Now().Unix()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mods[modID] = ModTimes{InstallTime: now, LastUpdated: now}
	return s.save()
}

// RecordUpdate bumps lastUpdated, preserving installTime when a record already exists.
// When no record exists, installTime falls back to installFallback (e.g. manifest mtime)
// or now when fallback is zero.
func (s *Service) RecordUpdate(modID string, installFallback int64) error {
	if modID == "" {
		return nil
	}
	now := time.Now().Unix()
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.mods[modID]
	if !ok {
		install := installFallback
		if install <= 0 {
			install = now
		}
		rec = ModTimes{InstallTime: install, LastUpdated: now}
	} else {
		rec.LastUpdated = now
	}
	s.mods[modID] = rec
	return s.save()
}

// SeedBatch inserts manifest-derived timestamps for mods without persisted records.
func (s *Service) SeedBatch(entries map[string]int64) error {
	if len(entries) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	dirty := false
	for modID, manifestTime := range entries {
		if modID == "" || manifestTime <= 0 {
			continue
		}
		if _, ok := s.mods[modID]; ok {
			continue
		}
		s.mods[modID] = ModTimes{InstallTime: manifestTime, LastUpdated: manifestTime}
		dirty = true
	}
	if !dirty {
		return nil
	}
	return s.save()
}

func (s *Service) Delete(modID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.mods, modID)
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
	if st.Mods == nil {
		st.Mods = map[string]ModTimes{}
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
