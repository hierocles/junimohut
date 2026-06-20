package profiles

import (
	"os"
	"path/filepath"
)

// ConfigManager handles profile-specific mod config.json files.
type ConfigManager struct {
	profilesDir string
	service     *Service
}

func NewConfigManager(profilesDir string, service *Service) *ConfigManager {
	return &ConfigManager{profilesDir: profilesDir, service: service}
}

// SaveModConfig backs up a single mod config.json for the active profile.
func (c *ConfigManager) SaveModConfig(modsRoot, modID, uniqueID string) error {
	return c.saveModConfig(modsRoot, modID, uniqueID)
}

// RestoreModConfig restores a single mod config.json for the active profile.
func (c *ConfigManager) RestoreModConfig(modsRoot, modID, uniqueID string) error {
	return c.restoreModConfig(modsRoot, modID, uniqueID)
}

// SaveConfigs backs up config.json for mods in modUniqueIDs (mod ID -> UniqueID).
func (c *ConfigManager) SaveConfigs(modsRoot string, modUniqueIDs map[string]string) error {
	if c.service.ActiveID() == "" {
		return nil
	}
	for modID, uniqueID := range modUniqueIDs {
		_ = c.saveModConfig(modsRoot, modID, uniqueID)
	}
	return nil
}

// RestoreConfigs restores config.json files for mods in modUniqueIDs.
func (c *ConfigManager) RestoreConfigs(modsRoot string, modUniqueIDs map[string]string) error {
	if c.service.ActiveID() == "" {
		return nil
	}
	for modID, uniqueID := range modUniqueIDs {
		_ = c.restoreModConfig(modsRoot, modID, uniqueID)
	}
	return nil
}

func (c *ConfigManager) saveModConfig(modsRoot, modID, uniqueID string) error {
	profileID := c.service.ActiveID()
	configsDir := filepath.Join(c.profilesDir, profileID, "configs")
	_ = os.MkdirAll(configsDir, 0o755)

	folderPath, _ := splitModID(modID)
	src := filepath.Join(modsRoot, filepath.FromSlash(folderPath), "config.json")
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}
	destDir := filepath.Join(configsDir, uniqueID)
	_ = os.MkdirAll(destDir, 0o755)
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(destDir, "config.json"), data, 0o644)
}

func (c *ConfigManager) restoreModConfig(modsRoot, modID, uniqueID string) error {
	profileID := c.service.ActiveID()
	src := filepath.Join(c.profilesDir, profileID, "configs", uniqueID, "config.json")
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}
	folderPath, _ := splitModID(modID)
	destDir := filepath.Join(modsRoot, filepath.FromSlash(folderPath))
	_ = os.MkdirAll(destDir, 0o755)
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(destDir, "config.json"), data, 0o644)
}

func splitModID(modID string) (folderPath, uniqueID string) {
	for i := 0; i < len(modID); i++ {
		if i+1 < len(modID) && modID[i] == ':' && modID[i+1] == ':' {
			return modID[:i], modID[i+2:]
		}
	}
	return modID, ""
}
