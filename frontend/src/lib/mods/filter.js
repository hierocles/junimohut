/** Tag visibility filter — display only; does not change mod enabled state. */
export function filterByCategories(mods, categories) {
    if (categories.length === 0)
        return mods;
    const visible = categories.filter((c) => c.visible);
    if (visible.length === 0 || visible.length === categories.length)
        return mods;
    const visibleIds = new Set(visible.map((c) => c.id));
    return mods.filter((m) => (m.categoryIds ?? []).some((id) => visibleIds.has(id)));
}
export function modHasUpdateAvailable(mod) {
    const state = mod.updateStatus?.state;
    return state === "update" || state === "update_available";
}
export function modHasDependencyIssues(mod) {
    return (mod.missingDependencyCount ?? 0) > 0;
}
export function filterByGridStatus(mods, filter) {
    if (filter === "none")
        return mods;
    if (filter === "updates")
        return mods.filter(modHasUpdateAvailable);
    return mods.filter(modHasDependencyIssues);
}
export function applyModFilters(mods, categories, gridStatusFilter) {
    return filterByGridStatus(filterByCategories(mods, categories), gridStatusFilter);
}
//# sourceMappingURL=filter.js.map