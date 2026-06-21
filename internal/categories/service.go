package categories

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/google/uuid"
)

// Category is a user-defined mod organization label.
type Category struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Color     string   `json:"color"`
	Visible   bool     `json:"visible"`
	SortOrder int      `json:"sortOrder"`
	ModIDs    []string `json:"modIds"`
}

type store struct {
	Categories []Category `json:"categories"`
}

// Service manages user-defined categories.
type Service struct {
	mu         sync.RWMutex
	path       string
	categories []Category
}

func NewService(path string) (*Service, error) {
	s := &Service{path: path, categories: []Category{}}
	err := s.load()
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		s.categories = DefaultCategories()
		if err := s.save(); err != nil {
			return nil, err
		}
		return s, nil
	}
	if s.mergeMissingDefaults() {
		if err := s.save(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// mergeMissingDefaults appends default categories whose stable IDs are absent.
// Skips when the store is empty so an intentional empty file is preserved.
func (s *Service) mergeMissingDefaults() bool {
	if len(s.categories) == 0 {
		return false
	}
	existing := map[string]bool{}
	for _, c := range s.categories {
		existing[c.ID] = true
	}
	var added bool
	for _, def := range DefaultCategories() {
		if existing[def.ID] {
			continue
		}
		s.categories = append(s.categories, def)
		added = true
	}
	return added
}

func (s *Service) List() []Category {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Category, len(s.categories))
	copy(out, s.categories)
	return out
}

func (s *Service) Create(name, color string) (Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := Category{
		ID:        uuid.NewString(),
		Name:      name,
		Color:     color,
		Visible:   true,
		SortOrder: len(s.categories),
		ModIDs:    []string{},
	}
	s.categories = append(s.categories, c)
	return c, s.save()
}

func (s *Service) Update(id, name, color string, visible bool, sortOrder int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.categories {
		if c.ID == id {
			if name != "" {
				c.Name = name
			}
			if color != "" {
				c.Color = color
			}
			c.Visible = visible
			c.SortOrder = sortOrder
			s.categories[i] = c
			return s.save()
		}
	}
	return os.ErrNotExist
}

func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var kept []Category
	for _, c := range s.categories {
		if c.ID != id {
			kept = append(kept, c)
		}
	}
	s.categories = kept
	return s.save()
}

func (s *Service) SetVisibility(id string, visible bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.categories {
		if c.ID == id {
			c.Visible = visible
			s.categories[i] = c
			return s.save()
		}
	}
	return os.ErrNotExist
}

func (s *Service) AssignMod(categoryID, modID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.categories {
		if c.ID == categoryID {
			for _, existing := range c.ModIDs {
				if existing == modID {
					return nil
				}
			}
			c.ModIDs = append(c.ModIDs, modID)
			s.categories[i] = c
			return s.save()
		}
	}
	return os.ErrNotExist
}

func (s *Service) UnassignMod(categoryID, modID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.categories {
		if c.ID == categoryID {
			var kept []string
			for _, m := range c.ModIDs {
				if m != modID {
					kept = append(kept, m)
				}
			}
			c.ModIDs = kept
			s.categories[i] = c
			return s.save()
		}
	}
	return os.ErrNotExist
}

func (s *Service) ModCategoryIDs(modID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var ids []string
	for _, c := range s.categories {
		for _, m := range c.ModIDs {
			if m == modID {
				ids = append(ids, c.ID)
				break
			}
		}
	}
	return ids
}

func (s *Service) HiddenCategoryIDs() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hidden := map[string]bool{}
	for _, c := range s.categories {
		if !c.Visible {
			hidden[c.ID] = true
		}
	}
	return hidden
}

func (s *Service) CategoryModMap() map[string][]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := map[string][]string{}
	for _, c := range s.categories {
		m[c.ID] = append([]string{}, c.ModIDs...)
	}
	return m
}

func (s *Service) Reorder(ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	byID := map[string]Category{}
	for _, c := range s.categories {
		byID[c.ID] = c
	}
	var ordered []Category
	for i, id := range ids {
		if c, ok := byID[id]; ok {
			c.SortOrder = i
			ordered = append(ordered, c)
		}
	}
	s.categories = ordered
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
	s.categories = st.Categories
	return nil
}

func (s *Service) save() error {
	st := store{Categories: s.categories}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}
