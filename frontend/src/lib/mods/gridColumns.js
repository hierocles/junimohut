export const GRID_COLUMNS = [
    { id: 'enabled', label: 'Enabled' },
    { id: 'name', label: 'Name', required: true },
    { id: 'tags', label: 'Tags' },
    { id: 'author', label: 'Author' },
    { id: 'version', label: 'Version' },
    { id: 'folder', label: 'Folder' },
    { id: 'status', label: 'Status' },
];
export const DEFAULT_VISIBLE_COLUMNS = GRID_COLUMNS.map((c) => c.id);
const COLUMN_ORDER = new Map(GRID_COLUMNS.map((c, i) => [c.id, i]));
export function normalizeVisibleColumns(raw) {
    if (!raw?.length)
        return [...DEFAULT_VISIBLE_COLUMNS];
    const selected = new Set();
    for (const id of raw) {
        if (COLUMN_ORDER.has(id))
            selected.add(id);
    }
    selected.add('name');
    return GRID_COLUMNS.filter((c) => selected.has(c.id)).map((c) => c.id);
}
export function isColumnVisible(visible, id) {
    return normalizeVisibleColumns(visible).includes(id);
}
export function toggleVisibleColumn(current, id, visible) {
    const def = GRID_COLUMNS.find((c) => c.id === id);
    if (def?.required)
        return normalizeVisibleColumns(current);
    const next = new Set(normalizeVisibleColumns(current));
    if (visible)
        next.add(id);
    else
        next.delete(id);
    next.add('name');
    return GRID_COLUMNS.filter((c) => next.has(c.id)).map((c) => c.id);
}
export function visibleColumnCount(raw) {
    return normalizeVisibleColumns(raw).length;
}
//# sourceMappingURL=gridColumns.js.map