<script lang="ts">
  import { tick } from "svelte";
  import { ChevronDown, ChevronRight, ChevronUp } from "@lucide/svelte";
  import { createAnnouncer } from "@sv-kit/a11y-keys";
  import type { Category, Mod } from "$lib/api/client";
  import { modStatusInfo, modStatusSortKey } from "$lib/mods/modStatus";
  import ModGridHeaderMenu from "$lib/components/ModGridHeaderMenu.svelte";
  import {
    MOD_GRID_DROP_ID,
    pathsFromDataTransfer,
    useNativeArchiveFileDrop,
  } from "$lib/wails/archiveFileDrop";
  import TagPicker from "$lib/components/TagPicker.svelte";
  import {
    emptyLibraryState,
    workspaceOnboardingText,
    gridUpdatesFilterEmptyTitle,
    gridUpdatesFilterEmptyHint,
    gridDependencyFilterEmptyTitle,
    gridDependencyFilterEmptyHint,
    gridUpdatesFilterLabel,
    gridDependencyFilterLabel,
    gridTagsLearnEmphasis,
    gridTagsLearnHint,
    tagsCellAddLabel,
    tagsCellEditLabel,
    tagsOverflowLabel,
    gridBulkDeleteLabel,
    modContainsOverwritesLabel,
    modContainsOverwritesTooltip,
    gridTagsFilterEmptyTitle,
    gridTagsFilterEmptyHint,
    gridClearFilter,
    gridQuickStartLabel,
    gridUpdatesFilterMeta,
    gridDependencyFilterMeta,
    gridTagsFilteringMeta,
    gridTagsFilteringBadge,
    gridBulkSelectedLabel,
    gridBulkEnableSelected,
    gridBulkDisableSelected,
    gridBulkClearSelection,
    gridBulkShiftHint,
    gridBulkKeyboardHint,
    gridSortClearedAnnounce,
    gridSelectionClearedAnnounce,
    gridBulkDeleteOpeningAnnounce,
    learnHintDismissLabel,
    gridBundlePartsLabel,
    gridBundleExpandLabel,
    gridBundleCollapseLabel,
  } from "$lib/copy";
  import {
    buildGridDisplayRows,
    bundleFolderLabel,
    bundleIsChecked,
    bundleIsFullyDisabled,
    bundleIsIndeterminate,
    bundlePartCount,
    bundlePartTypeLabel,
    bundleVersionLabel,
    isBundleMod,
    loadExpandedBundleIds,
    saveExpandedBundleIds,
    type GridDisplayRow,
  } from "$lib/mods/bundles";
  import emptyStateIllustration from "$lib/assets/brand/empty-state-illustration.svg?raw";
  import { displayModName } from "$lib/mods/names";
  import { layoutTagChips } from "$lib/mods/tagChipLayout";
  import { applyModFilters, type GridStatusFilter } from "$lib/mods/filter";
  import {
    GRID_COLUMNS,
    isColumnVisible,
    normalizeVisibleColumns,
    toggleVisibleColumn,
    visibleColumnCount,
    type GridColumnId,
  } from "$lib/mods/gridColumns";

  interface Props {
    mods: Mod[];
    categories: Category[];
    gridStatusFilter?: GridStatusFilter;
    onClearGridStatusFilter?: () => void;
    selectedModId: string | null;
    searchQuery?: string;
    refreshing?: boolean;
    onselect: (id: string) => void;
    ontoggle: (id: string, enabled: boolean) => void;
    onbulktoggle: (ids: string[], enabled: boolean) => void | Promise<void>;
    onbulkdelete: (mods: Mod[]) => void | Promise<void>;
    ondownloadupdate: (mod: Mod) => Promise<void>;
    oncontext: (mod: Mod, action: string, event?: MouseEvent) => void;
    onqueueinstall: (paths: string[]) => void;
    ontoggletag: (
      modId: string,
      categoryId: string,
      assign: boolean,
    ) => void | Promise<void>;
    visibleColumns?: string[] | null;
    lastUpdateCheck?: number;
    oncolumnschange?: (columns: GridColumnId[]) => void | Promise<void>;
  }

  let {
    mods,
    categories,
    gridStatusFilter = "none",
    onClearGridStatusFilter,
    selectedModId,
    searchQuery = "",
    refreshing = false,
    onselect,
    ontoggle,
    onbulktoggle,
    onbulkdelete,
    ondownloadupdate,
    oncontext,
    onqueueinstall,
    ontoggletag,
    visibleColumns = null,
    lastUpdateCheck = 0,
    oncolumnschange,
  }: Props = $props();

  const ENABLE_COL_WIDTH = 36;
  const ROW_HEIGHT_MOD = 44;
  const ROW_HEIGHT_EMPTY = 200;

  type ResizableColumn =
    | "name"
    | "tags"
    | "author"
    | "version"
    | "folder"
    | "installed"
    | "status";
  type SortColumn =
    | "name"
    | "author"
    | "version"
    | "folder"
    | "installed"
    | "status";
  type SortDirection = "asc" | "desc";

  const SORT_COLUMNS: SortColumn[] = [
    "name",
    "author",
    "version",
    "folder",
    "installed",
    "status",
  ];
  const SORT_COLUMN_LABELS: Record<SortColumn, string> = {
    name: "Name",
    author: "Author",
    version: "Version",
    folder: "Folder",
    installed: "Installed",
    status: "Status",
  };

  const MIN_COL_WIDTHS: Record<ResizableColumn, number> = {
    name: 120,
    tags: 100,
    author: 80,
    version: 72,
    folder: 80,
    installed: 96,
    status: 168,
  };

  const DEFAULT_COL_WIDTHS: Record<ResizableColumn, number> = {
    name: 200,
    tags: 160,
    author: 128,
    version: 112,
    folder: 160,
    installed: 120,
    status: 200,
  };

  function loadColumnWidths(): Record<ResizableColumn, number> {
    try {
      const raw = localStorage.getItem("sdvm-modgrid-columns");
      if (!raw) return { ...DEFAULT_COL_WIDTHS };
      const parsed = JSON.parse(raw) as Partial<
        Record<ResizableColumn, number>
      >;
      const widths = { ...DEFAULT_COL_WIDTHS };
      for (const key of Object.keys(DEFAULT_COL_WIDTHS) as ResizableColumn[]) {
        const w = parsed[key];
        if (typeof w === "number" && w >= MIN_COL_WIDTHS[key]) {
          widths[key] = w;
        }
      }
      return widths;
    } catch {
      return { ...DEFAULT_COL_WIDTHS };
    }
  }

  function loadBulkHintDismissed(): boolean {
    try {
      return localStorage.getItem("sdvm-bulk-hint-dismissed") === "1";
    } catch {
      return false;
    }
  }

  function loadTagsHintDismissed(): boolean {
    try {
      return localStorage.getItem("sdvm-tags-hint-dismissed") === "1";
    } catch {
      return false;
    }
  }

  function loadWorkspaceHintDismissed(): boolean {
    try {
      return localStorage.getItem("sdvm-workspace-hint-dismissed") === "1";
    } catch {
      return false;
    }
  }

  function loadSortPref(): {
    column: SortColumn | null;
    direction: SortDirection | null;
  } {
    try {
      const raw = localStorage.getItem("sdvm-modgrid-sort");
      if (!raw) return { column: "name", direction: "asc" };
      const parsed = JSON.parse(raw) as {
        column?: unknown;
        direction?: unknown;
      };
      if (parsed.column === null && parsed.direction === null) {
        return { column: null, direction: null };
      }
      const column = parsed.column;
      const direction = parsed.direction;
      if (
        typeof column === "string" &&
        SORT_COLUMNS.includes(column as SortColumn) &&
        (direction === "asc" || direction === "desc")
      ) {
        return { column: column as SortColumn, direction };
      }
      return { column: "name", direction: "asc" };
    } catch {
      return { column: "name", direction: "asc" };
    }
  }

  function saveSortPref(
    column: SortColumn | null,
    direction: SortDirection | null,
  ) {
    try {
      localStorage.setItem(
        "sdvm-modgrid-sort",
        JSON.stringify({ column, direction }),
      );
    } catch {
      /* storage unavailable */
    }
  }

  const initialSort = loadSortPref();

  let columnWidths = $state(loadColumnWidths());
  let resizeState = $state<{
    column: ResizableColumn;
    startX: number;
    startWidth: number;
  } | null>(null);

  const isResizing = $derived(resizeState != null);

  let bulkSelected = $state<Set<string>>(new Set());
  let lastClickedId = $state<string | null>(null);
  let downloadingIds = $state<Set<string>>(new Set());
  let bulkActionLoading = $state(false);
  let bulkHintDismissed = $state(loadBulkHintDismissed());
  let tagsHintDismissed = $state(loadTagsHintDismissed());
  let workspaceHintDismissed = $state(loadWorkspaceHintDismissed());
  let focusedModId = $state<string | null>(null);
  let scrollEl = $state<HTMLElement | null>(null);
  let tagPicker = $state<{ mod: Mod; x: number; y: number } | null>(null);
  let tagPickerBusy = $state(false);
  let dropDragOver = $state(false);
  let sortColumn = $state<SortColumn | null>(initialSort.column);
  let sortDirection = $state<SortDirection | null>(initialSort.direction);
  let headerMenu = $state<{ x: number; y: number } | null>(null);
  let expandedBundleIds = $state(loadExpandedBundleIds());

  const activeVisibleColumns = $derived(
    normalizeVisibleColumns(visibleColumns),
  );
  const visibleColCount = $derived(visibleColumnCount(visibleColumns));
  const colVisible = (id: GridColumnId) =>
    isColumnVisible(activeVisibleColumns, id);

  function modName(mod: Mod): string {
    return displayModName(mod);
  }

  function formatInstallDate(ts: number): { label: string; title: string } {
    if (!ts) return { label: "—", title: "" };
    const date = new Date(ts * 1000);
    return {
      label: date.toLocaleDateString(undefined, {
        month: "short",
        day: "numeric",
        year: "numeric",
      }),
      title: date.toLocaleString(),
    };
  }

  function statusSortKey(mod: Mod): number {
    return modStatusSortKey(mod);
  }

  function compareMods(
    a: Mod,
    b: Mod,
    column: SortColumn,
    direction: SortDirection,
  ): number {
    let cmp = 0;
    switch (column) {
      case "name":
        cmp = modName(a).localeCompare(modName(b), undefined, {
          sensitivity: "base",
        });
        break;
      case "author":
        cmp = (a.manifest?.Author ?? "").localeCompare(
          b.manifest?.Author ?? "",
          undefined,
          {
            sensitivity: "base",
          },
        );
        break;
      case "version":
        cmp = bundleVersionLabel(a).localeCompare(
          bundleVersionLabel(b),
          undefined,
          {
            numeric: true,
            sensitivity: "base",
          },
        );
        break;
      case "folder":
        cmp = bundleFolderLabel(a).localeCompare(
          bundleFolderLabel(b),
          undefined,
          {
            sensitivity: "base",
          },
        );
        break;
      case "installed": {
        const aT = a.installTime;
        const bT = b.installTime;
        if (!aT && !bT) cmp = 0;
        else if (!aT) return 1;
        else if (!bT) return -1;
        else cmp = aT - bT;
        break;
      }
      case "status": {
        cmp = statusSortKey(a) - statusSortKey(b);
        if (cmp === 0)
          cmp = modName(a).localeCompare(modName(b), undefined, {
            sensitivity: "base",
          });
        break;
      }
    }
    return direction === "asc" ? cmp : -cmp;
  }

  function sortModList(
    list: Mod[],
    column: SortColumn | null,
    direction: SortDirection | null,
  ): Mod[] {
    if (!column || !direction) return list;
    return [...list].sort((a, b) => compareMods(a, b, column, direction));
  }

  function ariaSortValue(
    column: SortColumn,
  ): "ascending" | "descending" | "none" {
    if (sortColumn !== column || !sortDirection) return "none";
    return sortDirection === "asc" ? "ascending" : "descending";
  }

  function toggleSort(column: SortColumn) {
    if (sortColumn !== column) {
      sortColumn = column;
      sortDirection = "asc";
    } else if (sortDirection === "asc") {
      sortDirection = "desc";
    } else {
      sortColumn = null;
      sortDirection = null;
    }
    saveSortPref(sortColumn, sortDirection);

    if (sortColumn && sortDirection) {
      const dir = sortDirection === "asc" ? "ascending" : "descending";
      sr.announce(`Sorted by ${SORT_COLUMN_LABELS[sortColumn]}, ${dir}`);
    } else {
      sr.announce(gridSortClearedAnnounce);
    }
  }

  function onInstallDragEnter(e: DragEvent) {
    if (useNativeArchiveFileDrop) return;
    if (!e.dataTransfer?.types.includes("Files")) return;
    e.preventDefault();
    dropDragOver = true;
  }

  function onInstallDragOver(e: DragEvent) {
    if (useNativeArchiveFileDrop) return;
    if (!e.dataTransfer?.types.includes("Files")) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = "copy";
    dropDragOver = true;
  }

  function onInstallDragLeave(e: DragEvent) {
    if (useNativeArchiveFileDrop) return;
    const related = e.relatedTarget as Node | null;
    const current = e.currentTarget as HTMLElement;
    if (related && current.contains(related)) return;
    dropDragOver = false;
  }

  function onInstallDrop(e: DragEvent) {
    if (useNativeArchiveFileDrop) return;
    e.preventDefault();
    dropDragOver = false;
    const paths = pathsFromDataTransfer(e.dataTransfer);
    if (paths.length) onqueueinstall(paths);
  }

  const sr = createAnnouncer();

  $effect(() => {
    if (!sortColumn || colVisible(sortColumn)) return;
    sortColumn = null;
    sortDirection = null;
    saveSortPref(null, null);
  });

  function openHeaderMenu(e: MouseEvent) {
    e.preventDefault();
    headerMenu = { x: e.clientX, y: e.clientY };
  }

  function closeHeaderMenu() {
    headerMenu = null;
  }

  async function setColumnVisible(id: GridColumnId, visible: boolean) {
    const next = toggleVisibleColumn(activeVisibleColumns, id, visible);
    if (!oncolumnschange) return;
    await oncolumnschange(next);
    const label = GRID_COLUMNS.find((c) => c.id === id)?.label ?? id;
    sr.announce(visible ? `${label} column shown` : `${label} column hidden`);
  }

  const categoryById = $derived(new Map(categories.map((c) => [c.id, c])));

  const filteredMods = $derived(
    applyModFilters(mods, categories, gridStatusFilter),
  );
  const sortedMods = $derived(
    sortModList(filteredMods, sortColumn, sortDirection),
  );
  const displayRows = $derived(
    buildGridDisplayRows(sortedMods, expandedBundleIds, searchQuery),
  );

  const visibleModIds = $derived(displayRows.map((row) => row.mod.id));

  const activeBulkSelection = $derived(
    new Set(visibleModIds.filter((id) => bulkSelected.has(id))),
  );

  const deletableBulkMods = $derived(
    sortedMods.filter(
      (mod) => activeBulkSelection.has(mod.id) && !mod.isCoreMod,
    ),
  );

  const isEmptyGrid = $derived(sortedMods.length === 0);

  const showBulkHint = $derived(!bulkHintDismissed && sortedMods.length > 0);
  const showWorkspaceHint = $derived(!workspaceHintDismissed);
  const showTagsHint = $derived(!tagsHintDismissed && sortedMods.length > 0);
  const activeLearnHint = $derived.by(
    (): "workspace" | "tags" | "bulk" | null => {
      if (activeBulkSelection.size > 0) return null;
      if (showWorkspaceHint) return "workspace";
      if (showTagsHint) return "tags";
      if (showBulkHint) return "bulk";
      return null;
    },
  );
  const tagFilterCount = $derived(categories.filter((c) => c.visible).length);
  const tagFilterActive = $derived(
    categories.length > 0 &&
      tagFilterCount > 0 &&
      tagFilterCount < categories.length,
  );

  const emptyState = $derived(
    gridStatusFilter === "updates" && mods.length > 0
      ? {
          title: gridUpdatesFilterEmptyTitle(),
          hint: gridUpdatesFilterEmptyHint(),
          tip: undefined as string | undefined,
        }
      : gridStatusFilter === "dependencies" && mods.length > 0
        ? {
            title: gridDependencyFilterEmptyTitle(),
            hint: gridDependencyFilterEmptyHint(),
            tip: undefined as string | undefined,
          }
        : tagFilterActive && mods.length > 0
          ? {
              title: gridTagsFilterEmptyTitle,
              hint: gridTagsFilterEmptyHint,
              tip: undefined as string | undefined,
            }
          : emptyLibraryState(searchQuery),
  );

  $effect(() => {
    if (!tagPicker) return;
    const id = tagPicker.mod.id;
    const fresh = displayRows.find((row) => row.mod.id === id)?.mod;
    if (fresh && fresh !== tagPicker.mod) {
      tagPicker = { ...tagPicker, mod: fresh };
    }
  });

  function closeTagPicker() {
    tagPicker = null;
  }

  function categoriesForMod(mod: Mod): Category[] {
    return (mod.categoryIds ?? [])
      .map((id) => categoryById.get(id))
      .filter((c): c is Category => c != null);
  }

  function openTagPicker(mod: Mod, e: MouseEvent) {
    e.stopPropagation();
    e.preventDefault();
    tagPicker = { mod, x: e.clientX, y: e.clientY };
    dismissTagsHint();
  }

  async function handleTagToggle(categoryId: string, assign: boolean) {
    if (!tagPicker || tagPickerBusy) return;
    tagPickerBusy = true;
    try {
      await ontoggletag(tagPicker.mod.id, categoryId, assign);
    } finally {
      tagPickerBusy = false;
    }
  }

  function statusInfo(mod: Mod, row: GridDisplayRow): {
    text: string;
    badge: string | null;
    title?: string;
  } {
    const info = modStatusInfo(
      row.kind === "child" ? stripUpdateStatus(mod) : mod,
      lastUpdateCheck,
    );
    return { text: info.text, badge: info.badge, title: info.title };
  }

  function stripUpdateStatus(mod: Mod): Mod {
    return { ...mod, updateStatus: {} };
  }

  function versionDisplay(mod: Mod): { text: string; class: string } {
    const current = mod.manifest?.Version ?? "";
    const state = mod.updateStatus?.state;
    if (state === "update" || state === "update_available") {
      const latest = mod.updateStatus?.latestVersion;
      if (latest) {
        return { text: `${current} → ${latest}`, class: "state-update" };
      }
    }
    return { text: current, class: "" };
  }

  function canDownloadUpdate(mod: Mod): boolean {
    const state = mod.updateStatus?.state;
    if (state !== "update" && state !== "update_available") return false;
    return (
      mod.manifest?.UpdateKeys?.some((k) => k.startsWith("Nexus:")) ?? false
    );
  }

  function dismissBulkHint() {
    bulkHintDismissed = true;
    try {
      localStorage.setItem("sdvm-bulk-hint-dismissed", "1");
    } catch {
      /* storage unavailable */
    }
  }

  function dismissTagsHint() {
    tagsHintDismissed = true;
    try {
      localStorage.setItem("sdvm-tags-hint-dismissed", "1");
    } catch {
      /* storage unavailable */
    }
  }

  function dismissWorkspaceHint() {
    workspaceHintDismissed = true;
    try {
      localStorage.setItem("sdvm-workspace-hint-dismissed", "1");
    } catch {
      /* storage unavailable */
    }
  }

  function announceBulkSelection(count: number) {
    if (count === 0) return;
    sr.announce(
      count === 1
        ? "1 mod selected"
        : `${count} mods selected. Use Enable, Disable, Delete, or Clear selection.`,
    );
  }

  function indeterminateCheckbox(
    node: HTMLInputElement,
    indeterminate: boolean,
  ) {
    node.indeterminate = indeterminate;
    return {
      update(value: boolean) {
        node.indeterminate = value;
      },
    };
  }

  function toggleBundleExpanded(mod: Mod, e?: MouseEvent) {
    e?.stopPropagation();
    e?.preventDefault();
    const next = new Set(expandedBundleIds);
    if (next.has(mod.id)) next.delete(mod.id);
    else next.add(mod.id);
    expandedBundleIds = next;
    saveExpandedBundleIds(next);
    sr.announce(
      next.has(mod.id) ? gridBundleExpandLabel : gridBundleCollapseLabel,
    );
  }

  function rowCheckboxChecked(mod: Mod, row: GridDisplayRow): boolean {
    if (row.kind === "child") return mod.enabled;
    if (isBundleMod(mod)) return bundleIsChecked(mod);
    return mod.enabled;
  }

  function rowCheckboxIndeterminate(mod: Mod, row: GridDisplayRow): boolean {
    return (
      row.kind === "parent" && isBundleMod(mod) && bundleIsIndeterminate(mod)
    );
  }

  function rowIsDisabled(mod: Mod, row: GridDisplayRow): boolean {
    if (mod.isCoreMod) return false;
    if (row.kind === "child") return !mod.enabled;
    if (isBundleMod(mod)) return bundleIsFullyDisabled(mod);
    return !mod.enabled;
  }

  function onRowClick(row: GridDisplayRow, e: MouseEvent) {
    const mod = row.mod;
    const id = mod.id;
    focusedModId = id;

    if (e.ctrlKey || e.metaKey) {
      const next = new Set(bulkSelected);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      bulkSelected = next;
      onselect(id);
      lastClickedId = id;
      announceBulkSelection(next.size);
      return;
    }

    if (e.shiftKey && lastClickedId) {
      const ids = visibleModIds;
      const start = ids.indexOf(lastClickedId);
      const end = ids.indexOf(id);
      if (start !== -1 && end !== -1) {
        const [from, to] = start < end ? [start, end] : [end, start];
        const next = new Set([...bulkSelected, ...ids.slice(from, to + 1)]);
        bulkSelected = next;
        onselect(id);
        lastClickedId = id;
        announceBulkSelection(next.size);
        return;
      }
    }

    bulkSelected = new Set();
    onselect(id);
    lastClickedId = id;
  }

  function clearBulkSelection() {
    bulkSelected = new Set();
    sr.announce(gridSelectionClearedAnnounce);
  }

  async function bulkEnable(enabled: boolean) {
    const ids = [...activeBulkSelection];
    if (!ids.length || bulkActionLoading) return;
    bulkActionLoading = true;
    try {
      await onbulktoggle(ids, enabled);
      bulkSelected = new Set();
      const n = ids.length;
      sr.announce(
        `${enabled ? "Enabled" : "Disabled"} ${n === 1 ? "1 mod" : `${n} mods`}`,
      );
    } finally {
      bulkActionLoading = false;
    }
  }

  async function bulkDelete() {
    const targets = deletableBulkMods;
    if (!targets.length || bulkActionLoading) return;
    bulkActionLoading = true;
    try {
      await onbulkdelete(targets);
      bulkSelected = new Set();
      const n = targets.length;
      sr.announce(gridBulkDeleteOpeningAnnounce(n));
    } finally {
      bulkActionLoading = false;
    }
  }

  async function focusModRow(id: string) {
    focusedModId = id;
    if (
      sortedMods.findIndex((m) => m.id === id) === -1 &&
      displayRows.findIndex((row) => row.mod.id === id) === -1
    )
      return;
    await tick();
    scrollEl?.querySelector<HTMLElement>(`tr[data-mod-id="${id}"]`)?.focus();
  }

  async function navigateModRow(
    delta: number,
    jump: "none" | "home" | "end" = "none",
  ) {
    const ids = visibleModIds;
    if (!ids.length) return;

    let idx: number;
    const currentId = focusedModId ?? selectedModId ?? ids[0];
    const currentIdx = ids.indexOf(currentId);

    if (jump === "home") idx = 0;
    else if (jump === "end") idx = ids.length - 1;
    else
      idx = Math.max(
        0,
        Math.min(ids.length - 1, (currentIdx === -1 ? 0 : currentIdx) + delta),
      );

    const nextId = ids[idx];
    if (!nextId) return;
    onselect(nextId);
    lastClickedId = nextId;
    await focusModRow(nextId);
  }

  function onRowKeydown(row: GridDisplayRow, e: KeyboardEvent) {
    const mod = row.mod;
    const target = e.target as HTMLElement;
    if (target.closest("button, input, .resize-handle")) return;

    if (
      row.kind === "parent" &&
      isBundleMod(mod) &&
      (e.key === "ArrowRight" || e.key === "ArrowLeft")
    ) {
      e.preventDefault();
      const expanded = expandedBundleIds.has(mod.id);
      if (e.key === "ArrowRight" && !expanded) toggleBundleExpanded(mod);
      if (e.key === "ArrowLeft" && expanded) toggleBundleExpanded(mod);
      return;
    }

    if (e.key === "Enter") {
      if (e.target !== e.currentTarget) return;
      e.preventDefault();
      bulkSelected = new Set();
      onselect(mod.id);
      lastClickedId = mod.id;
      focusedModId = mod.id;
      return;
    }

    if (e.key === " ") {
      if (e.target !== e.currentTarget) return;
      e.preventDefault();
      if (mod.isCoreMod) return;
      const nextEnabled = !rowCheckboxChecked(mod, row);
      ontoggle(mod.id, nextEnabled);
      sr.announce(`${modName(mod)} ${nextEnabled ? "enabled" : "disabled"}`);
    }
  }

  function onGridKeydown(e: KeyboardEvent) {
    if (e.key === "Escape" && headerMenu) {
      e.preventDefault();
      e.stopPropagation();
      closeHeaderMenu();
      return;
    }
    if (e.key === "Escape" && tagPicker) {
      e.preventDefault();
      e.stopPropagation();
      closeTagPicker();
      return;
    }
    if (e.key === "Escape" && activeBulkSelection.size > 0) {
      e.preventDefault();
      clearBulkSelection();
      return;
    }

    const target = e.target as HTMLElement;
    if (target.closest("button, input, .resize-handle, .th-sort-btn, thead"))
      return;

    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        void navigateModRow(1);
        break;
      case "ArrowUp":
        e.preventDefault();
        void navigateModRow(-1);
        break;
      case "Home":
        e.preventDefault();
        void navigateModRow(0, "home");
        break;
      case "End":
        e.preventDefault();
        void navigateModRow(0, "end");
        break;
    }
  }

  async function downloadUpdate(mod: Mod, e: MouseEvent) {
    e.stopPropagation();
    if (downloadingIds.has(mod.id)) return;
    downloadingIds = new Set([...downloadingIds, mod.id]);
    try {
      await ondownloadupdate(mod);
    } finally {
      const next = new Set(downloadingIds);
      next.delete(mod.id);
      downloadingIds = next;
    }
  }

  function startColumnResize(column: ResizableColumn, e: MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    resizeState = {
      column,
      startX: e.clientX,
      startWidth: columnWidths[column],
    };
  }

  function onColumnResizeMove(e: MouseEvent) {
    if (!resizeState) return;
    const delta = e.clientX - resizeState.startX;
    const width = Math.max(
      MIN_COL_WIDTHS[resizeState.column],
      resizeState.startWidth + delta,
    );
    columnWidths = { ...columnWidths, [resizeState.column]: width };
  }

  function endColumnResize() {
    if (!resizeState) return;
    resizeState = null;
    try {
      localStorage.setItem(
        "sdvm-modgrid-columns",
        JSON.stringify(columnWidths),
      );
    } catch {
      /* storage unavailable */
    }
  }
</script>

<svelte:window onmousemove={onColumnResizeMove} onmouseup={endColumnResize} />

<div
  id={MOD_GRID_DROP_ID}
  data-file-drop-target
  class="mod-grid flex min-h-0 min-w-0 flex-1 flex-col"
  class:mod-grid--drop-target={!useNativeArchiveFileDrop && dropDragOver}
  role="region"
  aria-label="Mod list. Drop mod archives here to install."
  ondragenter={onInstallDragEnter}
  ondragover={onInstallDragOver}
  ondragleave={onInstallDragLeave}
  ondrop={onInstallDrop}
>
  {#if refreshing}
    <div
      class="mod-grid-refresh-indicator refresh-sweep-indicator"
      role="status"
      aria-live="polite"
      aria-label="Refreshing mod list"
    ></div>
  {/if}
  {#if activeBulkSelection.size >= 1}
    <div
      class="mod-grid-chrome-bar mod-grid-chrome-bar--bulk"
      role="status"
      aria-live="polite"
    >
      <span class="bulk-count"
        >{gridBulkSelectedLabel(activeBulkSelection.size)}</span
      >
      <button
        type="button"
        class="btn btn-sm preset-tonal"
        disabled={bulkActionLoading}
        onclick={() => bulkEnable(true)}
      >
        {gridBulkEnableSelected}
      </button>
      <button
        type="button"
        class="btn btn-sm preset-tonal"
        disabled={bulkActionLoading}
        onclick={() => bulkEnable(false)}
      >
        {gridBulkDisableSelected}
      </button>
      <button
        type="button"
        class="btn btn-sm preset-filled-error-500"
        disabled={bulkActionLoading || deletableBulkMods.length === 0}
        onclick={() => bulkDelete()}
      >
        {gridBulkDeleteLabel(
          deletableBulkMods.length || activeBulkSelection.size,
        )}
      </button>
      <button
        type="button"
        class="anchor text-surface-400"
        disabled={bulkActionLoading}
        onclick={clearBulkSelection}
      >
        {gridBulkClearSelection}
      </button>
      {#if activeBulkSelection.size === 1}
        <span class="type-caption type-meta">{gridBulkShiftHint}</span>
      {/if}
    </div>
  {/if}
  {#if tagFilterActive}
    <div class="mod-grid-chrome-bar mod-grid-chrome-bar--status" role="status">
      <span class="state-badge state-badge--info">
        {gridTagsFilteringBadge(tagFilterCount)}
      </span>
      <span class="type-meta">{gridTagsFilteringMeta}</span>
    </div>
  {/if}
  {#if gridStatusFilter === "updates"}
    <div class="mod-grid-chrome-bar mod-grid-chrome-bar--status" role="status">
      <span class="state-badge state-badge--update">
        {gridUpdatesFilterLabel(sortedMods.length)}
      </span>
      <span class="type-meta">{gridUpdatesFilterMeta}</span>
      {#if onClearGridStatusFilter}
        <button
          type="button"
          class="anchor text-surface-400 ml-auto"
          onclick={onClearGridStatusFilter}
        >
          {gridClearFilter}
        </button>
      {/if}
    </div>
  {:else if gridStatusFilter === "dependencies"}
    <div class="mod-grid-chrome-bar mod-grid-chrome-bar--status" role="status">
      <span class="state-badge state-badge--error">
        {gridDependencyFilterLabel(sortedMods.length)}
      </span>
      <span class="type-meta">{gridDependencyFilterMeta}</span>
      {#if onClearGridStatusFilter}
        <button
          type="button"
          class="anchor text-surface-400 ml-auto"
          onclick={onClearGridStatusFilter}
        >
          {gridClearFilter}
        </button>
      {/if}
    </div>
  {/if}
  {#if activeLearnHint === "workspace"}
    <div
      class="mod-grid-chrome-bar mod-grid-chrome-bar--learn mod-grid-chrome-bar--learn-accent"
      role="note"
    >
      <span class="mod-grid-chrome-bar-label">{gridQuickStartLabel}</span>
      <span class="type-meta text-surface-300">{workspaceOnboardingText()}</span
      >
      <button
        type="button"
        class="anchor text-surface-400 ml-auto"
        onclick={dismissWorkspaceHint}>Got it</button
      >
    </div>
  {:else if activeLearnHint === "tags"}
    <div
      class="mod-grid-chrome-bar mod-grid-chrome-bar--learn mod-grid-chrome-bar--learn-accent"
    >
      <span class="mod-grid-chrome-bar-emphasis">{gridTagsLearnEmphasis}</span>
      <span>{gridTagsLearnHint}</span>
      <button
        type="button"
        class="anchor text-surface-400 ml-auto"
        onclick={dismissTagsHint}>{learnHintDismissLabel}</button
      >
    </div>
  {:else if activeLearnHint === "bulk"}
    <div class="mod-grid-chrome-bar mod-grid-chrome-bar--learn">
      <span>{gridBulkKeyboardHint}</span>
      <button
        type="button"
        class="anchor text-surface-400 ml-auto"
        onclick={dismissBulkHint}>{learnHintDismissLabel}</button
      >
    </div>
  {/if}

  <div
    bind:this={scrollEl}
    class="min-h-0 flex-1 overflow-auto"
    role="region"
    aria-label="Mod list scroll area"
  >
    <table
      class="mod-table table w-full type-ui"
      class:is-resizing={isResizing}
      role="grid"
      aria-label="Mod list"
      aria-keyshortcuts="ArrowUp ArrowDown Home End Space Enter Escape"
      onkeydown={onGridKeydown}
    >
      <caption class="sr-only">
        Installed mods. Right-click column headers to show or hide columns.
        Click headers to sort. Arrow keys move between rows. Space toggles
        enabled. Enter opens details.
      </caption>
      <colgroup>
        {#if colVisible("enabled")}
          <col style="width: {ENABLE_COL_WIDTH}px" />
        {/if}
        {#if colVisible("name")}
          <col style="width: {columnWidths.name}px" />
        {/if}
        {#if colVisible("tags")}
          <col style="width: {columnWidths.tags}px" />
        {/if}
        {#if colVisible("author")}
          <col style="width: {columnWidths.author}px" />
        {/if}
        {#if colVisible("version")}
          <col style="width: {columnWidths.version}px" />
        {/if}
        {#if colVisible("folder")}
          <col style="width: {columnWidths.folder}px" />
        {/if}
        {#if colVisible("installed")}
          <col style="width: {columnWidths.installed}px" />
        {/if}
        {#if colVisible("status")}
          <col style="width: {columnWidths.status}px" />
        {/if}
      </colgroup>
      <thead class="mod-grid-thead" oncontextmenu={openHeaderMenu}>
        <tr class="mod-table-head-row">
          {#if colVisible("enabled")}
            <th scope="col">
              <span class="sr-only">Enabled</span>
            </th>
          {/if}
          {#if colVisible("name")}
            <th
              scope="col"
              class="th-resizable th-sortable"
              aria-sort={ariaSortValue("name")}
            >
              <button
                type="button"
                class="th-sort-btn"
                onclick={() => toggleSort("name")}
              >
                <span class="type-label th-label">Name</span>
                {#if sortColumn === "name" && sortDirection === "asc"}
                  <ChevronUp
                    size={12}
                    strokeWidth={2.5}
                    aria-hidden="true"
                    class="th-sort-icon"
                  />
                {:else if sortColumn === "name" && sortDirection === "desc"}
                  <ChevronDown
                    size={12}
                    strokeWidth={2.5}
                    aria-hidden="true"
                    class="th-sort-icon"
                  />
                {/if}
              </button>
              <button
                type="button"
                class="resize-handle"
                aria-label="Resize Name column"
                onmousedown={(e) => startColumnResize("name", e)}
              ></button>
            </th>
          {/if}
          {#if colVisible("tags")}
            <th scope="col" class="th-resizable">
              <span class="type-label th-label">Tags</span>
              <button
                type="button"
                class="resize-handle"
                aria-label="Resize Tags column"
                onmousedown={(e) => startColumnResize("tags", e)}
              ></button>
            </th>
          {/if}
          {#if colVisible("author")}
            <th
              scope="col"
              class="th-resizable th-sortable"
              aria-sort={ariaSortValue("author")}
            >
              <button
                type="button"
                class="th-sort-btn"
                onclick={() => toggleSort("author")}
              >
                <span class="type-label th-label">Author</span>
                {#if sortColumn === "author" && sortDirection === "asc"}
                  <ChevronUp
                    size={12}
                    strokeWidth={2.5}
                    aria-hidden="true"
                    class="th-sort-icon"
                  />
                {:else if sortColumn === "author" && sortDirection === "desc"}
                  <ChevronDown
                    size={12}
                    strokeWidth={2.5}
                    aria-hidden="true"
                    class="th-sort-icon"
                  />
                {/if}
              </button>
              <button
                type="button"
                class="resize-handle"
                aria-label="Resize Author column"
                onmousedown={(e) => startColumnResize("author", e)}
              ></button>
            </th>
          {/if}
          {#if colVisible("version")}
            <th
              scope="col"
              class="th-resizable th-sortable"
              aria-sort={ariaSortValue("version")}
            >
              <button
                type="button"
                class="th-sort-btn"
                onclick={() => toggleSort("version")}
              >
                <span class="type-label th-label">Version</span>
                {#if sortColumn === "version" && sortDirection === "asc"}
                  <ChevronUp
                    size={12}
                    strokeWidth={2.5}
                    aria-hidden="true"
                    class="th-sort-icon"
                  />
                {:else if sortColumn === "version" && sortDirection === "desc"}
                  <ChevronDown
                    size={12}
                    strokeWidth={2.5}
                    aria-hidden="true"
                    class="th-sort-icon"
                  />
                {/if}
              </button>
              <button
                type="button"
                class="resize-handle"
                aria-label="Resize Version column"
                onmousedown={(e) => startColumnResize("version", e)}
              ></button>
            </th>
          {/if}
          {#if colVisible("folder")}
            <th
              scope="col"
              class="th-resizable th-sortable"
              aria-sort={ariaSortValue("folder")}
            >
              <button
                type="button"
                class="th-sort-btn"
                onclick={() => toggleSort("folder")}
              >
                <span class="type-label th-label">Folder</span>
                {#if sortColumn === "folder" && sortDirection === "asc"}
                  <ChevronUp
                    size={12}
                    strokeWidth={2.5}
                    aria-hidden="true"
                    class="th-sort-icon"
                  />
                {:else if sortColumn === "folder" && sortDirection === "desc"}
                  <ChevronDown
                    size={12}
                    strokeWidth={2.5}
                    aria-hidden="true"
                    class="th-sort-icon"
                  />
                {/if}
              </button>
              <button
                type="button"
                class="resize-handle"
                aria-label="Resize Folder column"
                onmousedown={(e) => startColumnResize("folder", e)}
              ></button>
            </th>
          {/if}
          {#if colVisible("installed")}
            <th
              scope="col"
              class="th-resizable th-sortable"
              aria-sort={ariaSortValue("installed")}
            >
              <button
                type="button"
                class="th-sort-btn"
                onclick={() => toggleSort("installed")}
              >
                <span class="type-label th-label">Installed</span>
                {#if sortColumn === "installed" && sortDirection === "asc"}
                  <ChevronUp
                    size={12}
                    strokeWidth={2.5}
                    aria-hidden="true"
                    class="th-sort-icon"
                  />
                {:else if sortColumn === "installed" && sortDirection === "desc"}
                  <ChevronDown
                    size={12}
                    strokeWidth={2.5}
                    aria-hidden="true"
                    class="th-sort-icon"
                  />
                {/if}
              </button>
              <button
                type="button"
                class="resize-handle"
                aria-label="Resize Installed column"
                onmousedown={(e) => startColumnResize("installed", e)}
              ></button>
            </th>
          {/if}
          {#if colVisible("status")}
            <th
              scope="col"
              class="th-resizable th-sortable"
              aria-sort={ariaSortValue("status")}
            >
              <button
                type="button"
                class="th-sort-btn"
                onclick={() => toggleSort("status")}
              >
                <span class="type-label th-label">Status</span>
                {#if sortColumn === "status" && sortDirection === "asc"}
                  <ChevronUp
                    size={12}
                    strokeWidth={2.5}
                    aria-hidden="true"
                    class="th-sort-icon"
                  />
                {:else if sortColumn === "status" && sortDirection === "desc"}
                  <ChevronDown
                    size={12}
                    strokeWidth={2.5}
                    aria-hidden="true"
                    class="th-sort-icon"
                  />
                {/if}
              </button>
              <button
                type="button"
                class="resize-handle"
                aria-label="Resize Status column"
                onmousedown={(e) => startColumnResize("status", e)}
              ></button>
            </th>
          {/if}
        </tr>
      </thead>
      <tbody>
        {#if isEmptyGrid}
          <tr style="height: {ROW_HEIGHT_EMPTY}px">
            <td colspan={visibleColCount} class="empty-state-cell">
              <div class="empty-state-inner">
                {#if !searchQuery.trim()}
                  <div class="empty-state-mark" aria-hidden="true">
                    {@html emptyStateIllustration}
                  </div>
                {/if}
                <p class="empty-state-title">{emptyState.title}</p>
                <p class="empty-state-hint">{emptyState.hint}</p>
                {#if emptyState.tip}
                  <p class="empty-state-tip">{emptyState.tip}</p>
                {/if}
              </div>
            </td>
          </tr>
        {:else}
          {#each displayRows as row (row.mod.id + ":" + row.kind)}
            {@render gridRow(row)}
          {/each}
        {/if}
      </tbody>
    </table>
  </div>
</div>

{#if headerMenu}
  <ModGridHeaderMenu
    x={headerMenu.x}
    y={headerMenu.y}
    {visibleColumns}
    ontoggle={setColumnVisible}
    onclose={closeHeaderMenu}
  />
{/if}

{#snippet gridRow(row: GridDisplayRow)}
  {@const mod = row.mod}
  {@const status = statusInfo(mod, row)}
  {@const versionText =
    row.kind === "child"
      ? (mod.manifest?.Version ?? "")
      : row.kind === "parent" && isBundleMod(mod)
        ? bundleVersionLabel(mod)
        : versionDisplay(mod).text}
  {@const versionClass =
    row.kind === "child"
      ? ""
      : row.kind === "parent" && isBundleMod(mod)
        ? ""
        : versionDisplay(mod).class}
  {@const installedDate = formatInstallDate(mod.installTime)}
  {@const modCategories = categoriesForMod(mod)}
  {@const tagLayout = layoutTagChips(modCategories, columnWidths.tags)}
  {@const isDownloading = downloadingIds.has(mod.id)}
  {@const isSelected = selectedModId === mod.id}
  {@const isBulkSelected = activeBulkSelection.has(mod.id)}
  {@const bundleExpanded = isBundleMod(mod) && expandedBundleIds.has(mod.id)}
  {@const showBundleParts =
    row.kind === "parent" && isBundleMod(mod) && !bundleExpanded}
  <tr
    class="mod-row cursor-pointer hover:bg-surface-800/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/60 focus-visible:ring-inset"
    class:selected={isSelected}
    class:bulk-selected={isBulkSelected && activeBulkSelection.size >= 1}
    class:disabled-row={rowIsDisabled(mod, row)}
    class:mod-row--bundle-parent={row.kind === "parent" && isBundleMod(mod)}
    class:mod-row--bundle-child={row.kind === "child"}
    style="height: {ROW_HEIGHT_MOD}px"
    data-mod-id={mod.id}
    tabindex={focusedModId === mod.id || (focusedModId == null && isSelected)
      ? 0
      : -1}
    aria-selected={isSelected || isBulkSelected}
    onclick={(e) => onRowClick(row, e)}
    onkeydown={(e) => onRowKeydown(row, e)}
    oncontextmenu={(e) => {
      e.preventDefault();
      oncontext(mod, "menu", e);
    }}
  >
    {#if colVisible("enabled")}
      <td class="text-center" role="gridcell">
        <input
          type="checkbox"
          class="checkbox checkbox-sm"
          checked={rowCheckboxChecked(mod, row)}
          use:indeterminateCheckbox={rowCheckboxIndeterminate(mod, row)}
          disabled={mod.isCoreMod}
          aria-label="Enable {modName(mod)}"
          onclick={(e) => e.stopPropagation()}
          onchange={(e) =>
            ontoggle(mod.id, (e.currentTarget as HTMLInputElement).checked)}
        />
      </td>
    {/if}
    {#if colVisible("name")}
      <td class="cell-truncate" role="gridcell">
        <div
          class="mod-name-cell"
          class:mod-name-cell--child={row.kind === "child"}
        >
          {#if row.kind === "parent" && isBundleMod(mod)}
            <button
              type="button"
              class="bundle-toggle"
              aria-expanded={bundleExpanded}
              aria-label={bundleExpanded
                ? gridBundleCollapseLabel
                : gridBundleExpandLabel}
              onclick={(e) => toggleBundleExpanded(mod, e)}
            >
              {#if bundleExpanded}
                <ChevronDown size={14} strokeWidth={2.5} aria-hidden="true" />
              {:else}
                <ChevronRight size={14} strokeWidth={2.5} aria-hidden="true" />
              {/if}
            </button>
          {:else if row.kind === "child"}
            <span class="bundle-child-rail" aria-hidden="true"></span>
          {/if}
          <span class="mod-name min-w-0 truncate text-surface-50"
            >{modName(mod)}</span
          >
          {#if showBundleParts}
            <span class="bundle-parts type-caption type-meta"
              >{gridBundlePartsLabel(bundlePartCount(mod))}</span
            >
          {/if}
          {#if row.kind === "child"}
            <span class="bundle-part-type type-caption type-meta"
              >{bundlePartTypeLabel(mod)}</span
            >
          {/if}
        </div>
      </td>
    {/if}
    {#if colVisible("tags")}
      <td
        class="tags-cell"
        class:tags-cell--empty={modCategories.length === 0}
        class:tags-cell--bundle-child={row.kind === "child"}
        role="gridcell"
      >
        {#if row.kind === "child"}
          <span class="type-caption type-meta" aria-hidden="true">—</span>
        {:else}
          <button
            type="button"
            class="tags-cell-trigger"
            aria-label={tagsCellEditLabel(
              modName(mod),
              modCategories.map((c) => c.name),
            )}
            onclick={(e) => openTagPicker(mod, e)}
          >
            <div class="tags-cell-inner">
              {#if modCategories.length === 0}
                <span class="tags-add-pill type-caption">
                  <span class="tags-add-icon" aria-hidden="true">+</span>
                  {tagsCellAddLabel}
                </span>
              {:else}
                {#each tagLayout.visible as cat (cat.id)}
                  <span
                    class="tag-chip chip-colored type-caption"
                    style:--chip-color={cat.color || "var(--color-primary-500)"}
                    title={cat.name}
                  >
                    {cat.name}
                  </span>
                {/each}
                {#if tagLayout.overflowCount > 0}
                  <span
                    class="tag-chip tag-chip-overflow type-caption"
                    title={tagLayout.overflowLabel}
                    aria-label={tagsOverflowLabel(
                      tagLayout.overflowCount,
                      tagLayout.overflowLabel,
                    )}
                  >
                    +{tagLayout.overflowCount}
                  </span>
                {/if}
              {/if}
            </div>
          </button>
        {/if}
      </td>
    {/if}
    {#if colVisible("author")}
      <td class="cell-truncate type-meta" role="gridcell">
        {mod.manifest?.Author || "—"}
      </td>
    {/if}
    {#if colVisible("version")}
      <td class="cell-truncate type-data {versionClass}" role="gridcell">
        {versionText || "—"}
      </td>
    {/if}
    {#if colVisible("folder")}
      <td class="cell-truncate type-meta" role="gridcell">
        {row.kind === "parent" && isBundleMod(mod)
          ? bundleFolderLabel(mod)
          : mod.folderPath}
      </td>
    {/if}
    {#if colVisible("installed")}
      <td
        class="cell-truncate type-meta"
        role="gridcell"
        title={installedDate.title || undefined}
      >
        {installedDate.label}
      </td>
    {/if}
    {#if colVisible("status")}
      <td role="gridcell">
        <div
          class="status-cell flex min-w-0 items-center gap-2 overflow-hidden"
        >
          <span
            class="min-w-0 truncate {status.badge ?? 'state-muted'}"
            title={status.title}
          >
            {status.text}
          </span>
          {#if mod.containsOverwrites}
            <span
              class="state-badge state-badge--patch type-caption shrink-0"
              title={modContainsOverwritesTooltip()}
            >
              {modContainsOverwritesLabel()}
            </span>
          {/if}
          {#if row.kind === "parent" && canDownloadUpdate(mod)}
            <button
              type="button"
              class="btn btn-sm preset-filled-primary-500 download-btn shrink-0 font-medium"
              disabled={isDownloading}
              aria-busy={isDownloading}
              onclick={(e) => downloadUpdate(mod, e)}
            >
              {isDownloading ? "Downloading…" : "Get update"}
            </button>
          {/if}
        </div>
      </td>
    {/if}
  </tr>
{/snippet}

{#if tagPicker}
  <TagPicker
    mod={tagPicker.mod}
    {categories}
    x={tagPicker.x}
    y={tagPicker.y}
    busy={tagPickerBusy}
    ontoggle={handleTagToggle}
    onclose={closeTagPicker}
  />
{/if}

<style>
  .mod-grid {
    position: relative;
  }

  .mod-grid[data-file-drop-target]:global(.file-drop-target-active)::after,
  .mod-grid--drop-target::after {
    content: "";
    position: absolute;
    inset: 0;
    pointer-events: none;
    border: 1px dashed var(--color-primary-500);
    background-color: color-mix(
      in oklab,
      var(--color-primary-500) 8%,
      transparent
    );
    z-index: 1;
  }

  .mod-table {
    table-layout: fixed;
  }

  .mod-table :global(th),
  .mod-table :global(td) {
    padding: var(--space-2) var(--space-3);
    vertical-align: middle;
  }

  .mod-table :global(td:first-child),
  .mod-table :global(.mod-table-head-row th:first-child) {
    padding-inline: var(--space-2);
    width: var(--space-10);
  }

  .mod-grid-refresh-indicator {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    z-index: calc(var(--z-sticky) + 1);
    pointer-events: none;
  }

  .bulk-count {
    font-size: var(--type-ui);
    font-weight: var(--weight-bold);
    color: var(--color-surface-50);
    letter-spacing: -0.01em;
  }

  .empty-state-cell {
    padding: var(--space-12) var(--space-6);
    text-align: center;
  }

  .empty-state-inner {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-2);
    max-width: 36rem;
    margin-inline: auto;
  }

  .empty-state-mark {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 4rem;
    height: 4rem;
    margin-bottom: var(--space-2);
    color: var(--jh-illustration-fg, var(--color-surface-500));
    border-radius: var(--radius-soft);
    background-color: var(
      --jh-illustration-bg,
      color-mix(in oklab, var(--color-surface-900) 50%, var(--sdvm-panel))
    );
  }

  .empty-state-mark :global(svg) {
    display: block;
    width: 3.5rem;
    height: 3.5rem;
  }

  .empty-state-title {
    font-size: var(--type-body);
    font-weight: var(--weight-semibold);
    color: var(--color-surface-200);
    margin-bottom: var(--space-2);
    text-wrap: balance;
  }

  .empty-state-hint {
    font-size: var(--type-ui);
    color: var(--color-surface-400);
    text-wrap: pretty;
    margin: 0;
  }

  .empty-state-tip {
    margin: var(--space-2) 0 0;
    padding: var(--space-2) var(--space-3);
    font-size: var(--type-caption);
    color: var(--color-surface-300);
    text-wrap: pretty;
    border: 1px solid var(--sdvm-border);
    border-radius: var(--radius-base, 0.25rem);
    background-color: color-mix(
      in oklab,
      var(--color-surface-900) 35%,
      var(--sdvm-panel)
    );
  }

  .mod-name {
    font-weight: var(--weight-semibold);
  }

  .tags-cell {
    max-width: 0;
    padding: 0;
    vertical-align: middle;
  }

  .tags-cell-trigger {
    display: flex;
    align-items: center;
    width: 100%;
    min-width: 0;
    min-height: 100%;
    margin: 0;
    padding: var(--space-1) var(--space-2);
    border: 0;
    border-radius: 0;
    background: transparent;
    color: inherit;
    text-align: left;
    cursor: pointer;
  }

  .tags-cell-trigger:focus-visible {
    outline: 2px solid var(--color-primary-500);
    outline-offset: -2px;
  }

  .tags-cell--empty .tags-cell-trigger:hover,
  .tags-cell--empty .tags-cell-trigger:focus-visible {
    background-color: color-mix(
      in oklab,
      var(--color-primary-500) 6%,
      transparent
    );
  }

  .tags-add-pill {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    max-width: 100%;
    padding: 0.125rem var(--space-2);
    border: 1px solid var(--sdvm-border);
    border-radius: var(--radius-base, 0.25rem);
    color: var(--color-surface-400);
    font-weight: var(--weight-medium);
    background-color: color-mix(in oklab, var(--sdvm-raised) 80%, transparent);
  }

  .tags-cell--empty .tags-cell-trigger:hover .tags-add-pill,
  .tags-cell--empty .tags-cell-trigger:focus-visible .tags-add-pill {
    border-color: color-mix(
      in oklab,
      var(--color-primary-400) 45%,
      var(--sdvm-border)
    );
    color: var(--color-surface-200);
    background-color: color-mix(
      in oklab,
      var(--color-primary-500) 10%,
      var(--sdvm-raised)
    );
  }

  .tags-add-icon {
    flex-shrink: 0;
    font-weight: var(--weight-bold);
    line-height: 1;
    color: var(--color-primary-400);
  }

  .tags-cell-inner {
    display: flex;
    flex-wrap: nowrap;
    align-items: center;
    gap: var(--space-1);
    min-width: 0;
    width: 100%;
    overflow: hidden;
  }

  .tag-chip {
    display: inline-block;
    flex-shrink: 0;
    max-width: 100%;
    padding: 0.125rem var(--space-2);
    border-radius: var(--radius-base, 0.25rem);
    font-size: var(--type-caption);
    font-weight: var(--weight-semibold);
    letter-spacing: 0.01em;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    vertical-align: middle;
  }

  .tag-chip-overflow {
    flex-shrink: 0;
    color: var(--color-surface-300);
    background-color: color-mix(
      in oklab,
      var(--color-surface-800) 75%,
      var(--sdvm-panel)
    );
    border: 1px solid var(--sdvm-border);
  }

  .mod-row.disabled-row .mod-name {
    color: var(--color-surface-400);
    font-weight: var(--weight-medium);
  }

  .mod-table.is-resizing {
    cursor: col-resize;
    user-select: none;
  }

  .cell-truncate {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 0;
  }

  tr.selected {
    background-color: var(--sdvm-selection);
  }

  tr.mod-row {
    transition: background-color var(--motion-fast) var(--ease-out-quart);
  }

  tr.bulk-selected {
    background-color: var(--sdvm-selection-bulk);
  }

  tr.selected.bulk-selected {
    background-color: var(--sdvm-selection-combined);
  }

  .status-cell {
    justify-content: space-between;
  }

  .download-btn {
    min-height: 1.75rem;
    padding: 0 0.5rem;
    font-size: var(--type-caption);
    line-height: var(--leading-snug);
  }

  .mod-row--bundle-parent {
    background-color: color-mix(in oklab, var(--sdvm-raised) 55%, transparent);
  }

  .mod-row--bundle-child {
    background-color: color-mix(in oklab, var(--sdvm-panel) 88%, transparent);
  }

  .mod-name-cell {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    min-width: 0;
  }

  .mod-name-cell--child {
    padding-left: var(--space-1);
  }

  .bundle-toggle {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    width: 1.25rem;
    height: 1.25rem;
    margin: 0;
    padding: 0;
    border: 0;
    border-radius: var(--radius-base, 0.25rem);
    background: transparent;
    color: var(--color-surface-300);
    cursor: pointer;
  }

  .bundle-toggle:hover,
  .bundle-toggle:focus-visible {
    color: var(--color-surface-50);
    background-color: color-mix(
      in oklab,
      var(--color-primary-500) 10%,
      transparent
    );
  }

  .bundle-child-rail {
    flex-shrink: 0;
    width: 1.25rem;
    height: 1.25rem;
    border-left: 1px solid var(--sdvm-border);
    margin-left: 0.35rem;
  }

  .bundle-parts,
  .bundle-part-type {
    flex-shrink: 0;
    white-space: nowrap;
  }

  .tags-cell--bundle-child {
    opacity: 0.55;
  }

  @media (prefers-reduced-motion: reduce) {
    tr.mod-row {
      transition: none;
    }
  }
</style>
