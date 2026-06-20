package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const appName = "JunimoHut"

// Store manages persisted settings and app data paths.
type Store struct {
	mu       sync.RWMutex
	settings Settings
	dataDir  string
}

// NewStore creates a store and loads settings from disk.
func NewStore() (*Store, error) {
	dataDir, err := ResolveDataDir()
	if err != nil {
		return nil, err
	}
	return newStoreAt(dataDir)
}

// NewStoreForDir creates a store at an explicit data directory (for tests).
func NewStoreForDir(dataDir string, settings Settings) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	s := &Store{dataDir: dataDir, settings: settings}
	if err := s.save(); err != nil {
		return nil, err
	}
	return s, nil
}

func newStoreAt(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	s := &Store{dataDir: dataDir, settings: DefaultSettings()}
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *Store) DataDir() string { return s.dataDir }
func (s *Store) ProfilesDir() string {
	return filepath.Join(s.dataDir, "profiles")
}
func (s *Store) SelectedModsDir() string {
	return filepath.Join(s.dataDir, "selected-mods")
}
func (s *Store) CategoriesPath() string {
	return filepath.Join(s.dataDir, "categories.json")
}
func (s *Store) ModNamesPath() string {
	return filepath.Join(s.dataDir, "mod-names.json")
}
func (s *Store) ConfigPath() string {
	return filepath.Join(s.dataDir, "config.json")
}
func (s *Store) DownloadsDir() string {
	return filepath.Join(s.dataDir, "downloads")
}

func (s *Store) Get() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

func (s *Store) Set(settings Settings) error {
	s.mu.Lock()
	s.settings = settings
	s.mu.Unlock()
	return s.save()
}

func (s *Store) Update(fn func(*Settings)) error {
	s.mu.Lock()
	fn(&s.settings)
	s.mu.Unlock()
	return s.save()
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.ConfigPath())
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return json.Unmarshal(data, &s.settings)
}

func (s *Store) save() error {
	s.mu.RLock()
	data, err := json.MarshalIndent(s.settings, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return err
	}
	return os.WriteFile(s.ConfigPath(), data, 0o644)
}

// EnsureDirs creates required application data directories.
func (s *Store) EnsureDirs() error {
	for _, dir := range []string{s.ProfilesDir(), s.SelectedModsDir(), s.DownloadsDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}
