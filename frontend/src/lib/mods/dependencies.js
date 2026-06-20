function parseVersion(version) {
    const cleaned = version.trim().replace(/^v/i, "");
    const parts = cleaned.split(/[.-]/).map((p) => parseInt(p, 10));
    if (parts.some((n) => Number.isNaN(n)))
        return null;
    return parts;
}
function versionSatisfies(installedVersion, minimumVersion) {
    const minimum = minimumVersion.trim();
    if (!minimum)
        return true;
    const installed = installedVersion.trim();
    if (!installed)
        return false;
    const instParts = parseVersion(installed);
    const minParts = parseVersion(minimum);
    if (!instParts || !minParts)
        return true;
    const len = Math.max(instParts.length, minParts.length);
    for (let i = 0; i < len; i++) {
        const a = instParts[i] ?? 0;
        const b = minParts[i] ?? 0;
        if (a > b)
            return true;
        if (a < b)
            return false;
    }
    return true;
}
function nexusModIDFromManifest(mod) {
    for (const key of mod.manifest?.UpdateKeys ?? []) {
        if (key.startsWith("Nexus:"))
            return key.slice("Nexus:".length);
    }
    return "";
}
function collectDependencyEntries(mod) {
    const seen = new Map();
    const contentPack = mod.manifest?.ContentPackFor;
    if (contentPack?.UniqueID) {
        seen.set(contentPack.UniqueID, {
            uniqueID: contentPack.UniqueID,
            minimumVersion: contentPack.MinimumVersion ?? "",
            isRequired: true,
            isContentPack: true,
        });
    }
    for (const dep of mod.manifest?.Dependencies ?? []) {
        if (!dep.UniqueID)
            continue;
        const required = dep.IsRequired ?? true;
        const existing = seen.get(dep.UniqueID);
        if (!existing) {
            seen.set(dep.UniqueID, {
                uniqueID: dep.UniqueID,
                minimumVersion: dep.MinimumVersion ?? "",
                isRequired: required,
                isContentPack: false,
            });
            continue;
        }
        if (required && !existing.isRequired)
            existing.isRequired = true;
        if (!existing.minimumVersion && dep.MinimumVersion) {
            existing.minimumVersion = dep.MinimumVersion;
        }
    }
    return [...seen.values()];
}
function newDependencyIssue(entry, state, provider) {
    const issue = {
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
export function resolveDependencies(mods) {
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
function modIndexByUniqueID(mods) {
    const byUniqueID = new Map();
    for (const mod of mods) {
        const uid = mod.manifest?.UniqueID;
        if (!uid || byUniqueID.has(uid))
            continue;
        byUniqueID.set(uid, mod);
    }
    return byUniqueID;
}
function resolveModIssues(mod, byUniqueID) {
    const selfID = mod.manifest?.UniqueID ?? "";
    const issues = [];
    for (const entry of collectDependencyEntries(mod)) {
        if (!entry.uniqueID || entry.uniqueID === selfID)
            continue;
        const provider = byUniqueID.get(entry.uniqueID);
        if (!provider) {
            if (entry.isRequired) {
                issues.push(newDependencyIssue(entry, "missing"));
            }
            continue;
        }
        const installedVersion = provider.manifest?.Version ?? "";
        if (entry.minimumVersion &&
            !versionSatisfies(installedVersion, entry.minimumVersion)) {
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
function issueToRowState(state) {
    if (state === "version_too_low")
        return "version_too_low";
    if (state === "disabled")
        return "disabled";
    if (state === "missing")
        return "missing";
    return "missing";
}
export function dependencyRowsForMod(mod, allMods) {
    const selfID = mod.manifest?.UniqueID ?? "";
    const byUniqueID = modIndexByUniqueID(allMods);
    const issueByID = new Map((mod.dependencyIssues ?? []).map((issue) => [issue.uniqueID, issue]));
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
        const provider = byUniqueID.get(entry.uniqueID);
        if (provider && entry.uniqueID !== selfID) {
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
export function dependentRowsForMod(mod, allMods) {
    const providerID = mod.manifest?.UniqueID;
    if (!providerID)
        return [];
    const rows = [];
    for (const candidate of allMods) {
        if (candidate.id === mod.id)
            continue;
        for (const entry of collectDependencyEntries(candidate)) {
            if (entry.uniqueID !== providerID)
                continue;
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
export function countModsWithDependencyIssues(mods) {
    return mods.filter((m) => (m.missingDependencyCount ?? 0) > 0).length;
}
export function nexusSearchUrl(uniqueID) {
    return `https://www.nexusmods.com/stardewvalley/mods/?keyword=${encodeURIComponent(uniqueID)}`;
}
export function nexusModPageUrl(nexusModId) {
    return `https://www.nexusmods.com/stardewvalley/mods/${nexusModId}`;
}
export function previewInstallDependenciesMock(archivePaths, library) {
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
//# sourceMappingURL=dependencies.js.map