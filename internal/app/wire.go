package app

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Services bundles all Wails domain services for registration.
type Services struct {
	Core           *Core
	System         *SystemService
	Settings       *SettingsService
	Mods           *ModsService
	Profiles       *ProfilesService
	Categories     *CategoriesService
	SMAPI          *SMAPIService
	Nexus          *NexusService
	ConfigEditor   *ConfigEditorService
}

// NewServices constructs Core and all domain services.
func NewServices(opts CoreOptions) *Services {
	core := NewCore(opts)
	return &Services{
		Core:         core,
		System:       NewSystemService(core),
		Settings:     NewSettingsService(core),
		Mods:         NewModsService(core),
		Profiles:     NewProfilesService(core),
		Categories:   NewCategoriesService(core),
		SMAPI:        NewSMAPIService(core),
		Nexus:        NewNexusService(core),
		ConfigEditor: NewConfigEditorService(core),
	}
}

// Register returns Wails service wrappers in startup order.
func (s *Services) Register() []application.Service {
	return []application.Service{
		application.NewService(s.System),
		application.NewService(s.Settings),
		application.NewService(s.Mods),
		application.NewService(s.Profiles),
		application.NewService(s.Categories),
		application.NewService(s.SMAPI),
		application.NewService(s.Nexus),
		application.NewService(s.ConfigEditor),
	}
}
