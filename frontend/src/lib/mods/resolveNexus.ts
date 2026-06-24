import type { Mod } from "$lib/api/client";

function nexusIdFromUpdateKeys(keys: string[] | null | undefined): number {
  for (const key of keys ?? []) {
    if (key.startsWith("Nexus:")) {
      const id = Number.parseInt(key.slice("Nexus:".length), 10);
      if (Number.isFinite(id) && id > 0) return id;
    }
  }
  return 0;
}

/** Best Nexus mod ID for page links and downloads (manifest → dataset → bundle). */
export function resolvedNexusModId(mod: Mod): number {
  if (mod.resolvedNexusModId && mod.resolvedNexusModId > 0) {
    return mod.resolvedNexusModId;
  }
  const fromManifest = nexusIdFromUpdateKeys(mod.manifest?.UpdateKeys);
  if (fromManifest > 0) return fromManifest;
  if (mod.bundleNexusId && mod.bundleNexusId > 0) return mod.bundleNexusId;
  return 0;
}
