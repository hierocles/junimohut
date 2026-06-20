/** Returns mods deduplicated by id, preserving first occurrence order. */
export function dedupeMods(mods) {
    const seen = new Set();
    const out = [];
    for (const mod of mods) {
        if (seen.has(mod.id))
            continue;
        seen.add(mod.id);
        out.push(mod);
    }
    return out;
}
/** When the same SMAPI UniqueID appears under multiple folders, keep one entry. */
export function dedupeModsByUniqueID(mods) {
    const seen = new Set();
    const out = [];
    for (const mod of mods) {
        const uid = mod.manifest?.UniqueID ?? mod.id;
        if (seen.has(uid))
            continue;
        seen.add(uid);
        out.push(mod);
    }
    return out;
}
//# sourceMappingURL=dedupe.js.map