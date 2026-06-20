package mods

// Manifest represents a SMAPI mod manifest.json.
type Manifest struct {
	Name                 string          `json:"Name"`
	Author               string          `json:"Author"`
	Version              string          `json:"Version"`
	Description          string          `json:"Description"`
	UniqueID             string          `json:"UniqueID"`
	EntryDll             string          `json:"EntryDll"`
	UpdateKeys           []string        `json:"UpdateKeys"`
	ContentPackFor       *ContentPackFor `json:"ContentPackFor"`
	UpdateCautionMessage string          `json:"UpdateCautionMessage"`
	Dependencies         []ModDependency `json:"Dependencies"`
}

type ContentPackFor struct {
	UniqueID       string `json:"UniqueID"`
	MinimumVersion string `json:"MinimumVersion"`
}

type ModDependency struct {
	UniqueID       string    `json:"UniqueID"`
	MinimumVersion string    `json:"MinimumVersion"`
	IsRequired     *flexBool `json:"IsRequired"` // nil = required (SMAPI default)
}

// DependencyIssue describes an unsatisfied mod dependency.
type DependencyIssue struct {
	UniqueID         string `json:"uniqueID"`
	MinimumVersion   string `json:"minimumVersion"`
	IsRequired       bool   `json:"isRequired"`
	IsContentPack    bool   `json:"isContentPack"`
	State            string `json:"state"` // missing, version_too_low, disabled
	InstalledName    string `json:"installedName,omitempty"`
	InstalledVersion string `json:"installedVersion,omitempty"`
	ProviderModID    string `json:"providerModId,omitempty"`
	NexusModID       string `json:"nexusModId,omitempty"`
}

// InstallDependencyPreview describes dependency warnings for a mod in an install queue.
type InstallDependencyPreview struct {
	ArchivePath string            `json:"archivePath"`
	ModName     string            `json:"modName"`
	UniqueID    string            `json:"uniqueID"`
	Issues      []DependencyIssue `json:"issues"`
}

// UpdateStatus describes mod update state.
type UpdateStatus struct {
	State         string `json:"state"` // current, update_available, incompatible, unofficial
	LatestVersion string `json:"latestVersion"`
	ModPageURL    string `json:"modPageUrl"`
	Message       string `json:"message"`
}

// Mod is a discovered mod with runtime state.
type Mod struct {
	ID           string       `json:"id"`
	FolderPath   string       `json:"folderPath"` // relative to mods root
	AbsolutePath string       `json:"absolutePath"`
	Manifest     Manifest     `json:"manifest"`
	Enabled      bool         `json:"enabled"`
	CategoryIDs  []string     `json:"categoryIds"`
	GroupKey     string       `json:"groupKey"`
	GroupLabel   string       `json:"groupLabel"`
	UpdateStatus UpdateStatus `json:"updateStatus"`
	HasConfig    bool         `json:"hasConfig"`
	IsCoreMod    bool         `json:"isCoreMod"`
	InstallTime            int64             `json:"installTime"`
	LastUpdated            int64             `json:"lastUpdated"`
	DependencyIssues       []DependencyIssue `json:"dependencyIssues"`
	MissingDependencyCount int               `json:"missingDependencyCount"`
	SavedDownloadPath      string            `json:"savedDownloadPath,omitempty"`
}

// ModGroup is a collection of mods for UI grouping.
type ModGroup struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Mods  []Mod  `json:"mods"`
}

// DeleteModsResult summarizes bulk mod deletion.
type DeleteModsResult struct {
	DeletedCount         int      `json:"deletedCount"`
	ArchivesDeletedCount int      `json:"archivesDeletedCount"`
	Errors               []string `json:"errors,omitempty"`
}

// SMAPI core mods that are always enabled.
var CoreModIDs = map[string]bool{
	"Pathoschild.ConsoleCommands": true,
	"Pathoschild.ErrorHandler":    true,
	"Pathoschild.SaveBackup":      true,
}
