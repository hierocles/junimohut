package mods

// GroupingMode constants for mod list organization.
const (
	GroupingFolder          = "folder"
	GroupingFolderCondensed = "folder_condensed"
	GroupingContentPack     = "contentpack"
)

// ValidGrouping returns whether a grouping mode is supported.
func ValidGrouping(mode string) bool {
	switch mode {
	case GroupingFolder, GroupingFolderCondensed, GroupingContentPack:
		return true
	default:
		return false
	}
}
