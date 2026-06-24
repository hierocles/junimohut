import type { Category } from "../../../bindings/junimohut/internal/categories/models";
import type { Mod } from "$lib/api/client";
import {
  type InstallDependencyPreview,
  resolveDependencies,
  countModsWithDependencyIssues,
} from "$lib/mods/dependencies";
import { pathBasename } from "$lib/copy";
import { collapseDisplayMods } from "$lib/mods/bundles";
import type { Profile } from "../../../bindings/junimohut/internal/profiles/models";
import type { Settings } from "../../../bindings/junimohut/internal/config/models";
import type { DownloadRecord } from "../../../bindings/junimohut/internal/nexus/models";
import {
  DEFAULT_TAG_IDS,
  buildDefaultCategories,
  type DefaultTagKey,
} from "$lib/mods/defaultTags";

const MODS_ROOT = "C:/Games/Stardew Valley/Mods";

export const USE_MOCK_DATA = import.meta.env.VITE_USE_MOCK_DATA === "true";

type ModSeed = {
  folderPath: string;
  uniqueID: string;
  name: string;
  author: string;
  version: string;
  description?: string;
  category: DefaultTagKey;
  enabled?: boolean;
  hasConfig?: boolean;
  isCoreMod?: boolean;
  update?: { latestVersion: string; message?: string };
  incompatible?: { message: string };
  contentPackFor?: { uniqueID: string; minimumVersion: string };
  dependencies?: Array<{
    uniqueID: string;
    minimumVersion?: string;
    isRequired?: boolean;
  }>;
  nexusId?: string;
  savedDownloadPath?: string;
  customName?: string;
};

const MOD_SEEDS: ModSeed[] = [
  // Quality of Life (6)
  {
    folderPath: "LookupAnything",
    uniqueID: "Candidus42.LookupAnything",
    name: "Lookup Anything",
    customName: "Lookup",
    author: "Candidus42",
    version: "1.47.0",
    description:
      "Shows live info about anything you hover over or hold a button on.",
    category: "qol",
    hasConfig: true,
    nexusId: "509",
    savedDownloadPath:
      "C:/Users/Example/AppData/JunimoHut/downloads/LookupAnything.zip",
  },
  {
    folderPath: "SkipIntro",
    uniqueID: "Cat.SkipIntro",
    name: "Skip Intro",
    author: "Cat",
    version: "1.1.0",
    category: "qol",
    nexusId: "1529",
  },
  {
    folderPath: "BetterShopMenu",
    uniqueID: "cat.bettershopmenu",
    name: "Better Shop Menu",
    author: "cat",
    version: "1.6.2",
    category: "ui",
    nexusId: "2010",
  },
  {
    folderPath: "UIInfoSuite2",
    uniqueID: "AnnEntis.UISuite",
    name: "UI Info Suite 2",
    author: "AnnEntis",
    version: "2.3.1",
    category: "ui",
    hasConfig: true,
    enabled: false,
    nexusId: "5098",
  },
  {
    folderPath: "FastAnimations",
    uniqueID: "spacechase0.FastAnimations",
    name: "Fast Animations",
    author: "spacechase0",
    version: "1.11.2",
    category: "qol",
    nexusId: "1089",
  },
  {
    folderPath: "TractorMod",
    uniqueID: "Pathoschild.TractorMod",
    name: "Tractor Mod",
    author: "Pathoschild",
    version: "4.20.0",
    description:
      "Buy a tractor to more efficiently work with crops, clear twigs and rocks, and more.",
    category: "farming",
    hasConfig: true,
    update: {
      latestVersion: "4.21.0",
      message: "Fixes multiplayer tool sync in 1.6.",
    },
    nexusId: "1401",
  },

  // Visual & Graphics (6)
  {
    folderPath: "[CP] SeasonalOutfits",
    uniqueID: "Poltergeinx.SeasonalOutfits",
    name: "Seasonal Villager Outfits",
    author: "Poltergeinx",
    version: "3.0.1",
    category: "visual",
    contentPackFor: {
      uniqueID: "Pathoschild.ContentPatcher",
      minimumVersion: "2.0.0",
    },
    nexusId: "5588",
  },
  {
    folderPath: "[CP] StarblueValley",
    uniqueID: "Lita.StarblueValley",
    name: "Starblue Valley",
    author: "Lita",
    version: "2.1.4",
    category: "visual",
    contentPackFor: {
      uniqueID: "Pathoschild.ContentPatcher",
      minimumVersion: "2.0.0",
    },
    nexusId: "3387",
  },
  {
    folderPath: "[CP] VibrantPastoral",
    uniqueID: "gramplet.vibrantpastoral",
    name: "Vibrant Pastoral",
    author: "gramplet",
    version: "1.0.6",
    category: "visual",
    contentPackFor: {
      uniqueID: "Pathoschild.ContentPatcher",
      minimumVersion: "2.0.0",
    },
    nexusId: "5226",
  },
  {
    folderPath: "EastScarp",
    uniqueID: "LemurKat.EastScarp",
    name: "East Scarp",
    author: "LemurKat",
    version: "2.4.8",
    category: "expansions",
    update: { latestVersion: "2.5.0" },
    nexusId: "4644",
  },
  {
    folderPath: "RidgesideVillage",
    uniqueID: "Rafseazz.RSV",
    name: "Ridgeside Village",
    author: "Rafseazz",
    version: "2.5.2",
    category: "expansions",
    nexusId: "7286",
  },
  {
    folderPath: "[CP] AnimatedFish",
    uniqueID: "GZhyn.AnimatedFish",
    name: "Animated Fish",
    author: "GZhyn",
    version: "1.2.0",
    category: "visual",
    enabled: false,
    contentPackFor: {
      uniqueID: "Pathoschild.ContentPatcher",
      minimumVersion: "2.0.0",
    },
    nexusId: "5735",
  },

  // Gameplay & content (6)
  {
    folderPath: "MailFrameworkMod",
    uniqueID: "DIGUS.MailFrameworkMod",
    name: "Mail Framework Mod",
    author: "DIGUS",
    version: "1.18.0",
    category: "framework",
    hasConfig: true,
    nexusId: "7846",
  },
  {
    folderPath: "WalkToDesert",
    uniqueID: "LemurKat.WalkToDesert",
    name: "Walk to the Desert",
    author: "LemurKat",
    version: "1.3.2",
    category: "maps",
    nexusId: "7332",
  },
  {
    folderPath: "FishingTrawler",
    uniqueID: "spacechase0.FishingTrawler",
    name: "Fishing Trawler",
    author: "spacechase0",
    version: "3.18.1",
    category: "farming",
    nexusId: "6254",
  },
  {
    folderPath: "RefinedAgedSpirit",
    uniqueID: "Rafseazz.RefinedAgedSpirit",
    name: "Refined Aged Spirit",
    author: "Rafseazz",
    version: "1.1.0",
    category: "items",
    nexusId: "10212",
  },
  {
    folderPath: "ImmersiveMarnie",
    uniqueID: "LemurKat.ImmersiveMarnie",
    name: "Immersive Marnie",
    author: "LemurKat",
    version: "1.2.0",
    category: "characters",
    enabled: false,
    incompatible: {
      message: "Requires Stardew Valley 1.6.14 or later.",
    },
    nexusId: "7742",
  },
  {
    folderPath: "StardewValleyExpanded",
    uniqueID: "FlashShifter.StardewValleyExpandedCP",
    name: "Stardew Valley Expanded",
    author: "FlashShifter",
    version: "1.14.23",
    category: "expansions",
    hasConfig: true,
    update: { latestVersion: "1.14.24" },
    nexusId: "3753",
  },

  // Framework & APIs (7)
  {
    folderPath: "[SMAPI] ConsoleCommands",
    uniqueID: "Pathoschild.ConsoleCommands",
    name: "Console Commands",
    author: "Pathoschild",
    version: "4.0.0",
    category: "framework",
    isCoreMod: true,
    nexusId: "3109",
  },
  {
    folderPath: "[SMAPI] ErrorHandler",
    uniqueID: "Pathoschild.ErrorHandler",
    name: "Error Handler",
    author: "Pathoschild",
    version: "4.0.0",
    category: "framework",
    isCoreMod: true,
    nexusId: "3109",
  },
  {
    folderPath: "[SMAPI] SaveBackup",
    uniqueID: "Pathoschild.SaveBackup",
    name: "Save Backup",
    author: "Pathoschild",
    version: "4.0.0",
    category: "framework",
    isCoreMod: true,
    nexusId: "3109",
  },
  {
    folderPath: "ContentPatcher",
    uniqueID: "Pathoschild.ContentPatcher",
    name: "Content Patcher",
    author: "Pathoschild",
    version: "2.4.4",
    category: "framework",
    nexusId: "1915",
  },
  {
    folderPath: "JsonAssets",
    uniqueID: "spacechase0.JsonAssets",
    name: "Json Assets",
    author: "spacechase0",
    version: "1.11.2",
    category: "framework",
    nexusId: "1720",
  },
  {
    folderPath: "SpaceCore",
    uniqueID: "spacechase0.SpaceCore",
    name: "SpaceCore",
    author: "spacechase0",
    version: "1.19.0",
    category: "framework",
    enabled: false,
    nexusId: "1348",
  },
  {
    folderPath: "GenericModConfigMenu",
    uniqueID: "spacechase0.GenericModConfigMenu",
    name: "Generic Mod Config Menu",
    author: "spacechase0",
    version: "1.14.0",
    category: "framework",
    hasConfig: true,
    nexusId: "5098",
  },

  // Dependency demo mods
  {
    folderPath: "ScheduleViewer",
    uniqueID: "Demo.ScheduleViewer",
    name: "Schedule Viewer (Demo)",
    author: "Demo",
    version: "1.0.0",
    category: "qol",
    dependencies: [
      { uniqueID: "Demo.MissingLibrary", isRequired: true },
      { uniqueID: "Demo.OptionalMissing", isRequired: false },
    ],
  },
  {
    folderPath: "BrokenContentPack",
    uniqueID: "Demo.BrokenContentPack",
    name: "Broken Content Pack (Demo)",
    author: "Demo",
    version: "1.0.0",
    category: "visual",
    contentPackFor: {
      uniqueID: "Demo.MissingFramework",
      minimumVersion: "1.0.0",
    },
  },
  {
    folderPath: "HighCPRequirement",
    uniqueID: "Demo.HighCPRequirement",
    name: "High CP Requirement (Demo)",
    author: "Demo",
    version: "1.0.0",
    category: "framework",
    dependencies: [
      {
        uniqueID: "Pathoschild.ContentPatcher",
        minimumVersion: "99.0.0",
        isRequired: true,
      },
    ],
  },
  {
    folderPath: "NeedsSpaceCore",
    uniqueID: "Demo.NeedsSpaceCore",
    name: "Needs SpaceCore (Demo)",
    author: "Demo",
    version: "1.0.0",
    category: "framework",
    dependencies: [{ uniqueID: "spacechase0.SpaceCore", isRequired: true }],
  },
];

function buildMod(seed: ModSeed, enabledOverrides: Map<string, boolean>): Mod {
  const id = `${seed.folderPath}::${seed.uniqueID}`;
  const enabled = enabledOverrides.get(id) ?? seed.enabled ?? true;

  return {
    id,
    folderPath: seed.folderPath,
    absolutePath: `${MODS_ROOT}/${seed.folderPath}`,
    manifest: {
      Name: seed.name,
      Author: seed.author,
      Version: seed.version,
      Description: seed.description ?? "",
      UniqueID: seed.uniqueID,
      EntryDll: seed.contentPackFor
        ? ""
        : `${seed.name.replace(/\s/g, "")}.dll`,
      UpdateKeys: seed.nexusId ? [`Nexus:${seed.nexusId}`] : null,
      ContentPackFor: seed.contentPackFor
        ? {
            UniqueID: seed.contentPackFor.uniqueID,
            MinimumVersion: seed.contentPackFor.minimumVersion,
          }
        : null,
      UpdateCautionMessage: "",
      Dependencies: seed.dependencies
        ? seed.dependencies.map((dep) => ({
            UniqueID: dep.uniqueID,
            MinimumVersion: dep.minimumVersion ?? "",
            IsRequired: dep.isRequired ?? false,
          }))
        : null,
    },
    enabled: seed.isCoreMod ? true : enabled,
    categoryIds: [DEFAULT_TAG_IDS[seed.category]],
    updateStatus: seed.incompatible
      ? {
          state: "incompatible",
          latestVersion: "",
          modPageUrl: seed.nexusId
            ? `https://www.nexusmods.com/stardewvalley/mods/${seed.nexusId}`
            : "",
          message: seed.incompatible.message,
        }
      : seed.update
        ? {
            state: "update_available",
            latestVersion: seed.update.latestVersion,
            modPageUrl: seed.nexusId
              ? `https://www.nexusmods.com/stardewvalley/mods/${seed.nexusId}`
              : "",
            message: seed.update.message ?? "",
          }
        : { state: "current", latestVersion: "", modPageUrl: "", message: "" },
    hasConfig: seed.hasConfig ?? false,
    hasJsonFiles: seed.hasConfig ?? false,
    jsonFileCount: seed.hasConfig ? 1 : 0,
    isCoreMod: seed.isCoreMod ?? false,
    installTime: 1704067200,
    lastUpdated: 1711929600,
    dependencyIssues: [],
    missingDependencyCount: 0,
    containsOverwrites: false,
    savedDownloadPath: seed.savedDownloadPath,
    customName: resolvedCustomName(id, seed),
  };
}

function resolvedCustomName(id: string, seed: ModSeed): string | undefined {
  if (customNameOverrides.has(id)) {
    const value = customNameOverrides.get(id) ?? "";
    return value.trim() || undefined;
  }
  return seed.customName?.trim() || undefined;
}

function buildCategories(mods: Mod[]): Category[] {
  const byCategory: Record<string, string[]> = Object.fromEntries(
    Object.values(DEFAULT_TAG_IDS).map((id) => [id, [] as string[]]),
  );
  for (const mod of mods) {
    for (const catId of mod.categoryIds ?? []) {
      if (byCategory[catId]) {
        byCategory[catId].push(mod.id);
      }
    }
  }
  return buildDefaultCategories(byCategory);
}

const enabledOverrides = new Map<string, boolean>();
const customNameOverrides = new Map<string, string>();

function allMods(): Mod[] {
  return collapseDisplayMods(
    resolveDependencies(
      MOD_SEEDS.map((seed) => buildMod(seed, enabledOverrides)),
    ),
  );
}

export function filterMods(
  mods: Mod[],
  search: string,
  hideDisabled: string,
): Mod[] {
  const q = search.toLowerCase().trim();
  return mods.filter((m) => {
    if (hideDisabled === "enabled" && !m.enabled) return false;
    if (hideDisabled === "disabled" && m.enabled) return false;
    if (!q) return true;
    const hay =
      `${m.manifest?.Name} ${m.customName ?? ""} ${m.manifest?.Author} ${m.manifest?.UniqueID} ${m.folderPath}`.toLowerCase();
    return hay.includes(q);
  });
}

import { filterByCategories } from "$lib/mods/filter";

export const MOCK_PROFILES: Profile[] = [
  { id: "profile-main", name: "Main Farm", isActive: true, enabledMods: {} },
  { id: "profile-coop", name: "Co-op Lite", isActive: false, enabledMods: {} },
  {
    id: "profile-seasonal",
    name: "Seasonal Playthrough",
    isActive: false,
    enabledMods: {},
  },
];

export const MOCK_SETTINGS: Settings = {
  gamePath: "C:/Games/Stardew Valley",
  smapiPath: "C:/Games/Stardew Valley/StardewModdingAPI.exe",
  modsRoot: MODS_ROOT,
  ignoreHiddenFolders: true,
  profileSpecificConfigs: true,
  autoEnableOnInstall: true,
  theme: "stardew-dark",
  language: "en",
  showThumbnails: false,
  autoSaveProfileChanges: true,
  alwaysAskDeleteOnUpdate: false,
  showInstallSummary: true,
  hideDisabledFilter: "none",
  visibleColumns: [
    "enabled",
    "name",
    "tags",
    "author",
    "version",
    "folder",
    "installed",
    "status",
  ],
  windowWidth: 1430,
  windowHeight: 900,
  setupComplete: true,
  lastUpdateCheck: 1711929600,
  ignoredModUpdates: {},
};

export function getMockRefreshData(search: string, hideDisabled: string) {
  const mods = allMods();
  const categories = buildCategories(mods);
  const filtered = filterMods(mods, search, hideDisabled);
  const readyCount = filtered.filter(
    (m) =>
      m.updateStatus?.state === "update_available" ||
      m.updateStatus?.state === "update",
  ).length;
  const dependencyIssueCount = countModsWithDependencyIssues(mods);
  const incompatibleCount = mods.filter(
    (m) => m.updateStatus?.state === "incompatible",
  ).length;

  return {
    mods: filtered,
    profiles: MOCK_PROFILES,
    categories,
    settings: MOCK_SETTINGS,
    smapiVersion: "4.0.0",
    readyCount,
    dependencyIssueCount,
    incompatibleCount,
    unmanagedMods: [],
    duplicateMods: [],
  };
}

export function getMockInstallDependencyPreview(
  paths: string[],
): InstallDependencyPreview[] {
  if (paths.length === 0) return [];
  return [
    {
      archivePath: paths[0],
      modName: pathBasename(paths[0]),
      uniqueID: "pending.install",
      issues: [
        {
          uniqueID: "Demo.MissingLibrary",
          minimumVersion: "",
          isRequired: true,
          isContentPack: false,
          state: "missing",
        },
      ],
    },
  ];
}

export function setMockModEnabled(modId: string, enabled: boolean) {
  enabledOverrides.set(modId, enabled);
}

export function setMockModCustomName(modId: string, name: string) {
  const trimmed = name.trim();
  if (trimmed) customNameOverrides.set(modId, trimmed);
  else customNameOverrides.delete(modId);
}

export function getMockInstallNamePreview(paths: string[]) {
  return paths.map((archivePath) => {
    const base = pathBasename(archivePath).toLowerCase();
    if (!base.includes("seasonal") && !base.includes("open-windows")) {
      return {
        archivePath,
        needsDisplayNameChoice: false,
        mods: [
          {
            officialName: pathBasename(archivePath).replace(/\.[^.]+$/, ""),
            folderLabel: pathBasename(archivePath).replace(/\.[^.]+$/, ""),
            destFolder: pathBasename(archivePath).replace(/\.[^.]+$/, ""),
            uniqueID: "mock.install",
          },
        ],
      };
    }
    return {
      archivePath,
      needsDisplayNameChoice: true,
      mods: [
        {
          officialName: "[CP] Seasonal Open Windows",
          folderLabel: "[CP] Seasonal Open Windows",
          destFolder: "[CP] Seasonal Open Windows",
          uniqueID: "OB7.SOWindows",
        },
        {
          officialName: "[CP] Seasonal Open Windows",
          folderLabel: "[CP] Seasonal Open Windows - BIRCH",
          destFolder: "[CP] Seasonal Open Windows - BIRCH",
          uniqueID: "OB7.SOWindows.birch",
        },
      ],
    };
  });
}

export function getMockSavedDownloads(): DownloadRecord[] {
  const now = Math.floor(Date.now() / 1000);
  return [
    {
      archivePath:
        "C:/Users/Example/AppData/JunimoHut/downloads/LookupAnything.zip",
      fileName: "LookupAnything.zip",
      uniqueId: "Candidus42.LookupAnything",
      nexusModId: 509,
      downloadedAt: now - 86400 * 2,
    },
    {
      archivePath:
        "C:/Users/Example/AppData/JunimoHut/downloads/ContentPatcher.zip",
      fileName: "ContentPatcher.zip",
      uniqueId: "Pathoschild.ContentPatcher",
      nexusModId: 1915,
      downloadedAt: now - 86400 * 10,
    },
    {
      archivePath:
        "C:/Users/Example/AppData/JunimoHut/downloads/orphan-pack.zip",
      fileName: "orphan-pack.zip",
      uniqueId: "",
      nexusModId: 0,
      downloadedAt: now - 3600 * 6,
    },
  ];
}
