import type { Mod } from "$lib/api/client";

export type BundleCheckState = "all" | "none" | "partial";

export function isBundleMod(mod: Mod): boolean {
  return (mod.bundleChildren?.length ?? 0) > 0;
}

export function bundleCheckState(mod: Mod): BundleCheckState {
  const total = mod.enabledTotal ?? mod.bundleChildren?.length ?? 0;
  const enabled = mod.enabledCount ?? 0;
  if (total === 0) return mod.enabled ? "all" : "none";
  if (enabled === 0) return "none";
  if (enabled >= total) return "all";
  return "partial";
}

export function bundleIsChecked(mod: Mod): boolean {
  return bundleCheckState(mod) === "all";
}

export function bundleIsIndeterminate(mod: Mod): boolean {
  return bundleCheckState(mod) === "partial";
}

export function bundleIsFullyDisabled(mod: Mod): boolean {
  return bundleCheckState(mod) === "none";
}

export function bundlePartCount(mod: Mod): number {
  return mod.bundleChildren?.length ?? mod.enabledTotal ?? 0;
}

export function bundleDeleteFolderPaths(mod: Mod): string[] {
  if (!isBundleMod(mod)) {
    return mod.folderPath ? [mod.folderPath] : [];
  }
  return (mod.bundleChildren ?? [])
    .map((child) => child.folderPath)
    .filter((path) => path.length > 0);
}

export function bundlePartTypeLabel(mod: Mod): string {
  if (mod.manifest?.EntryDll) return "C#";
  const cpFor = mod.manifest?.ContentPackFor?.UniqueID ?? "";
  if (cpFor.includes("AlternativeTextures")) return "AT";
  if (cpFor) return "CP";
  return "Mod";
}

export function bundleFolderLabel(mod: Mod): string {
  const count = bundlePartCount(mod);
  if (!isBundleMod(mod)) return mod.folderPath;
  if (mod.folderPath) return mod.folderPath;
  return count === 1
    ? (mod.bundleChildren?.[0]?.folderPath ?? "")
    : `${count} folders`;
}

export function bundleVersionLabel(mod: Mod): string {
  if (!isBundleMod(mod)) return mod.manifest?.Version ?? "";
  const versions = new Set<string>();
  for (const child of mod.bundleChildren ?? []) {
    const v = child.manifest?.Version?.trim();
    if (v) versions.add(v);
  }
  if (versions.size === 0) return mod.manifest?.Version ?? "";
  if (versions.size === 1) return [...versions][0];
  return [...versions].sort().join(" · ");
}

export type GridDisplayRow =
  | { kind: "parent"; mod: Mod }
  | { kind: "child"; mod: Mod; parentId: string };

export function buildGridDisplayRows(
  mods: Mod[],
  expandedBundleIds: ReadonlySet<string>,
  searchQuery = "",
): GridDisplayRow[] {
  const query = searchQuery.trim().toLowerCase();
  const rows: GridDisplayRow[] = [];

  for (const mod of mods) {
    if (!isBundleMod(mod)) {
      rows.push({ kind: "parent", mod });
      continue;
    }

    const childMatches =
      query.length > 0
        ? (mod.bundleChildren ?? []).filter((child) =>
            modMatchesSearch(child, query),
          )
        : [];
    const expanded = expandedBundleIds.has(mod.id) || childMatches.length > 0;

    rows.push({ kind: "parent", mod });
    if (expanded) {
      for (const child of mod.bundleChildren ?? []) {
        rows.push({ kind: "child", mod: child, parentId: mod.id });
      }
    }
  }

  return rows;
}

function modMatchesSearch(mod: Mod, query: string): boolean {
  const hay = [
    mod.customName,
    mod.manifest?.Name,
    mod.manifest?.Author,
    mod.manifest?.UniqueID,
    mod.folderPath,
  ]
    .filter(Boolean)
    .join(" ")
    .toLowerCase();
  return hay.includes(query);
}

const EXPANDED_STORAGE_KEY = "sdvm-bundle-expanded";

export function loadExpandedBundleIds(): Set<string> {
  try {
    const raw = localStorage.getItem(EXPANDED_STORAGE_KEY);
    if (!raw) return new Set();
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return new Set();
    return new Set(parsed.filter((id): id is string => typeof id === "string"));
  } catch {
    return new Set();
  }
}

export function saveExpandedBundleIds(ids: Set<string>) {
  try {
    localStorage.setItem(EXPANDED_STORAGE_KEY, JSON.stringify([...ids]));
  } catch {
    /* storage unavailable */
  }
}

export function findModInList(mods: Mod[], modId: string): Mod | undefined {
  for (const mod of mods) {
    if (mod.id === modId) return mod;
    if (isBundleMod(mod)) {
      const child = mod.bundleChildren?.find((part) => part.id === modId);
      if (child) return child;
    }
  }
  return undefined;
}

/** Pick the mod row that owns config.json / JSON files for editing. */
export function configTargetMod(mod: Mod): Mod | undefined {
  if (!mod.hasJsonFiles && !mod.hasConfig) return undefined;
  if (!isBundleMod(mod)) return mod;
  if (mod.folderPath && mod.hasJsonFiles) return mod;
  const childWithJson = mod.bundleChildren?.find(
    (part) => part.hasJsonFiles || part.hasConfig,
  );
  return childWithJson ?? (mod.hasJsonFiles ? mod : undefined);
}

export function findBundleParent(mods: Mod[], modId: string): Mod | undefined {
  for (const mod of mods) {
    if (!isBundleMod(mod)) continue;
    if (mod.id === modId) return mod;
    if (mod.bundleChildren?.some((child) => child.id === modId)) return mod;
  }
  return undefined;
}

/** True when mod is a bundle part row, not the synthetic parent. */
export function isBundleChildMod(mods: Mod[], mod: Mod): boolean {
  if (isBundleMod(mod)) return false;
  return findBundleParent(mods, mod.id) !== undefined;
}

/** Bundle parents own Nexus update state; parts route through the parent. */
export function bundleUpdateTarget(mods: Mod[], mod: Mod): Mod {
  if (isBundleMod(mod)) return mod;
  return findBundleParent(mods, mod.id) ?? mod;
}

function nexusIdFromUpdateKeys(keys: string[] | null | undefined): number {
  if (!keys) return 0;
  for (const key of keys) {
    const match = /^Nexus:(\d+)$/i.exec(key.trim());
    if (match) return Number.parseInt(match[1], 10);
  }
  return 0;
}

/** Mirrors backend bundle collapse for mock data and tests. */
export function collapseDisplayMods(mods: Mod[]): Mod[] {
  const byNexus = new Map<number, Mod[]>();
  for (const mod of mods) {
    if (mod.isCoreMod) continue;
    const nexusId = nexusIdFromUpdateKeys(mod.manifest?.UpdateKeys ?? null);
    if (!nexusId) continue;
    const group = byNexus.get(nexusId) ?? [];
    group.push(mod);
    byNexus.set(nexusId, group);
  }

  const collapseIds = new Set<number>();
  for (const [nexusId, group] of byNexus) {
    if (group.length >= 2) collapseIds.add(nexusId);
  }
  if (collapseIds.size === 0) return mods;

  const emitted = new Set<number>();
  const out: Mod[] = [];
  for (const mod of mods) {
    const nexusId = nexusIdFromUpdateKeys(mod.manifest?.UpdateKeys ?? null);
    if (nexusId && collapseIds.has(nexusId)) {
      if (emitted.has(nexusId)) continue;
      emitted.add(nexusId);
      const children = [...(byNexus.get(nexusId) ?? [])].sort((a, b) =>
        a.folderPath.localeCompare(b.folderPath),
      );
      const enabledCount = children.filter((child) => child.enabled).length;
      const bundleChildren = children.map((child) => ({
        ...child,
        updateStatus: {},
      }));
      out.push({
        ...children[0],
        id: `${children[0].folderPath || ""}::pack:nexus:${nexusId}`,
        folderPath: commonFolderPrefix(children.map((c) => c.folderPath)),
        manifest: {
          ...children[0].manifest,
          Name: bundleDisplayNameFromChildren(children),
          UniqueID: `pack:nexus:${nexusId}`,
          EntryDll: "",
          UpdateKeys: children[0].manifest?.UpdateKeys ?? [`Nexus:${nexusId}`],
        },
        enabled: enabledCount > 0,
        enabledPartial: enabledCount > 0 && enabledCount < children.length,
        enabledCount,
        enabledTotal: children.length,
        bundleChildren,
        bundleNexusId: nexusId,
        packSiblingUIDs: children
          .map((child) => child.manifest?.UniqueID)
          .filter((uid): uid is string => Boolean(uid)),
      });
      continue;
    }
    out.push(mod);
  }
  return out;
}

function commonFolderPrefix(paths: string[]): string {
  if (paths.length === 0) return "";
  const split = (path: string) => (path ? path.split("/") : []);
  let parts = split(paths[0]);
  for (const path of paths.slice(1)) {
    const other = split(path);
    let i = 0;
    while (i < parts.length && i < other.length && parts[i] === other[i]) i++;
    parts = parts.slice(0, i);
    if (parts.length === 0) return "";
  }
  return parts.join("/");
}

function bundleDisplayNameFromChildren(children: Mod[]): string {
  const prefix = commonFolderPrefix(children.map((child) => child.folderPath));
  if (prefix) {
    const base = prefix.split("/").pop() ?? prefix;
    return base.replace(/^\(AT\)\s+/, "").trim();
  }
  const names = children.map(
    (child) => child.customName || child.manifest?.Name || child.folderPath,
  );
  return names[0] ?? "Mod bundle";
}
