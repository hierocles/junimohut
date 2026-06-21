import type { Mod } from "$lib/api/client";

export type DependencyRow = {
  uniqueID: string;
  minimumVersion: string;
  isRequired: boolean;
  isContentPack: boolean;
  state: "satisfied" | "missing" | "version_too_low" | "disabled" | "optional";
  installedName?: string;
  installedVersion?: string;
  providerModId?: string;
  nexusModId?: string;
};

export type DependentRow = {
  modId: string;
  name: string;
  uniqueID: string;
  version: string;
  enabled: boolean;
  minimumVersion: string;
  isRequired: boolean;
  isContentPack: boolean;
  nexusModId?: string;
};

export type InstallDependencyPreview = {
  archivePath: string;
  modName: string;
  uniqueID: string;
  issues: NonNullable<Mod["dependencyIssues"]>;
};

type DepEntry = {
  uniqueID: string;
  minimumVersion: string;
  isRequired: boolean;
  isContentPack: boolean;
};

function parseVersion(version: string): number[] | null {
  const cleaned = version.trim().replace(/^v/i, "");
  const parts = cleaned.split(/[.-]/).map((p) => parseInt(p, 10));
  if (parts.some((n) => Number.isNaN(n))) return null;
  return parts;
}

function versionSatisfies(
  installedVersion: string,
  minimumVersion: string,
): boolean {
  const minimum = minimumVersion.trim();
  if (!minimum) return true;
  const installed = installedVersion.trim();
  if (!installed) return false;

  const instParts = parseVersion(installed);
  const minParts = parseVersion(minimum);
  if (!instParts || !minParts) return true;

  const len = Math.max(instParts.length, minParts.length);
  for (let i = 0; i < len; i++) {
    const a = instParts[i] ?? 0;
    const b = minParts[i] ?? 0;
    if (a > b) return true;
    if (a < b) return false;
  }
  return true;
}

function nexusModIDFromManifest(mod: Mod): string {
  for (const key of mod.manifest?.UpdateKeys ?? []) {
    if (key.startsWith("Nexus:")) return key.slice("Nexus:".length);
  }
  return "";
}

function canonicalUniqueID(uid: string): string {
  return uid.toLowerCase();
}

function uniqueIDsEqual(a: string, b: string): boolean {
  return a.localeCompare(b, undefined, { sensitivity: "accent" }) === 0;
}

function collectDependencyEntries(mod: Mod): DepEntry[] {
  const seen = new Map<string, DepEntry>();

  const contentPack = mod.manifest?.ContentPackFor;
  if (contentPack?.UniqueID) {
    seen.set(canonicalUniqueID(contentPack.UniqueID), {
      uniqueID: contentPack.UniqueID,
      minimumVersion: contentPack.MinimumVersion ?? "",
      isRequired: true,
      isContentPack: true,
    });
  }

  for (const dep of mod.manifest?.Dependencies ?? []) {
    if (!dep.UniqueID) continue;
    const required = dep.IsRequired ?? true;
    const key = canonicalUniqueID(dep.UniqueID);
    const existing = seen.get(key);
    if (!existing) {
      seen.set(key, {
        uniqueID: dep.UniqueID,
        minimumVersion: dep.MinimumVersion ?? "",
        isRequired: required,
        isContentPack: false,
      });
      continue;
    }
    if (required && !existing.isRequired) existing.isRequired = true;
    if (!existing.minimumVersion && dep.MinimumVersion) {
      existing.minimumVersion = dep.MinimumVersion;
    }
  }

  return [...seen.values()];
}

function newDependencyIssue(
  entry: DepEntry,
  state: NonNullable<Mod["dependencyIssues"]>[number]["state"],
  provider?: Mod,
): NonNullable<Mod["dependencyIssues"]>[number] {
  const issue: NonNullable<Mod["dependencyIssues"]>[number] = {
    uniqueID: entry.uniqueID,
    minimumVersion: entry.minimumVersion,
    isRequired: entry.isRequired,
    isContentPack: entry.isContentPack,
    state,
  };
  if (provider) {
    issue.installedName = provider.manifest?.Name;
    issue.installedVersion = provider.manifest?.Version;
    issue.providerModId = provider.id;
    issue.nexusModId = nexusModIDFromManifest(provider);
  }
  return issue;
}

/** Client-side dependency resolution for mock mode and detail-pane rows. */
export function resolveDependencies(mods: Mod[]): Mod[] {
  const byUniqueID = modIndexByUniqueID(mods);

  return mods.map((mod) => {
    const issues = resolveModIssues(mod, byUniqueID);
    return {
      ...mod,
      dependencyIssues: issues,
      missingDependencyCount: issues.length,
    };
  });
}

function modIndexByUniqueID(mods: Mod[]): Map<string, Mod> {
  const byUniqueID = new Map<string, Mod>();
  for (const mod of mods) {
    registerModInUniqueIDIndex(byUniqueID, mod.manifest?.UniqueID ?? "", mod);
    for (const uid of mod.packSiblingUIDs ?? []) {
      registerModInUniqueIDIndex(byUniqueID, uid, mod);
    }
  }
  return byUniqueID;
}

function registerModInUniqueIDIndex(
  byUniqueID: Map<string, Mod>,
  uid: string,
  mod: Mod,
): void {
  if (!uid) return;
  const key = canonicalUniqueID(uid);
  if (byUniqueID.has(key)) return;
  byUniqueID.set(key, mod);
}

function resolveModIssues(
  mod: Mod,
  byUniqueID: Map<string, Mod>,
): NonNullable<Mod["dependencyIssues"]> {
  const selfID = mod.manifest?.UniqueID ?? "";
  const issues: NonNullable<Mod["dependencyIssues"]> = [];

  for (const entry of collectDependencyEntries(mod)) {
    if (!entry.uniqueID || uniqueIDsEqual(entry.uniqueID, selfID)) continue;

    const provider = byUniqueID.get(canonicalUniqueID(entry.uniqueID));
    if (!provider) {
      if (entry.isRequired) {
        issues.push(newDependencyIssue(entry, "missing"));
      }
      continue;
    }

    const installedVersion = provider.manifest?.Version ?? "";
    if (
      entry.minimumVersion &&
      !versionSatisfies(installedVersion, entry.minimumVersion)
    ) {
      if (entry.isRequired) {
        issues.push(newDependencyIssue(entry, "version_too_low", provider));
      }
      continue;
    }
    if (!provider.enabled && entry.isRequired) {
      issues.push(newDependencyIssue(entry, "disabled", provider));
    }
  }

  return issues;
}

function issueToRowState(state: string): DependencyRow["state"] {
  if (state === "version_too_low") return "version_too_low";
  if (state === "disabled") return "disabled";
  if (state === "missing") return "missing";
  return "missing";
}

export function dependencyRowsForMod(
  mod: Mod,
  allMods: Mod[],
): DependencyRow[] {
  const selfID = mod.manifest?.UniqueID ?? "";
  const byUniqueID = modIndexByUniqueID(allMods);

  const issueByID = new Map(
    (mod.dependencyIssues ?? []).map((issue) => [issue.uniqueID, issue]),
  );

  return collectDependencyEntries(mod).map((entry) => {
    const issue = issueByID.get(entry.uniqueID);
    if (issue) {
      return {
        uniqueID: entry.uniqueID,
        minimumVersion: entry.minimumVersion,
        isRequired: entry.isRequired,
        isContentPack: entry.isContentPack,
        state: issueToRowState(issue.state),
        installedName: issue.installedName,
        installedVersion: issue.installedVersion,
        providerModId: issue.providerModId,
        nexusModId: issue.nexusModId,
      };
    }

    const provider = byUniqueID.get(canonicalUniqueID(entry.uniqueID));
    if (provider && !uniqueIDsEqual(entry.uniqueID, selfID)) {
      return {
        uniqueID: entry.uniqueID,
        minimumVersion: entry.minimumVersion,
        isRequired: entry.isRequired,
        isContentPack: entry.isContentPack,
        state: "satisfied",
        installedName: provider.manifest?.Name,
        installedVersion: provider.manifest?.Version,
        providerModId: provider.id,
        nexusModId: nexusModIDFromManifest(provider),
      };
    }

    return {
      uniqueID: entry.uniqueID,
      minimumVersion: entry.minimumVersion,
      isRequired: entry.isRequired,
      isContentPack: entry.isContentPack,
      state: entry.isRequired ? "missing" : "optional",
    };
  });
}

/** Mods in the library that declare a dependency on the given mod's Unique ID. */
export function dependentRowsForMod(mod: Mod, allMods: Mod[]): DependentRow[] {
  const providerID = mod.manifest?.UniqueID;
  if (!providerID) return [];

  const rows: DependentRow[] = [];
  for (const candidate of allMods) {
    if (candidate.id === mod.id) continue;

    for (const entry of collectDependencyEntries(candidate)) {
      if (!uniqueIDsEqual(entry.uniqueID, providerID)) continue;
      rows.push({
        modId: candidate.id,
        name: candidate.manifest?.Name ?? candidate.folderPath,
        uniqueID: candidate.manifest?.UniqueID ?? "",
        version: candidate.manifest?.Version ?? "",
        enabled: candidate.enabled,
        minimumVersion: entry.minimumVersion,
        isRequired: entry.isRequired,
        isContentPack: entry.isContentPack,
        nexusModId: nexusModIDFromManifest(candidate) || undefined,
      });
      break;
    }
  }

  return rows.sort((a, b) => a.name.localeCompare(b.name));
}

export function countModsWithDependencyIssues(mods: Mod[]): number {
  return mods.filter((m) => (m.missingDependencyCount ?? 0) > 0).length;
}

export function nexusSearchUrl(uniqueID: string): string {
  return `https://www.nexusmods.com/stardewvalley/mods/?keyword=${encodeURIComponent(uniqueID)}`;
}

export function nexusModPageUrl(nexusModId: string): string {
  return `https://www.nexusmods.com/stardewvalley/mods/${nexusModId}`;
}

export function previewInstallDependenciesMock(
  archivePaths: string[],
  library: Mod[],
): InstallDependencyPreview[] {
  void archivePaths;
  return library
    .filter((m) => (m.missingDependencyCount ?? 0) > 0)
    .slice(0, 1)
    .map((m) => ({
      archivePath: archivePaths[0] ?? "",
      modName: m.manifest?.Name ?? m.folderPath,
      uniqueID: m.manifest?.UniqueID ?? "",
      issues: m.dependencyIssues ?? [],
    }));
}
