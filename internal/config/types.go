package config

// Settings holds application configuration persisted to disk.
type Settings struct {
	GamePath                string   `json:"gamePath"`
	SMAPIPath               string   `json:"smapiPath"`
	ModsRoot                string   `json:"modsRoot"`
	IgnoreHiddenFolders     bool     `json:"ignoreHiddenFolders"`
	ProfileSpecificConfigs  bool     `json:"profileSpecificConfigs"`
	AutoEnableOnInstall     bool     `json:"autoEnableOnInstall"`
	Theme                   string   `json:"theme"`
	Language                string   `json:"language"`
	ShowThumbnails          bool     `json:"showThumbnails"`
	AutoSaveProfileChanges  bool     `json:"autoSaveProfileChanges"`
	AlwaysAskDeleteOnUpdate bool     `json:"alwaysAskDeleteOnUpdate"`
	ModGrouping             string   `json:"modGrouping"` // folder, contentpack, folder_condensed
	HideDisabledFilter      string   `json:"hideDisabledFilter"`
	VisibleColumns          []string `json:"visibleColumns"`
	WindowWidth             int      `json:"windowWidth"`
	WindowHeight            int      `json:"windowHeight"`
	SetupComplete           bool     `json:"setupComplete"`
	LastUpdateCheck         int64    `json:"lastUpdateCheck"`
	NexusAPIKey string `json:"-"` // stored in keyring when set
}

// DefaultSettings returns sensible defaults for a fresh install.
func DefaultSettings() Settings {
	return Settings{
		IgnoreHiddenFolders:    true,
		ProfileSpecificConfigs: false,
		AutoEnableOnInstall:    true,
		Theme:                  "stardew-dark",
		Language:               "en",
		AutoSaveProfileChanges: true,
		ModGrouping:            "folder",
		HideDisabledFilter:     "none",
		VisibleColumns:         []string{"enabled", "name", "tags", "author", "version", "folder", "installed", "status"},
		WindowWidth:            1430,
		WindowHeight:           900,
	}
}
