package profiles

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"junimohut/internal/mods"

	"github.com/google/uuid"
)

// Profile represents a mod loadout.
type Profile struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	EnabledMods map[string]bool `json:"enabledMods"`
	IsActive    bool            `json:"isActive"`
}

// Service manages mod profiles.
type Service struct {
	mu          sync.RWMutex
	profiles    []Profile
	activeID    string
	profilesDir string
}

func NewService(profilesDir string) (*Service, error) {
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		return nil, err
	}
	s := &Service{profilesDir: profilesDir, profiles: []Profile{}}
	if err := s.loadAll(); err != nil {
		return nil, err
	}
	if len(s.profiles) == 0 {
		p := Profile{
			ID:          uuid.NewString(),
			Name:        "Default",
			EnabledMods: map[string]bool{},
			IsActive:    true,
		}
		s.profiles = []Profile{p}
		s.activeID = p.ID
		_ = s.saveProfile(p)
	}
	return s, nil
}

func (s *Service) List() []Profile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Profile, len(s.profiles))
	copy(out, s.profiles)
	return out
}

func (s *Service) Active() Profile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.profiles {
		if p.ID == s.activeID {
			return p
		}
	}
	if len(s.profiles) > 0 {
		return s.profiles[0]
	}
	return Profile{EnabledMods: map[string]bool{}}
}

func (s *Service) ActiveID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeID
}

func (s *Service) EnabledMods() map[string]bool {
	p := s.Active()
	if p.EnabledMods == nil {
		return map[string]bool{}
	}
	out := make(map[string]bool, len(p.EnabledMods))
	for k, v := range p.EnabledMods {
		out[k] = v
	}
	return out
}

func (s *Service) Create(name string) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := Profile{
		ID:          uuid.NewString(),
		Name:        name,
		EnabledMods: map[string]bool{},
	}
	s.profiles = append(s.profiles, p)
	if err := s.saveProfile(p); err != nil {
		return Profile{}, err
	}
	return p, nil
}

func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == s.activeID && len(s.profiles) <= 1 {
		return fmt.Errorf("cannot delete the only profile")
	}
	var kept []Profile
	for _, p := range s.profiles {
		if p.ID != id {
			kept = append(kept, p)
		} else {
			_ = os.Remove(s.profilePath(id))
		}
	}
	s.profiles = kept
	if s.activeID == id && len(kept) > 0 {
		s.activeID = kept[0].ID
		kept[0].IsActive = true
		_ = s.saveProfile(kept[0])
	}
	return nil
}

func (s *Service) Rename(id, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, p := range s.profiles {
		if p.ID == id {
			s.profiles[i].Name = name
			return s.saveProfile(s.profiles[i])
		}
	}
	return fmt.Errorf("profile not found")
}

func (s *Service) SetActive(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := false
	for i, p := range s.profiles {
		s.profiles[i].IsActive = p.ID == id
		if p.ID == id {
			found = true
			s.activeID = id
		}
	}
	if !found {
		return fmt.Errorf("profile not found")
	}
	for _, p := range s.profiles {
		_ = s.saveProfile(p)
	}
	return nil
}

func (s *Service) SetModEnabled(modID string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, p := range s.profiles {
		if p.ID == s.activeID {
			if p.EnabledMods == nil {
				p.EnabledMods = map[string]bool{}
			}
			if mods.IsPackModID(modID) {
				mods.MigratePackEnableState(p.EnabledMods, modID, enabled)
			} else {
				p.EnabledMods[modID] = enabled
			}
			s.profiles[i] = p
			return s.saveProfile(p)
		}
	}
	return fmt.Errorf("no active profile")
}

func (s *Service) UpdateEnabledMods(profileID string, enabled map[string]bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, p := range s.profiles {
		if p.ID == profileID {
			p.EnabledMods = enabled
			s.profiles[i] = p
			return s.saveProfile(p)
		}
	}
	return fmt.Errorf("profile not found")
}

func (s *Service) SaveEnabledMods(enabled map[string]bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, p := range s.profiles {
		if p.ID == s.activeID {
			p.EnabledMods = enabled
			s.profiles[i] = p
			return s.saveProfile(p)
		}
	}
	return fmt.Errorf("no active profile")
}

func (s *Service) profilePath(id string) string {
	return filepath.Join(s.profilesDir, id+".json")
}

func (s *Service) configsDir(profileID string) string {
	return filepath.Join(s.profilesDir, profileID, "configs")
}

func (s *Service) loadAll() error {
	entries, err := os.ReadDir(s.profilesDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.profilesDir, e.Name()))
		if err != nil {
			continue
		}
		var p Profile
		if json.Unmarshal(data, &p) == nil {
			if p.EnabledMods == nil {
				p.EnabledMods = map[string]bool{}
			}
			s.profiles = append(s.profiles, p)
			if p.IsActive {
				s.activeID = p.ID
			}
		}
	}
	return nil
}

func (s *Service) saveProfile(p Profile) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.profilePath(p.ID), data, 0o644)
}
