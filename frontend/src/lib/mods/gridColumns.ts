export type GridColumnId =
  | "enabled"
  | "name"
  | "tags"
  | "author"
  | "version"
  | "folder"
  | "installed"
  | "status";

export type GridColumnDef = {
  id: GridColumnId;
  label: string;
  required?: boolean;
};

export const GRID_COLUMNS: GridColumnDef[] = [
  { id: "enabled", label: "Enabled" },
  { id: "name", label: "Name", required: true },
  { id: "tags", label: "Tags" },
  { id: "author", label: "Author" },
  { id: "version", label: "Version" },
  { id: "folder", label: "Folder" },
  { id: "installed", label: "Installed" },
  { id: "status", label: "Status" },
];

export const DEFAULT_VISIBLE_COLUMNS: GridColumnId[] = GRID_COLUMNS.map(
  (c) => c.id,
);

const COLUMN_ORDER = new Map(GRID_COLUMNS.map((c, i) => [c.id, i]));

export function normalizeVisibleColumns(
  raw: string[] | null | undefined,
): GridColumnId[] {
  if (!raw?.length) return [...DEFAULT_VISIBLE_COLUMNS];
  const selected = new Set<GridColumnId>();
  for (const id of raw) {
    if (COLUMN_ORDER.has(id as GridColumnId)) selected.add(id as GridColumnId);
  }
  selected.add("name");
  return GRID_COLUMNS.filter((c) => selected.has(c.id)).map((c) => c.id);
}

export function isColumnVisible(
  visible: GridColumnId[] | string[] | null | undefined,
  id: GridColumnId,
): boolean {
  return normalizeVisibleColumns(visible).includes(id);
}

export function toggleVisibleColumn(
  current: string[] | null | undefined,
  id: GridColumnId,
  visible: boolean,
): GridColumnId[] {
  const def = GRID_COLUMNS.find((c) => c.id === id);
  if (def?.required) return normalizeVisibleColumns(current);

  const next = new Set(normalizeVisibleColumns(current));
  if (visible) next.add(id);
  else next.delete(id);
  next.add("name");
  return GRID_COLUMNS.filter((c) => next.has(c.id)).map((c) => c.id);
}

export function visibleColumnCount(raw: string[] | null | undefined): number {
  return normalizeVisibleColumns(raw).length;
}
