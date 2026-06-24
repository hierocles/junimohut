<script lang="ts">
  import { ChevronDown, MoreHorizontal, X } from '@lucide/svelte';
  import { createAnnouncer } from '@sv-kit/a11y-keys';
  import type { Mod } from '$lib/api/client';
  import type { DownloadEntry } from '../../../bindings/junimohut/internal/nexus/models.js';
  import * as m from '$lib/paraglide/messages.js';
  import {
    downloadsBulkInstallLabel,
    downloadsBulkReinstallLabel,
    downloadsModLibraryLine,
    downloadsNexusIdLine,
    downloadsProgressAria,
    downloadsRowMoreAriaFor,
    downloadsSearchEmpty,
    downloadsSelectionCount,
    downloadsUniqueIdLine,
  } from '$lib/i18n/helpers';
  import {
    archiveSearchText,
    formatDownloadTimestamp,
    resolveArchiveMod,
    type SavedDownloadRecord,
  } from '$lib/mods/savedDownloads';

  const MIN_WIDTH = 280;
  const MAX_WIDTH = 480;
  const SKELETON_ROWS = 4;

  type HistoryRow = {
    record: SavedDownloadRecord;
    displayName: string;
    mod: Mod | null;
    unlinked: boolean;
  };

  type RowMenuItem = {
    action: 'folder' | 'nexus' | 'delete';
    label: string;
    danger?: boolean;
  };

  type MenuPos = {
    left: number;
    top: number;
    origin: string;
  };

  interface Props {
    activeDownloads: DownloadEntry[];
    savedDownloads: SavedDownloadRecord[];
    mods: Mod[];
    loading?: boolean;
    refreshing?: boolean;
    fetchError?: string;
    paneWidth: number;
    onclose: () => void;
    onretry?: () => void;
    onwidthchange: (width: number) => void;
    oninstall: (archivePath: string) => void;
    onreinstall: (mod: Mod, archivePath: string) => void;
    onshowfolder: (archivePath: string) => void;
    ondelete: (record: SavedDownloadRecord, displayName: string) => void;
    onopennexus: (nexusModId: number) => void;
    onviewmod?: (modId: string) => void;
    onbulkinstall?: (paths: string[]) => void;
    onbulkreinstall?: (items: { mod: Mod; archivePath: string }[]) => void;
  }

  let {
    activeDownloads,
    savedDownloads,
    mods,
    loading = false,
    refreshing = false,
    fetchError = '',
    paneWidth,
    onclose,
    onretry,
    onwidthchange,
    oninstall,
    onreinstall,
    onshowfolder,
    ondelete,
    onopennexus,
    onviewmod,
    onbulkinstall,
    onbulkreinstall,
  }: Props = $props();

  const sr = createAnnouncer();

  function loadBulkHintDismissed(): boolean {
    try {
      return localStorage.getItem('sdvm-downloads-bulk-hint-dismissed') === '1';
    } catch {
      return false;
    }
  }

  let searchQuery = $state('');
  let expandedRowPath = $state<string | null>(null);
  let focusedRowPath = $state<string | null>(null);
  let bulkSelected = $state<Set<string>>(new Set());
  let lastClickedPath = $state<string | null>(null);
  let bulkActionLoading = $state(false);
  let bulkHintDismissed = $state(loadBulkHintDismissed());
  let paneResize = $state<{ startX: number; startWidth: number } | null>(null);
  let openMenuRow = $state<HistoryRow | null>(null);
  let menuPos = $state<MenuPos | null>(null);
  let menuEl = $state<HTMLDivElement | undefined>();
  let menuTriggerEl = $state<HTMLButtonElement | null>(null);

  const MENU_WIDTH = 192;
  const MENU_ITEM_H = 32;
  const MENU_CHROME_H = 40;

  const activeItems = $derived(
    activeDownloads.filter((entry) => !entry.status.toLowerCase().includes('complete')),
  );

  const historyRows = $derived.by((): HistoryRow[] => {
    const query = searchQuery.trim().toLowerCase();
    return savedDownloads
      .map((record) => {
        const resolved = resolveArchiveMod(record, mods);
        return {
          record,
          displayName: resolved.displayName,
          mod: resolved.mod,
          unlinked: resolved.mod === null && !(record.nexusModId ?? 0),
        };
      })
      .filter((row) => {
        if (!query) return true;
        return archiveSearchText(row.record, row.displayName).includes(query);
      });
  });

  const visibleRowPaths = $derived(historyRows.map((row) => row.record.archivePath));

  const activeBulkSelection = $derived(
    new Set(visibleRowPaths.filter((path) => bulkSelected.has(path))),
  );

  const bulkInstallPaths = $derived(
    historyRows
      .filter((row) => activeBulkSelection.has(row.record.archivePath) && !row.mod)
      .map((row) => row.record.archivePath),
  );

  const bulkReinstallItems = $derived(
    historyRows
      .filter((row) => activeBulkSelection.has(row.record.archivePath) && row.mod)
      .map((row) => ({ mod: row.mod!, archivePath: row.record.archivePath })),
  );

  const showBulkHint = $derived(
    !bulkHintDismissed && historyRows.length > 0 && activeBulkSelection.size === 0,
  );

  const searchDisabled = $derived(
    loading || (Boolean(fetchError) && savedDownloads.length === 0),
  );

  const rowMenuItems = $derived.by((): RowMenuItem[] => {
    const row = openMenuRow;
    if (!row) return [];
    const items: RowMenuItem[] = [];
    items.push({ action: 'folder', label: m.downloads_action_show_folder() });
    if ((row.record.nexusModId ?? 0) > 0) {
      items.push({ action: 'nexus', label: m.downloads_action_open_nexus() });
    }
    items.push({ action: 'delete', label: m.downloads_action_delete(), danger: true });
    return items;
  });

  function statusBadge(status: string): string {
    const s = status.toLowerCase();
    if (s.includes('fail') || s.includes('error')) return 'state-badge state-badge--error';
    if (s.includes('complete') || s.includes('done') || s.includes('success')) {
      return 'state-badge state-badge--success';
    }
    if (s.includes('download') || s.includes('progress') || s.includes('pending')) {
      return 'state-badge state-badge--update';
    }
    return 'state-badge state-badge--info';
  }

  function startPaneResize(e: MouseEvent) {
    e.preventDefault();
    paneResize = { startX: e.clientX, startWidth: paneWidth };
  }

  function onPaneResizeMove(e: MouseEvent) {
    if (!paneResize) return;
    const delta = paneResize.startX - e.clientX;
    const next = Math.min(
      MAX_WIDTH,
      Math.max(MIN_WIDTH, paneResize.startWidth + delta),
    );
    onwidthchange(next);
  }

  function endPaneResize() {
    paneResize = null;
  }

  function closeRowMenu() {
    const trigger = menuTriggerEl;
    openMenuRow = null;
    menuPos = null;
    menuTriggerEl = null;
    queueMicrotask(() => trigger?.focus());
  }

  function rowMenuItemCount(row: HistoryRow): number {
    let count = 2; // folder + delete
    if ((row.record.nexusModId ?? 0) > 0) count += 1;
    return count;
  }

  function toggleRowDetails(archivePath: string) {
    expandedRowPath = expandedRowPath === archivePath ? null : archivePath;
  }

  function rowDetailId(archivePath: string): string {
    return `downloads-row-detail-${archivePath.replace(/[^a-zA-Z0-9_-]+/g, '-')}`;
  }

  function dismissBulkHint() {
    bulkHintDismissed = true;
    try {
      localStorage.setItem('sdvm-downloads-bulk-hint-dismissed', '1');
    } catch {
      /* storage unavailable */
    }
  }

  function announceBulkSelection(count: number) {
    if (count === 0) return;
    sr.announce(
      count === 1
        ? '1 archive selected'
        : `${count} archives selected. Use Install or Reinstall selected.`,
    );
  }

  function clearBulkSelection() {
    bulkSelected = new Set();
    sr.announce('Selection cleared');
  }

  function onHistoryIdentClick(e: MouseEvent, row: HistoryRow) {
    const path = row.record.archivePath;
    focusedRowPath = path;

    if (e.ctrlKey || e.metaKey) {
      e.preventDefault();
      const next = new Set(bulkSelected);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      bulkSelected = next;
      lastClickedPath = path;
      announceBulkSelection(next.size);
      return;
    }

    if (e.shiftKey && lastClickedPath) {
      const paths = visibleRowPaths;
      const start = paths.indexOf(lastClickedPath);
      const end = paths.indexOf(path);
      if (start !== -1 && end !== -1) {
        const [from, to] = start < end ? [start, end] : [end, start];
        const next = new Set([...bulkSelected, ...paths.slice(from, to + 1)]);
        bulkSelected = next;
        lastClickedPath = path;
        announceBulkSelection(next.size);
      }
    }

    lastClickedPath = path;
  }

  function runPrimaryRowAction(row: HistoryRow) {
    if (row.mod) {
      onreinstall(row.mod, row.record.archivePath);
    } else {
      oninstall(row.record.archivePath);
    }
  }

  function handleHistoryRowKeydown(e: KeyboardEvent, row: HistoryRow) {
    if (e.key !== 'Enter' || e.defaultPrevented) return;
    if ((e.target as HTMLElement).closest('button, input, [role="menuitem"]')) return;
    e.preventDefault();
    runPrimaryRowAction(row);
  }

  $effect(() => {
    const visible = new Set(visibleRowPaths);
    let changed = false;
    const next = new Set<string>();
    for (const path of bulkSelected) {
      if (visible.has(path)) next.add(path);
      else changed = true;
    }
    if (next.size !== bulkSelected.size) changed = true;
    if (changed) bulkSelected = next;
  });

  async function runBulkInstall() {
    const paths = bulkInstallPaths;
    if (!paths.length || bulkActionLoading || !onbulkinstall) return;
    bulkActionLoading = true;
    try {
      onbulkinstall(paths);
      bulkSelected = new Set();
      sr.announce(
        paths.length === 1
          ? 'Opening install for 1 archive'
          : `Opening install for ${paths.length} archives`,
      );
    } finally {
      bulkActionLoading = false;
    }
  }

  async function runBulkReinstall() {
    const items = bulkReinstallItems;
    if (!items.length || bulkActionLoading || !onbulkreinstall) return;
    bulkActionLoading = true;
    try {
      onbulkreinstall(items);
      bulkSelected = new Set();
    } finally {
      bulkActionLoading = false;
    }
  }

  function openRowMenu(e: MouseEvent, row: HistoryRow) {
    if (openMenuRow?.record.archivePath === row.record.archivePath) {
      closeRowMenu();
      return;
    }
    const btn = e.currentTarget as HTMLButtonElement;
    menuTriggerEl = btn;
    const rect = btn.getBoundingClientRect();
    const menuHeight = rowMenuItemCount(row) * MENU_ITEM_H + MENU_CHROME_H;
    let top = rect.bottom + 4;
    let origin = 'top right';
    if (top + menuHeight > window.innerHeight - 8) {
      top = Math.max(8, rect.top - menuHeight - 4);
      origin = 'bottom right';
    }
    const left = Math.min(
      Math.max(8, rect.right - MENU_WIDTH),
      window.innerWidth - MENU_WIDTH - 8,
    );
    openMenuRow = row;
    menuPos = { left, top, origin };
  }

  function handleRowMenuAction(action: RowMenuItem['action']) {
    const row = openMenuRow;
    if (!row) return;
    closeRowMenu();
    switch (action) {
      case 'folder':
        onshowfolder(row.record.archivePath);
        break;
      case 'nexus':
        if ((row.record.nexusModId ?? 0) > 0) onopennexus(row.record.nexusModId!);
        break;
      case 'delete':
        ondelete(row.record, row.displayName);
        break;
    }
  }

  function focusMenuItem(delta: number) {
    const buttons = menuEl?.querySelectorAll<HTMLButtonElement>('button.overlay-menu-item');
    if (!buttons?.length) return;
    const current = document.activeElement;
    let idx = [...buttons].indexOf(current as HTMLButtonElement);
    if (idx < 0) idx = 0;
    else if (delta > 0) idx = Math.min(idx + 1, buttons.length - 1);
    else idx = Math.max(idx - 1, 0);
    buttons[idx]?.focus();
  }

  $effect(() => {
    if (!openMenuRow) return;
    queueMicrotask(() => {
      const first = menuEl?.querySelector<HTMLButtonElement>('button.overlay-menu-item');
      (first ?? menuEl)?.focus();
    });
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        e.stopPropagation();
        closeRowMenu();
        return;
      }
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        focusMenuItem(1);
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        focusMenuItem(-1);
      }
    };
    window.addEventListener('keydown', onKey, true);
    return () => window.removeEventListener('keydown', onKey, true);
  });

  $effect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return;
      if (openMenuRow) return;
      if (bulkSelected.size > 0) {
        e.preventDefault();
        clearBulkSelection();
        return;
      }
      if (document.querySelector('dialog[open]')) return;
      e.preventDefault();
      onclose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  });
</script>

<svelte:window onmousemove={onPaneResizeMove} onmouseup={endPaneResize} />

<aside
  id="downloads-pane"
  class="downloads-pane app-panel"
  class:is-resizing={paneResize != null}
  aria-label={m.downloads_pane_title()}
  aria-busy={loading}
>
  {#if refreshing && !loading}
    <div
      class="downloads-refresh-indicator refresh-sweep-indicator"
      role="status"
      aria-live="polite"
      aria-label={m.downloads_refreshing_aria()}
    ></div>
  {/if}
  <button
    type="button"
    class="downloads-pane-resize-handle"
    aria-label={m.downloads_resize_aria()}
    onmousedown={startPaneResize}
  ></button>

  <header class="downloads-pane-header">
    <h2 class="type-section-head downloads-pane-title">{m.downloads_pane_title()}</h2>
    <button
      type="button"
      class="btn btn-sm preset-tonal toolbar-icon-btn"
      onclick={onclose}
      aria-label={m.downloads_pane_close_aria()}
    >
      <X size={14} aria-hidden="true" />
    </button>
  </header>

  {#if loading}
    <section class="downloads-pane-section" aria-labelledby="downloads-active-heading">
      <h3 id="downloads-active-heading" class="downloads-pane-section-label type-label">
        {m.downloads_active_section_label()}
      </h3>
      <ul class="downloads-skeleton-list" aria-hidden="true">
        {#each Array(2) as _, i (i)}
          <li class="downloads-skeleton-row">
            <span class="downloads-skeleton-line downloads-skeleton-line--title"></span>
            <span class="downloads-skeleton-line downloads-skeleton-line--bar"></span>
          </li>
        {/each}
      </ul>
      <p class="sr-only" role="status">{m.downloads_loading_aria()}</p>
    </section>

    <section class="downloads-pane-section downloads-pane-section--history" aria-labelledby="downloads-history-heading">
      <div class="downloads-history-head">
        <h3 id="downloads-history-heading" class="downloads-pane-section-label type-label">
          {m.downloads_history_section_label()}
        </h3>
        <div class="downloads-skeleton-line downloads-skeleton-line--search" aria-hidden="true"></div>
      </div>
      <ul class="downloads-skeleton-list downloads-skeleton-list--history" aria-hidden="true">
        {#each Array(SKELETON_ROWS) as _, i (i)}
          <li class="downloads-skeleton-row downloads-skeleton-row--history">
            <div class="downloads-skeleton-ident">
              <span class="downloads-skeleton-line downloads-skeleton-line--title"></span>
              <span class="downloads-skeleton-line downloads-skeleton-line--meta"></span>
              <span class="downloads-skeleton-line downloads-skeleton-line--date"></span>
            </div>
            <div class="downloads-skeleton-actions">
              <span class="downloads-skeleton-line downloads-skeleton-line--btn"></span>
              <span class="downloads-skeleton-line downloads-skeleton-line--icon"></span>
            </div>
          </li>
        {/each}
      </ul>
    </section>
  {:else}
  <section class="downloads-pane-section" aria-labelledby="downloads-active-heading">
    <h3 id="downloads-active-heading" class="downloads-pane-section-label type-label">
      {m.downloads_active_section_label()}
    </h3>
    {#if activeItems.length > 0}
      <ul class="downloads-active-list">
        {#each activeItems as entry (entry.id)}
          <li class="downloads-active-item">
            <div class="downloads-active-row">
              <span class="downloads-active-name truncate">{entry.modName}</span>
              <span class="{statusBadge(entry.status)} type-caption shrink-0">{entry.status}</span>
            </div>
            {#if entry.progress > 0 && entry.progress < 100}
              <div
                class="downloads-progress-track"
                role="progressbar"
                aria-valuenow={entry.progress}
                aria-valuemin={0}
                aria-valuemax={100}
                aria-label={downloadsProgressAria(entry.modName)}
              >
                <div
                  class="downloads-progress-bar"
                  style="transform: scaleX({entry.progress / 100})"
                ></div>
              </div>
            {/if}
          </li>
        {/each}
      </ul>
    {:else}
      <p class="downloads-pane-empty type-ui type-meta type-prose">{m.downloads_active_empty()}</p>
    {/if}
  </section>

  <section class="downloads-pane-section downloads-pane-section--history" aria-labelledby="downloads-history-heading">
    <div class="downloads-history-head">
      <h3 id="downloads-history-heading" class="downloads-pane-section-label type-label">
        {m.downloads_history_section_label()}
      </h3>
      <input
        class="input input-sm downloads-search"
        type="search"
        bind:value={searchQuery}
        placeholder={m.downloads_search_placeholder()}
        aria-label={m.downloads_search_placeholder()}
        disabled={searchDisabled}
      />
    </div>

    {#if fetchError}
      <div class="downloads-fetch-error" role="alert">
        <p class="downloads-fetch-error-text type-ui type-meta type-prose">{fetchError || m.downloads_load_error()}</p>
        {#if onretry}
          <button type="button" class="btn btn-sm preset-tonal" onclick={onretry}>
            {m.downloads_load_retry()}
          </button>
        {/if}
      </div>
    {/if}

    {#if savedDownloads.length === 0 && !fetchError}
      <p class="downloads-pane-empty type-ui type-meta type-prose">{m.downloads_empty_history()}</p>
    {:else if savedDownloads.length > 0 && historyRows.length === 0}
      <p class="downloads-pane-empty type-ui type-meta type-prose">
        {downloadsSearchEmpty(searchQuery.trim())}
      </p>
    {:else}
      {#if activeBulkSelection.size >= 1}
        <div
          class="mod-grid-chrome-bar mod-grid-chrome-bar--bulk downloads-bulk-bar"
          role="status"
          aria-live="polite"
          aria-busy={bulkActionLoading}
        >
          <span class="bulk-count">{downloadsSelectionCount(activeBulkSelection.size)}</span>
          {#if bulkInstallPaths.length > 0 && onbulkinstall}
            <button
              type="button"
              class="btn btn-sm preset-filled-primary-500 font-medium"
              disabled={bulkActionLoading}
              onclick={() => void runBulkInstall()}
            >
              {downloadsBulkInstallLabel(bulkInstallPaths.length)}
            </button>
          {/if}
          {#if bulkReinstallItems.length > 0 && onbulkreinstall}
            <button
              type="button"
              class="btn btn-sm preset-tonal"
              disabled={bulkActionLoading}
              onclick={() => void runBulkReinstall()}
            >
              {downloadsBulkReinstallLabel(bulkReinstallItems.length)}
            </button>
          {/if}
          <button
            type="button"
            class="anchor text-surface-400 downloads-bulk-clear"
            disabled={bulkActionLoading}
            onclick={clearBulkSelection}
          >
            {m.downloads_bulk_clear_selection()}
          </button>
          {#if activeBulkSelection.size === 1}
            <span class="downloads-bulk-range-hint type-caption type-meta">
              {m.downloads_bulk_range_hint()}
            </span>
          {/if}
        </div>
      {:else if showBulkHint}
        <div class="mod-grid-chrome-bar mod-grid-chrome-bar--learn downloads-bulk-bar">
          <span>{m.downloads_bulk_learn_hint()}</span>
          <button type="button" class="anchor text-surface-400 ml-auto" onclick={dismissBulkHint}>
            {m.learn_hint_dismiss_label()}
          </button>
        </div>
      {/if}
      <ul
        class="downloads-history-list type-ui"
        role="listbox"
        aria-labelledby="downloads-history-heading"
        aria-multiselectable="true"
        aria-keyshortcuts="Enter Escape"
      >
        {#each historyRows as row, rowIndex (row.record.archivePath)}
          {@const when = formatDownloadTimestamp(row.record.downloadedAt)}
          {@const rowId = `downloads-history-row-${row.record.archivePath.replace(/[^a-zA-Z0-9_-]+/g, '-')}`}
          {@const detailId = rowDetailId(row.record.archivePath)}
          {@const isBulkSelected = activeBulkSelection.has(row.record.archivePath)}
          {@const libraryLine = row.mod ? downloadsModLibraryLine(row.mod) : null}
          <li
            id={rowId}
            role="option"
            aria-selected={isBulkSelected}
            class="downloads-history-row"
            class:downloads-history-row--menu-open={openMenuRow?.record.archivePath === row.record.archivePath}
            class:downloads-history-row--expanded={expandedRowPath === row.record.archivePath}
            class:downloads-history-row--bulk-selected={isBulkSelected && activeBulkSelection.size >= 1}
            tabindex={focusedRowPath === row.record.archivePath ||
            (focusedRowPath == null && rowIndex === 0)
              ? 0
              : -1}
            onkeydown={(e) => handleHistoryRowKeydown(e, row)}
            onfocus={() => (focusedRowPath = row.record.archivePath)}
          >
            <div class="downloads-history-body">
              <button
                type="button"
                class="downloads-row-expand btn btn-sm preset-tonal toolbar-icon-btn"
                class:downloads-row-expand--open={expandedRowPath === row.record.archivePath}
                aria-label={expandedRowPath === row.record.archivePath
                  ? m.downloads_row_details_hide()
                  : m.downloads_row_details_show()}
                aria-expanded={expandedRowPath === row.record.archivePath}
                aria-controls={detailId}
                onclick={() => toggleRowDetails(row.record.archivePath)}
              >
                <ChevronDown size={14} aria-hidden="true" />
              </button>
              <button
                type="button"
                class="downloads-history-ident"
                onclick={(e) => onHistoryIdentClick(e, row)}
              >
                <div class="downloads-history-mod-line">
                  <span class="downloads-mod-name" title={row.displayName}>{row.displayName}</span>
                  {#if row.unlinked}
                    <span
                      class="state-badge state-badge--info type-caption shrink-0"
                      title="{m.downloads_unlinked_title()}. {m.downloads_unlinked_hint()}"
                    >{m.downloads_unlinked_badge()}</span>
                  {/if}
                </div>
                <span
                  class="downloads-archive-name type-mono"
                  title={row.record.fileName ?? row.record.archivePath}
                >
                  {row.record.fileName ?? row.record.archivePath}
                </span>
                <span
                  class="downloads-date type-data tabular-nums type-meta"
                  title={when.title}
                  aria-label={when.title || undefined}
                >
                  {when.label}
                </span>
              </button>
              <div class="downloads-row-actions">
                {#if row.mod}
                  <button
                    type="button"
                    class="btn btn-sm preset-filled-primary-500 downloads-primary-btn font-medium"
                    onclick={() => onreinstall(row.mod!, row.record.archivePath)}
                  >
                    {m.downloads_action_reinstall()}
                  </button>
                {:else}
                  <button
                    type="button"
                    class="btn btn-sm preset-filled-primary-500 downloads-primary-btn font-medium"
                    onclick={() => oninstall(row.record.archivePath)}
                  >
                    {m.downloads_action_install()}
                  </button>
                {/if}
                <button
                  type="button"
                  class="btn btn-sm preset-tonal toolbar-icon-btn downloads-more-btn"
                  aria-label={downloadsRowMoreAriaFor(row.displayName)}
                  aria-haspopup="menu"
                  aria-expanded={openMenuRow?.record.archivePath === row.record.archivePath}
                  onclick={(e) => openRowMenu(e, row)}
                >
                  <MoreHorizontal size={14} aria-hidden="true" />
                </button>
              </div>
            </div>
            {#if expandedRowPath === row.record.archivePath}
              <div id={detailId} class="downloads-row-detail type-caption type-meta">
                {#if row.unlinked}
                  <p class="downloads-row-detail-hint type-prose">{m.downloads_unlinked_hint()}</p>
                {/if}
                {#if libraryLine}
                  <p class="downloads-row-detail-line">{libraryLine}</p>
                {/if}
                {#if row.record.uniqueId}
                  <p class="downloads-row-detail-line">{downloadsUniqueIdLine(row.record.uniqueId)}</p>
                {/if}
                {#if (row.record.nexusModId ?? 0) > 0}
                  <p class="downloads-row-detail-line">{downloadsNexusIdLine(row.record.nexusModId!)}</p>
                {/if}
                <p class="downloads-row-detail-line truncate" title={row.record.archivePath}>
                  {row.record.archivePath}
                </p>
                {#if row.mod && onviewmod}
                  <button
                    type="button"
                    class="btn btn-sm preset-tonal downloads-view-mod-btn"
                    onclick={() => onviewmod(row.mod!.id)}
                  >
                    {m.downloads_view_in_library()}
                  </button>
                {/if}
              </div>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
  </section>
  {/if}
</aside>

{#if openMenuRow && menuPos}
  <div class="overlay-scrim overlay-scrim--menu" role="presentation" onclick={closeRowMenu}>
    <div
      bind:this={menuEl}
      class="overlay-menu-panel downloads-row-menu"
      style:left="{menuPos.left}px"
      style:top="{menuPos.top}px"
      style:--motion-origin={menuPos.origin}
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.stopPropagation()}
      role="menu"
      aria-label={downloadsRowMoreAriaFor(openMenuRow.displayName)}
      tabindex="-1"
    >
      <p class="overlay-menu-title type-caption type-meta truncate" title={openMenuRow.displayName}>
        {openMenuRow.displayName}
      </p>
      {#each rowMenuItems as item (item.action)}
        <button
          type="button"
          class="overlay-menu-item truncate"
          class:overlay-menu-item--danger={item.danger}
          role="menuitem"
          onclick={() => handleRowMenuAction(item.action)}
        >
          {item.label}
        </button>
      {/each}
    </div>
  </div>
{/if}

<style>
  .downloads-pane {
    position: relative;
    display: flex;
    flex-direction: column;
    width: 100%;
    min-width: 0;
    min-height: 0;
    height: 100%;
    overflow: hidden;
    border-left: 1px solid var(--sdvm-border);
    background-color: var(--sdvm-panel);
    container-type: inline-size;
    container-name: downloads-pane;
  }

  .downloads-pane-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
    flex-shrink: 0;
    padding: var(--space-3) var(--space-4);
    border-bottom: 1px solid var(--sdvm-border);
  }

  .downloads-pane-title {
    margin: 0;
    text-wrap: balance;
  }

  .downloads-pane-section {
    flex-shrink: 0;
    padding: var(--space-3) var(--space-4);
    border-bottom: 1px solid var(--sdvm-divider);
  }

  .downloads-pane-section--history {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
    padding-bottom: 0;
    border-bottom: none;
  }

  .downloads-pane-section-label {
    margin: 0;
    color: var(--color-surface-400);
  }

  .downloads-history-head {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    margin-bottom: var(--space-3);
  }

  .downloads-search {
    width: 100%;
  }

  .downloads-active-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    margin: var(--space-2) 0 0;
    padding: 0;
    list-style: none;
  }

  .downloads-active-item {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .downloads-active-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
  }

  .downloads-active-name {
    min-width: 0;
    font-weight: 600;
  }

  .downloads-progress-track {
    height: 0.25rem;
    border-radius: 9999px;
    background-color: var(--color-surface-700);
    overflow: hidden;
  }

  .downloads-progress-bar {
    width: 100%;
    height: 100%;
    border-radius: 9999px;
    background-color: var(--color-primary-400);
    transform-origin: left center;
    transition: transform var(--motion-medium) var(--ease-out-quart);
  }

  .downloads-history-list {
    flex: 1;
    min-height: 0;
    overflow: auto;
    margin: 0 calc(-1 * var(--space-4));
    padding: 0 var(--space-4) var(--space-4);
    list-style: none;
  }

  .downloads-history-row {
    padding: var(--space-2) var(--space-1);
    margin-inline: calc(-1 * var(--space-1));
    border-bottom: 1px solid color-mix(in oklab, var(--sdvm-border) 70%, transparent);
    border-radius: var(--radius-base, 0.25rem);
    transition: background-color var(--motion-fast) var(--ease-out-quart);
  }

  .downloads-history-row:hover {
    background-color: color-mix(in oklab, var(--sdvm-raised) 55%, transparent);
  }

  .downloads-history-row:focus-within {
    background-color: color-mix(in oklab, var(--sdvm-selection) 45%, transparent);
  }

  .downloads-history-row--menu-open {
    background-color: color-mix(in oklab, var(--sdvm-selection) 38%, transparent);
  }

  .downloads-history-row--expanded {
    background-color: color-mix(in oklab, var(--sdvm-raised) 40%, transparent);
  }

  .downloads-history-row--bulk-selected {
    background-color: color-mix(in oklab, var(--sdvm-selection-bulk) 85%, var(--sdvm-raised) 15%);
  }

  .downloads-history-row:last-child {
    border-bottom: none;
  }

  .downloads-history-body {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    min-width: 0;
  }

  .downloads-row-expand {
    flex-shrink: 0;
    min-height: var(--ctrl-h);
    width: var(--ctrl-h);
  }

  .downloads-row-expand :global(svg) {
    transition: transform var(--motion-fast) var(--ease-out-quart);
  }

  .downloads-row-expand--open :global(svg) {
    transform: rotate(180deg);
  }

  .downloads-row-detail {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-2);
    margin-top: var(--space-2);
    padding: var(--space-2) var(--space-1) 0 calc(var(--ctrl-h) + var(--space-1));
    border-top: 1px solid color-mix(in oklab, var(--sdvm-border) 60%, transparent);
  }

  .downloads-row-detail-hint {
    margin: 0;
    text-wrap: pretty;
  }

  .downloads-row-detail-line {
    margin: 0;
    width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .downloads-view-mod-btn {
    align-self: stretch;
  }

  .downloads-history-ident {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: calc(var(--space-1) / 2);
    cursor: pointer;
    border: none;
    background: transparent;
    padding: 0;
    text-align: left;
    color: inherit;
    font: inherit;
    border-radius: var(--radius-base, 0.25rem);
  }

  .downloads-history-ident:focus-visible {
    outline: 2px solid color-mix(in oklab, var(--color-primary-500) 55%, transparent);
    outline-offset: 1px;
  }

  .downloads-bulk-bar {
    margin: 0 calc(-1 * var(--space-4));
    padding-inline: var(--space-4);
  }

  .downloads-bulk-bar .bulk-count {
    font-weight: var(--weight-medium);
    color: var(--color-surface-200);
  }

  .downloads-bulk-clear {
    margin-inline-start: auto;
  }

  .downloads-bulk-range-hint {
    flex: 1 1 100%;
  }

  @container downloads-pane (max-width: 22rem) {
    .downloads-history-body {
      flex-wrap: wrap;
      row-gap: var(--space-2);
    }

    .downloads-history-ident {
      flex: 1 1 calc(100% - var(--ctrl-h) - var(--space-1));
    }

    .downloads-row-actions {
      flex-direction: row;
      width: 100%;
      justify-content: flex-end;
    }

    .downloads-primary-btn {
      flex: 1;
      max-width: 8.5rem;
    }
  }

  .downloads-history-mod-line {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    min-width: 0;
  }

  .downloads-mod-name {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-weight: 600;
  }

  .downloads-archive-name {
    display: block;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--color-surface-300);
  }

  .downloads-date {
    white-space: nowrap;
  }

  .downloads-row-actions {
    display: flex;
    flex-shrink: 0;
    flex-direction: column;
    align-items: stretch;
    gap: var(--space-1);
    width: 4.75rem;
  }

  .downloads-primary-btn {
    min-height: var(--ctrl-h);
    justify-content: center;
    font-weight: 600;
    white-space: nowrap;
  }

  .downloads-more-btn {
    min-height: var(--ctrl-h);
  }

  .downloads-more-btn[aria-expanded='true'] {
    background-color: color-mix(in oklab, var(--sdvm-raised) 88%, var(--color-primary-500) 12%);
    color: var(--color-surface-50);
  }

  .downloads-row-menu {
    width: min(12rem, calc(100vw - 1rem));
  }

  .downloads-row-menu :global(.overlay-menu-item--danger) {
    margin-top: var(--space-1);
    border-top: 1px solid var(--sdvm-divider);
  }

  .downloads-pane-resize-handle {
    position: absolute;
    top: 0;
    left: 0;
    z-index: 1;
    width: 0.375rem;
    height: 100%;
    padding: 0;
    border: 0;
    background: transparent;
    cursor: col-resize;
  }

  .downloads-pane-resize-handle::after {
    content: '';
    position: absolute;
    inset: 0 -0.125rem;
    border-left: 1px solid transparent;
    transition: border-color var(--motion-fast) var(--ease-out-quart);
  }

  .downloads-pane-resize-handle:hover::after,
  .downloads-pane.is-resizing .downloads-pane-resize-handle::after {
    border-left-color: color-mix(in oklab, var(--color-primary-400) 55%, transparent);
  }

  .downloads-pane-resize-handle:focus-visible::after {
    border-left-color: var(--color-primary-400);
  }

  .downloads-pane-empty {
    margin: var(--space-2) 0 0;
    padding: 0 0 var(--space-2);
    max-width: 38ch;
    text-wrap: pretty;
  }

  .downloads-refresh-indicator {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    z-index: 2;
  }

  .downloads-fetch-error {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-2);
    margin-bottom: var(--space-3);
    padding: var(--space-3);
    border: 1px solid color-mix(in oklab, var(--sdvm-error-fg) 35%, var(--sdvm-border));
    border-radius: var(--radius-base, 0.25rem);
    background-color: color-mix(in oklab, var(--sdvm-error-fg) 8%, var(--sdvm-raised));
  }

  .downloads-fetch-error-text {
    margin: 0;
    color: var(--color-surface-100);
    text-wrap: pretty;
  }

  .downloads-skeleton-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    margin: var(--space-2) 0 0;
    padding: 0;
    list-style: none;
  }

  .downloads-skeleton-list--history {
    flex: 1;
    min-height: 0;
    margin: 0;
    gap: var(--space-2);
  }

  .downloads-skeleton-row {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .downloads-skeleton-row--history {
    flex-direction: row;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2) 0;
    border-bottom: 1px solid color-mix(in oklab, var(--sdvm-border) 70%, transparent);
  }

  .downloads-skeleton-row--history:last-child {
    border-bottom: none;
  }

  .downloads-skeleton-ident {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: calc(var(--space-1) / 2);
  }

  .downloads-skeleton-actions {
    display: flex;
    flex-shrink: 0;
    flex-direction: column;
    gap: var(--space-1);
    width: 4.75rem;
  }

  .downloads-skeleton-line {
    display: block;
    border-radius: var(--radius-base, 0.25rem);
    background: linear-gradient(
      90deg,
      color-mix(in oklab, var(--sdvm-raised) 70%, transparent) 0%,
      color-mix(in oklab, var(--sdvm-border) 55%, transparent) 50%,
      color-mix(in oklab, var(--sdvm-raised) 70%, transparent) 100%
    );
    background-size: 200% 100%;
    animation: downloads-skeleton-shimmer 1.4s ease-in-out infinite;
  }

  .downloads-skeleton-line--title {
    height: 0.875rem;
    width: 72%;
  }

  .downloads-skeleton-line--bar {
    height: 0.25rem;
    width: 100%;
    border-radius: 9999px;
  }

  .downloads-skeleton-line--meta {
    height: 0.75rem;
    width: 88%;
  }

  .downloads-skeleton-line--date {
    height: 0.75rem;
    width: 38%;
  }

  .downloads-skeleton-line--search {
    height: var(--ctrl-h);
    width: 100%;
  }

  .downloads-skeleton-line--btn {
    height: var(--ctrl-h);
    width: 100%;
  }

  .downloads-skeleton-line--icon {
    height: var(--ctrl-h);
    width: 100%;
  }

  @keyframes downloads-skeleton-shimmer {
    0% {
      background-position: 200% 0;
    }
    100% {
      background-position: -200% 0;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .downloads-progress-bar {
      transition: none;
    }

    .downloads-row-expand,
    .downloads-row-expand :global(svg) {
      transition: none;
    }

    .downloads-pane-resize-handle::after {
      transition: none;
    }

    .downloads-history-row {
      transition: none;
    }

    .downloads-skeleton-line {
      animation: none;
      background: color-mix(in oklab, var(--sdvm-raised) 75%, transparent);
    }
  }
</style>
