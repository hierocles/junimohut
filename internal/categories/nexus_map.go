package categories

import "strings"

func normalizeNexusCategory(name string) string {
	key := strings.ToLower(strings.TrimSpace(name))
	return strings.Join(strings.Fields(key), " ")
}

// NexusCategoryDefersUntilManifest reports Nexus categories whose tag mapping
// may be overridden once install archives are scanned (e.g. Clothing → FS pack).
func NexusCategoryDefersUntilManifest(name string) bool {
	return normalizeNexusCategory(name) == "clothing"
}
// Returns empty string when there is no suitable SDVM tag.
func TagIDForNexusCategory(name string) string {
	key := normalizeNexusCategory(name)
	switch key {
	case "user interface":
		return "tag-ui"
	case "visuals and graphics", "portraits":
		return "tag-visual"
	case "characters", "new characters", "dialogue", "events":
		return "tag-characters"
	case "maps", "locations", "interiors", "buildings":
		return "tag-maps"
	case "items", "crafting", "furniture", "clothing":
		return "tag-items"
	case "crops", "livestock and animals", "fishing", "pets / horses", "pets/horses":
		return "tag-farming"
	case "gameplay mechanics", "player":
		return "tag-gameplay"
	case "expansions":
		return "tag-expansions"
	case "audio":
		return "tag-audio"
	case "modding tools":
		return "tag-framework"
	case "cheats":
		return "tag-cheats"
	default:
		return ""
	}
}
