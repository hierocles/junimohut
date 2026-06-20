import type { Mod } from "$lib/api/client";

export const DISPLAY_NAME_OFFICIAL = "official";
export const DISPLAY_NAME_FOLDER = "folder";

export function officialModName(mod: Mod): string {
  return mod.manifest?.Name ?? mod.folderPath;
}

function folderDisplayName(mod: Mod): string {
  const official = officialModName(mod);
  const parts = mod.folderPath.split("/").filter(Boolean);
  const leaf = parts[parts.length - 1];
  if (!leaf) return "";
  if (
    leaf.localeCompare(official, undefined, { sensitivity: "accent" }) === 0
  ) {
    return "";
  }
  return leaf;
}

/** Name shown in the mod grid (custom from backend, or official). */
export function displayModName(mod: Mod): string {
  const custom = mod.customName?.trim();
  if (custom) return custom;
  return officialModName(mod);
}

/** True when the user saved a custom display name that is not the default. */
export function hasUserCustomName(mod: Mod): boolean {
  const custom = mod.customName?.trim();
  if (!custom) return false;
  const official = officialModName(mod);
  const folder = folderDisplayName(mod);
  if (
    custom.localeCompare(official, undefined, { sensitivity: "accent" }) === 0
  ) {
    return false;
  }
  if (
    folder &&
    custom.localeCompare(folder, undefined, { sensitivity: "accent" }) === 0
  ) {
    return false;
  }
  return true;
}

export function inferDisplayName(
  folderPath: string,
  manifestName: string,
): string {
  const official = manifestName.trim() || folderPath;
  const parts = folderPath.split("/").filter(Boolean);
  const leaf = parts[parts.length - 1];
  if (!leaf) return "";
  if (
    leaf.localeCompare(official, undefined, { sensitivity: "accent" }) === 0
  ) {
    return "";
  }
  return leaf;
}

export function effectiveCustomName(
  storedCustomName: string | undefined,
  folderPath: string,
  manifestName: string,
  source: string,
): string | undefined {
  const stored = storedCustomName?.trim();
  if (stored) return stored;
  if (source !== DISPLAY_NAME_FOLDER) return undefined;
  return inferDisplayName(folderPath, manifestName) || undefined;
}
