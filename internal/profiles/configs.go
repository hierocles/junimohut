package profiles

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"junimohut/internal/mods"
)

// ConfigManager handles profile-specific mod JSON config files.
type ConfigManager struct {
	profilesDir string
	service     *Service
}

func NewConfigManager(profilesDir string, service *Service) *ConfigManager {
	return &ConfigManager{profilesDir: profilesDir, service: service}
}

// SaveModConfig backs up JSON config files for a mod for the active profile.
func (c *ConfigManager) SaveModConfig(modsRoot, modID, uniqueID string) error {
	return c.saveModConfigFiles(modsRoot, modID, uniqueID)
}

// RestoreModConfig restores JSON config files for a mod for the active profile.
func (c *ConfigManager) RestoreModConfig(modsRoot, modID, uniqueID string) error {
	return c.restoreModConfigFiles(modsRoot, modID, uniqueID)
}

// SaveConfigs backs up JSON config files for mods in modUniqueIDs (mod ID -> UniqueID).
func (c *ConfigManager) SaveConfigs(modsRoot string, modUniqueIDs map[string]string) error {
	if c.service.ActiveID() == "" {
		return nil
	}
	for modID, uniqueID := range modUniqueIDs {
		_ = c.saveModConfigFiles(modsRoot, modID, uniqueID)
	}
	return nil
}

// RestoreConfigs restores JSON config files for mods in modUniqueIDs.
func (c *ConfigManager) RestoreConfigs(modsRoot string, modUniqueIDs map[string]string) error {
	if c.service.ActiveID() == "" {
		return nil
	}
	for modID, uniqueID := range modUniqueIDs {
		_ = c.restoreModConfigFiles(modsRoot, modID, uniqueID)
	}
	return nil
}

func (c *ConfigManager) saveModConfigFiles(modsRoot, modID, uniqueID string) error {
	profileID := c.service.ActiveID()
	if profileID == "" {
		return nil
	}
	folderPath, _ := splitModID(modID)
	modDir := mods.ModDir(modsRoot, folderPath)
	paths, err := mods.ListJsonFileRelPaths(modDir)
	if err != nil || len(paths) == 0 {
		return nil
	}
	destBase := filepath.Join(c.profilesDir, profileID, "configs", uniqueID)
	for _, rel := range paths {
		src := filepath.Join(modDir, filepath.FromSlash(rel))
		dest := filepath.Join(destBase, filepath.FromSlash(rel))
		if err := copyFile(src, dest); err != nil {
			return err
		}
	}
	return nil
}

func (c *ConfigManager) restoreModConfigFiles(modsRoot, modID, uniqueID string) error {
	profileID := c.service.ActiveID()
	if profileID == "" {
		return nil
	}
	srcBase := filepath.Join(c.profilesDir, profileID, "configs", uniqueID)
	if _, err := os.Stat(srcBase); os.IsNotExist(err) {
		return nil
	}
	folderPath, _ := splitModID(modID)
	destModDir := mods.ModDir(modsRoot, folderPath)
	var paths []string
	err := filepath.WalkDir(srcBase, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".json") {
			return nil
		}
		rel, err := filepath.Rel(srcBase, path)
		if err != nil {
			return nil
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return err
	}
	for _, rel := range paths {
		src := filepath.Join(srcBase, filepath.FromSlash(rel))
		dest := filepath.Join(destModDir, filepath.FromSlash(rel))
		if err := copyFile(src, dest); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func splitModID(modID string) (folderPath, uniqueID string) {
	for i := 0; i < len(modID); i++ {
		if i+1 < len(modID) && modID[i] == ':' && modID[i+1] == ':' {
			return modID[:i], modID[i+2:]
		}
	}
	return modID, ""
}
