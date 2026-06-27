package app

import (
	"junimohut/internal/categories"
)

type CategoriesService struct {
	core *Core
}

func NewCategoriesService(core *Core) *CategoriesService {
	return &CategoriesService{core: core}
}

func (s *CategoriesService) ListCategories() []categories.Category {
	if err := s.core.RequireStarted(); err != nil {
		return nil
	}
	return s.core.Categories.List()
}

func (s *CategoriesService) CreateCategory(name, color string) (categories.Category, error) {
	if err := s.core.RequireStarted(); err != nil {
		return categories.Category{}, err
	}
	return s.core.Categories.Create(name, color)
}

func (s *CategoriesService) UpdateCategory(id, name, color string, visible bool, sortOrder int) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	return s.core.Categories.Update(id, name, color, visible, sortOrder)
}

func (s *CategoriesService) DeleteCategory(id string) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	return s.core.Categories.Delete(id)
}

func (s *CategoriesService) SetCategoryVisibility(id string, visible bool) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	return s.core.Categories.SetVisibility(id, visible)
}

func (s *CategoriesService) AssignModToCategory(categoryID, modID string) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	if err := s.core.Categories.AssignMod(categoryID, modID); err != nil {
		return err
	}
	s.core.Catalog.RefreshCategoryIDs()
	s.core.Events.EmitModsChanged()
	return nil
}

func (s *CategoriesService) UnassignModFromCategory(categoryID, modID string) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	if err := s.core.Categories.UnassignMod(categoryID, modID); err != nil {
		return err
	}
	s.core.Catalog.RefreshCategoryIDs()
	s.core.Events.EmitModsChanged()
	return nil
}

func (s *CategoriesService) ReorderCategories(ids []string) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	return s.core.Categories.Reorder(ids)
}
