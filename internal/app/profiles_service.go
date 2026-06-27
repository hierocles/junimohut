package app

import (
	"junimohut/internal/profiles"
)

type ProfilesService struct {
	core *Core
}

func NewProfilesService(core *Core) *ProfilesService {
	return &ProfilesService{core: core}
}

func (s *ProfilesService) ListProfiles() []profiles.Profile {
	if err := s.core.RequireStarted(); err != nil {
		return nil
	}
	return s.core.Profiles.List()
}

func (s *ProfilesService) CreateProfile(name string) (profiles.Profile, error) {
	if err := s.core.RequireStarted(); err != nil {
		return profiles.Profile{}, err
	}
	return s.core.Profiles.Create(name)
}

func (s *ProfilesService) DeleteProfile(id string) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	return s.core.Profiles.Delete(id)
}

func (s *ProfilesService) RenameProfile(id, name string) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	return s.core.Profiles.Rename(id, name)
}

func (s *ProfilesService) SetActiveProfile(id string) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	settings := s.core.Store.Get()
	if settings.ProfileSpecificConfigs {
		if err := s.core.ConfigMgr.SaveConfigs(settings.ModsRoot, modUniqueIDMapFromEnabled(s.core, s.core.Profiles.EnabledMods())); err != nil {
			return err
		}
	}
	if err := s.core.Profiles.SetActive(id); err != nil {
		return err
	}
	if settings.ProfileSpecificConfigs {
		if err := s.core.ConfigMgr.RestoreConfigs(settings.ModsRoot, modUniqueIDMapFromEnabled(s.core, s.core.Profiles.EnabledMods())); err != nil {
			return err
		}
	}
	return s.core.Catalog.Refresh(s.core.Ctx())
}

func (s *ProfilesService) SaveProfile() error {
	return nil
}
