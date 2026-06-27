package app

import (
	"context"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type SystemService struct {
	core *Core
}

func NewSystemService(core *Core) *SystemService {
	return &SystemService{core: core}
}

func (s *SystemService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	return s.core.Startup(ctx)
}

func (s *SystemService) CheckAppUpdate() (string, error) {
	return "0.1.0", nil
}

func (s *SystemService) RegisterNXMProtocol() error {
	return registerNXMProtocol()
}

func (s *SystemService) ProcessCommandLineArgs(args []string) {
	for _, arg := range args {
		if strings.HasPrefix(arg, "nxm://") {
			s.EmitNXMURL(arg)
			return
		}
	}
}

func (s *SystemService) EmitNXMURL(url string) {
	s.core.Events.EmitNXMURL(url)
}
