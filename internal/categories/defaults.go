package categories

// DefaultCategories returns the standard Stardew Valley mod tags seeded on first run.
func DefaultCategories() []Category {
	return []Category{
		{ID: "tag-qol", Name: "Quality of Life", Color: "#10b981", Visible: true, SortOrder: 0, ModIDs: []string{}},
		{ID: "tag-ui", Name: "UI & HUD", Color: "#0ea5e9", Visible: true, SortOrder: 1, ModIDs: []string{}},
		{ID: "tag-visual", Name: "Visual & Graphics", Color: "#8b5cf6", Visible: true, SortOrder: 2, ModIDs: []string{}},
		{ID: "tag-characters", Name: "Characters & Social", Color: "#d946ef", Visible: true, SortOrder: 3, ModIDs: []string{}},
		{ID: "tag-maps", Name: "Maps & Locations", Color: "#5b8a8a", Visible: true, SortOrder: 4, ModIDs: []string{}},
		{ID: "tag-items", Name: "Items & Crafting", Color: "#64748b", Visible: true, SortOrder: 5, ModIDs: []string{}},
		{ID: "tag-farming", Name: "Farming & Livestock", Color: "#22c55e", Visible: true, SortOrder: 6, ModIDs: []string{}},
		{ID: "tag-gameplay", Name: "Gameplay Mechanics", Color: "#f59e0b", Visible: true, SortOrder: 7, ModIDs: []string{}},
		{ID: "tag-expansions", Name: "Expansions & Overhauls", Color: "#ef4444", Visible: true, SortOrder: 8, ModIDs: []string{}},
		{ID: "tag-audio", Name: "Audio", Color: "#06b6d4", Visible: true, SortOrder: 9, ModIDs: []string{}},
		{ID: "tag-framework", Name: "Framework & Libraries", Color: "#4f46e5", Visible: true, SortOrder: 10, ModIDs: []string{}},
		{ID: "tag-cheats", Name: "Cheats & Unbalanced", Color: "#f97316", Visible: true, SortOrder: 11, ModIDs: []string{}},
		{ID: "tag-fashion-sense", Name: "Fashion Sense", Color: "#ec4899", Visible: true, SortOrder: 12, ModIDs: []string{}},
	}
}
