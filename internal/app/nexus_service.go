package app

import (
	"fmt"

	"junimohut/internal/categories"
	"junimohut/internal/mods"
	"junimohut/internal/nexus"
	"junimohut/internal/platform"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type NexusService struct { core *Core }
func NewNexusService(core *Core) *NexusService { return &NexusService{core: core} }

func (s *NexusService) SetNexusAPIKey(key string) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	return s.core.Nexus.SetAPIKey(key)
}

func (s *NexusService) ValidateNexusAPIKey() (bool, error) {
	if err := s.core.RequireStarted(); err != nil {
		return false, err
	}
	return s.core.Nexus.ValidateKey()
}

func (s *NexusService) ProbeNexusAPIKey() bool {
	if err := s.core.RequireStarted(); err != nil {
		return false
	}
	if !s.core.Nexus.IsConnected() {
		return false
	}
	ok, err := s.core.Nexus.ValidateKey()
	if err != nil && nexus.IsTransientNetworkError(err) {
		return true
	}
	return ok && err == nil
}

func (s *NexusService) IsNexusConnected() bool {
	if err := s.core.RequireStarted(); err != nil {
		return false
	}
	return s.core.Nexus.IsConnected()
}

func (s *NexusService) GetInstallSuggestedTags(archivePaths []string, modIDs []int) ([]string, error) {
	if err := s.core.RequireStarted(); err != nil {
		return nil, err
	}

	knownTags := map[string]bool{}
	for _, c := range s.core.Categories.List() {
		knownTags[c.ID] = true
	}

	fashionSense, err := mods.ArchivesContainFashionSense(archivePaths)
	if err != nil {
		return nil, err
	}

	var nexusTagIDs []string
	hasArchives := len(archivePaths) > 0
	if s.core.Nexus.IsConnected() {
		seen := map[string]bool{}
		for _, modID := range modIDs {
			if modID <= 0 {
				continue
			}
			name, err := s.core.Nexus.CategoryNameForMod(modID)
			if err != nil || name == "" {
				continue
			}
			tagID := categories.TagIDForNexusCategory(name)
			if tagID == "" || seen[tagID] || !knownTags[tagID] {
				continue
			}
			if !hasArchives && categories.NexusCategoryDefersUntilManifest(name) {
				continue
			}
			seen[tagID] = true
			nexusTagIDs = append(nexusTagIDs, tagID)
		}
	}

	return categories.MergeInstallSuggestedTags(nexusTagIDs, fashionSense, knownTags), nil
}

func (s *NexusService) GetNexusSuggestedTags(modIDs []int) ([]string, error) {
	return s.GetInstallSuggestedTags(nil, modIDs)
}

func (s *NexusService) EndorseMod(updateKey, version string) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	id, ok := nexus.ExtractNexusID(updateKey)
	if !ok {
		return fmt.Errorf("Mod has no Nexus update key")
	}
	return s.core.Nexus.EndorseMod(id, version)
}

func (s *NexusService) ListDownloads() []nexus.DownloadEntry {
	if err := s.core.RequireStarted(); err != nil {
		return nil
	}
	return s.core.Downloads.List()
}

func (s *NexusService) ListSavedDownloads() []nexus.DownloadRecord {
	if err := s.core.RequireStarted(); err != nil {
		return nil
	}
	s.core.DownloadIndex.Reconcile()
	return s.core.DownloadIndex.List()
}

func (s *NexusService) DeleteSavedDownload(archivePath string) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	return s.core.DownloadIndex.Delete(archivePath)
}

func (s *NexusService) RevealArchiveInFileManager(archivePath string) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	return platform.RevealInFileManager(archivePath)
}

func (s *NexusService) DownloadModUpdate(updateKey string, modName string) (string, error) {
	if err := s.core.RequireStarted(); err != nil {
		return "", err
	}
	id, ok := nexus.ExtractNexusID(updateKey)
	if !ok {
		return "", fmt.Errorf("Not a Nexus mod")
	}
	return s.downloadNexusFile(id, 0, modName, nil)
}

func (s *NexusService) HandleNXMURL(url string) (string, error) {
	if err := s.core.RequireStarted(); err != nil {
		return "", err
	}
	parsed, err := nexus.ParseNXMURL(url)
	if err != nil {
		return "", err
	}
	return s.downloadNexusFile(parsed.ModID, parsed.FileID, fmt.Sprintf("mod_%d", parsed.ModID), parsed.Auth)
}

func (s *NexusService) SelectArchives() ([]string, error) {
	if err := s.core.RequireStarted(); err != nil {
		return nil, err
	}
	if s.core.App == nil {
		return nil, fmt.Errorf("App not ready")
	}
	paths, err := s.core.App.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		Title:                   "Select mod archives",
		AllowsMultipleSelection: true,
		CanChooseFiles:          true,
		CanChooseDirectories:    false,
		Filters: []application.FileFilter{
			{DisplayName: "Mod archives", Pattern: "*.zip;*.7z;*.rar"},
		},
	}).PromptForMultipleSelection()
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return []string{}, nil
	}
	return paths, nil
}

func (s *NexusService) downloadNexusFile(modID, fileID int, modName string, auth *nexus.DownloadAuth) (string, error) {
	path, err := s.core.Downloads.DownloadFile(s.core.Nexus, modID, fileID, modName, auth)
	if err != nil {
		return "", err
	}
	s.core.Events.EmitDownloadReady(path)
	return path, nil
}
