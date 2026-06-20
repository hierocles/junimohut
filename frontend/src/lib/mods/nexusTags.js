import { DEFAULT_TAG_IDS } from "$lib/mods/defaultTags";
/** Mirror of internal/categories/nexus_map.go */
const NEXUS_CATEGORY_TO_TAG = {
    "user interface": DEFAULT_TAG_IDS.ui,
    "visuals and graphics": DEFAULT_TAG_IDS.visual,
    portraits: DEFAULT_TAG_IDS.visual,
    characters: DEFAULT_TAG_IDS.characters,
    "new characters": DEFAULT_TAG_IDS.characters,
    dialogue: DEFAULT_TAG_IDS.characters,
    events: DEFAULT_TAG_IDS.characters,
    maps: DEFAULT_TAG_IDS.maps,
    locations: DEFAULT_TAG_IDS.maps,
    interiors: DEFAULT_TAG_IDS.maps,
    buildings: DEFAULT_TAG_IDS.maps,
    items: DEFAULT_TAG_IDS.items,
    crafting: DEFAULT_TAG_IDS.items,
    furniture: DEFAULT_TAG_IDS.items,
    clothing: DEFAULT_TAG_IDS.items,
    crops: DEFAULT_TAG_IDS.farming,
    "livestock and animals": DEFAULT_TAG_IDS.farming,
    fishing: DEFAULT_TAG_IDS.farming,
    "pets / horses": DEFAULT_TAG_IDS.farming,
    "pets/horses": DEFAULT_TAG_IDS.farming,
    "gameplay mechanics": DEFAULT_TAG_IDS.gameplay,
    player: DEFAULT_TAG_IDS.gameplay,
    expansions: DEFAULT_TAG_IDS.expansions,
    audio: DEFAULT_TAG_IDS.audio,
    "modding tools": DEFAULT_TAG_IDS.framework,
    cheats: DEFAULT_TAG_IDS.cheats,
};
/** Known Nexus mod IDs in mock data → Nexus page category name. */
const MOCK_NEXUS_MOD_CATEGORIES = {
    509: "User Interface",
    1529: "Gameplay Mechanics",
    2010: "User Interface",
    5098: "User Interface",
    1089: "Gameplay Mechanics",
    1401: "Livestock and Animals",
    5588: "Visuals and Graphics",
    3387: "Visuals and Graphics",
    5226: "Visuals and Graphics",
    4644: "Expansions",
    7286: "Expansions",
    5735: "Visuals and Graphics",
    7846: "Modding Tools",
    7332: "Maps",
    6254: "Fishing",
    10212: "Items",
    7742: "Characters",
    3753: "Expansions",
    3109: "Modding Tools",
    1915: "Modding Tools",
    1720: "Modding Tools",
    1348: "Modding Tools",
};
export function parseNxmModId(raw) {
    const url = raw.trim();
    if (!url.toLowerCase().startsWith("nxm://"))
        return null;
    const parts = url.slice("nxm://".length).split("/");
    for (let i = 0; i < parts.length; i++) {
        if (parts[i] === "mods" && parts[i + 1]) {
            const id = Number.parseInt(parts[i + 1], 10);
            return Number.isFinite(id) && id > 0 ? id : null;
        }
    }
    return null;
}
export function nexusModIdFromUpdateKey(key) {
    if (!key.startsWith("Nexus:"))
        return null;
    const id = Number.parseInt(key.slice("Nexus:".length), 10);
    return Number.isFinite(id) && id > 0 ? id : null;
}
export function tagIdForNexusCategory(name) {
    const key = name.trim().toLowerCase().replace(/\s+/g, " ");
    return NEXUS_CATEGORY_TO_TAG[key] ?? "";
}
export function suggestedTagIdsForNexusMods(modIds, existingCategoryIds) {
    const out = [];
    const seen = new Set();
    for (const modId of modIds) {
        const categoryName = MOCK_NEXUS_MOD_CATEGORIES[modId];
        if (!categoryName)
            continue;
        const tagId = tagIdForNexusCategory(categoryName);
        if (!tagId || seen.has(tagId) || !existingCategoryIds.has(tagId))
            continue;
        seen.add(tagId);
        out.push(tagId);
    }
    return out;
}
//# sourceMappingURL=nexusTags.js.map