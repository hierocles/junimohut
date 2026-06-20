import type { Category } from "../../../bindings/junimohut/internal/categories/models";

/** Stable default tag IDs — must match internal/categories/defaults.go */
export const DEFAULT_TAG_IDS = {
  qol: "tag-qol",
  ui: "tag-ui",
  visual: "tag-visual",
  characters: "tag-characters",
  maps: "tag-maps",
  items: "tag-items",
  farming: "tag-farming",
  gameplay: "tag-gameplay",
  expansions: "tag-expansions",
  audio: "tag-audio",
  framework: "tag-framework",
  cheats: "tag-cheats",
} as const;

export type DefaultTagKey = keyof typeof DEFAULT_TAG_IDS;

export const DEFAULT_TAGS: Omit<Category, "modIds">[] = [
  {
    id: DEFAULT_TAG_IDS.qol,
    name: "Quality of Life",
    color: "#10b981",
    visible: true,
    sortOrder: 0,
  },
  {
    id: DEFAULT_TAG_IDS.ui,
    name: "UI & HUD",
    color: "#0ea5e9",
    visible: true,
    sortOrder: 1,
  },
  {
    id: DEFAULT_TAG_IDS.visual,
    name: "Visual & Graphics",
    color: "#8b5cf6",
    visible: true,
    sortOrder: 2,
  },
  {
    id: DEFAULT_TAG_IDS.characters,
    name: "Characters & Social",
    color: "#d946ef",
    visible: true,
    sortOrder: 3,
  },
  {
    id: DEFAULT_TAG_IDS.maps,
    name: "Maps & Locations",
    color: "#5b8a8a",
    visible: true,
    sortOrder: 4,
  },
  {
    id: DEFAULT_TAG_IDS.items,
    name: "Items & Crafting",
    color: "#64748b",
    visible: true,
    sortOrder: 5,
  },
  {
    id: DEFAULT_TAG_IDS.farming,
    name: "Farming & Livestock",
    color: "#22c55e",
    visible: true,
    sortOrder: 6,
  },
  {
    id: DEFAULT_TAG_IDS.gameplay,
    name: "Gameplay Mechanics",
    color: "#f59e0b",
    visible: true,
    sortOrder: 7,
  },
  {
    id: DEFAULT_TAG_IDS.expansions,
    name: "Expansions & Overhauls",
    color: "#ef4444",
    visible: true,
    sortOrder: 8,
  },
  {
    id: DEFAULT_TAG_IDS.audio,
    name: "Audio",
    color: "#06b6d4",
    visible: true,
    sortOrder: 9,
  },
  {
    id: DEFAULT_TAG_IDS.framework,
    name: "Framework & Libraries",
    color: "#4f46e5",
    visible: true,
    sortOrder: 10,
  },
  {
    id: DEFAULT_TAG_IDS.cheats,
    name: "Cheats & Unbalanced",
    color: "#f97316",
    visible: true,
    sortOrder: 11,
  },
];

export function buildDefaultCategories(
  modIdsByTag: Record<string, string[]>,
): Category[] {
  return DEFAULT_TAGS.map((tag) => ({
    ...tag,
    modIds: modIdsByTag[tag.id] ?? [],
  }));
}
