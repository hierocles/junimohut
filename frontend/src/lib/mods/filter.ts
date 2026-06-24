import type { Category, Mod } from "$lib/api/client";

export type GridStatusFilter =
  | "none"
  | "updates"
  | "dependencies"
  | "incompatible";

/** Tag visibility filter — display only; does not change mod enabled state. */
export function filterByCategories(mods: Mod[], categories: Category[]): Mod[] {
  if (categories.length === 0) return mods;

  const visible = categories.filter((c) => c.visible);
  if (visible.length === 0 || visible.length === categories.length) return mods;

  const visibleIds = new Set(visible.map((c) => c.id));
  return mods.filter((m) =>
    (m.categoryIds ?? []).some((id) => visibleIds.has(id)),
  );
}

export function modHasUpdateAvailable(mod: Mod): boolean {
  const state = mod.updateStatus?.state;
  return state === "update" || state === "update_available";
}

export function modHasDependencyIssues(mod: Mod): boolean {
  return (mod.missingDependencyCount ?? 0) > 0;
}

export function modIsIncompatible(mod: Mod): boolean {
  return mod.updateStatus?.state === "incompatible";
}

export function filterByGridStatus(
  mods: Mod[],
  filter: GridStatusFilter,
): Mod[] {
  if (filter === "none") return mods;
  if (filter === "updates") return mods.filter(modHasUpdateAvailable);
  if (filter === "incompatible") return mods.filter(modIsIncompatible);
  return mods.filter(modHasDependencyIssues);
}

export function applyModFilters(
  mods: Mod[],
  categories: Category[],
  gridStatusFilter: GridStatusFilter,
): Mod[] {
  return filterByGridStatus(
    filterByCategories(mods, categories),
    gridStatusFilter,
  );
}
