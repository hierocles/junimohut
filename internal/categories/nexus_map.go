package categories

import "strings"

// TagIDForNexusCategory maps a Nexus Mods page category name to a default tag ID.
// Returns empty string when there is no suitable SDVM tag.
func TagIDForNexusCategory(name string) string {
	key := strings.ToLower(strings.TrimSpace(name))
	key = strings.Join(strings.Fields(key), " ")
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
