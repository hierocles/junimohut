<script lang="ts">
  import { PanelLeftClose, Pencil } from '@lucide/svelte';
  import type { Category } from '$lib/api/client';
  import {
    TAG_COLOR_PRESETS,
    tagColorAria,
    tagsDeleteAria,
    tagsDeleteLabel,
    tagsFilterToggleAria,
    tagsFilterToggleTitle,
    tagsRenameAria,
    tagsRenameLabel,
    tagsResizeAria,
    tagsSidebarAllShown,
    tagsSidebarAria,
    tagsSidebarCancel,
    tagsSidebarColorGroupLabel,
    tagsSidebarColorLegend,
    tagsSidebarCreateFirst,
    tagsSidebarCreateTag,
    tagsSidebarCreating,
    tagsSidebarEmptyHint,
    tagsSidebarEmptyTitle,
    tagsSidebarFilterMeta,
    tagsSidebarFooterHint,
    tagsSidebarHideAria,
    tagsSidebarHideTitle,
    tagsSidebarNew,
    tagsSidebarShowAll,
    tagsSidebarTagNameLabel,
    tagsSidebarTagNamePlaceholder,
    tagsSidebarTitle,
  } from '$lib/copy';

  const MIN_WIDTH = 200;
  const MAX_WIDTH = 480;
  const DEFAULT_TAG_COLOR = TAG_COLOR_PRESETS[0].hex;

  interface Props {
    categories: Category[];
    creating?: boolean;
    ontoggle: (id: string, visible: boolean) => void;
    onshowall: () => void | Promise<void>;
    oncreate: (name: string, color: string) => void | Promise<void>;
    onrename: (id: string, name: string) => void | Promise<void>;
    ondelete: (id: string) => void;
    onwidthchange: (width: number) => void;
    onhide: () => void;
  }

  let {
    categories,
    creating = false,
    ontoggle,
    onshowall,
    oncreate,
    onrename,
    ondelete,
    onwidthchange,
    onhide,
  }: Props = $props();

  let editingId = $state<string | null>(null);
  let editingName = $state('');
  let editInput = $state<HTMLInputElement | undefined>();
  let renameBusy = $state(false);

  function startRename(cat: Category) {
    editingId = cat.id;
    editingName = cat.name;
    queueMicrotask(() => editInput?.select());
  }

  function cancelRename() {
    editingId = null;
    editingName = '';
  }

  async function submitRename(id: string) {
    const trimmed = editingName.trim();
    if (!trimmed || renameBusy) return cancelRename();
    renameBusy = true;
    try {
      await onrename(id, trimmed);
    } finally {
      renameBusy = false;
      editingId = null;
    }
  }

  let newName = $state('');
  let newColor = $state<string>(DEFAULT_TAG_COLOR);
  let showForm = $state(false);
  let nameInput = $state<HTMLInputElement | undefined>();
  let sidebarResize = $state<{ startX: number; startWidth: number } | null>(null);

  const sortedCategories = $derived(
    [...categories].sort((a, b) => a.sortOrder - b.sortOrder || a.name.localeCompare(b.name)),
  );

  const visibleFilterCount = $derived(categories.filter((c) => c.visible).length);
  const filterNarrowed = $derived(
    categories.length > 0 &&
      visibleFilterCount > 0 &&
      visibleFilterCount < categories.length,
  );

  function openCreateForm() {
    showForm = true;
    newName = '';
    newColor = DEFAULT_TAG_COLOR;
  }

  function closeCreateForm() {
    if (creating) return;
    showForm = false;
    newName = '';
  }

  function startSidebarResize(e: MouseEvent) {
    e.preventDefault();
    const shell = (e.currentTarget as HTMLElement).closest('.tag-sidebar') as HTMLElement | null;
    const width = shell?.getBoundingClientRect().width ?? MIN_WIDTH;
    sidebarResize = { startX: e.clientX, startWidth: width };
  }

  function onSidebarResizeMove(e: MouseEvent) {
    if (!sidebarResize) return;
    const next = Math.min(
      MAX_WIDTH,
      Math.max(MIN_WIDTH, sidebarResize.startWidth + (e.clientX - sidebarResize.startX)),
    );
    onwidthchange(next);
  }

  function endSidebarResize() {
    sidebarResize = null;
  }

  $effect(() => {
    if (!showForm) return;
    queueMicrotask(() => nameInput?.focus());
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        closeCreateForm();
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  });

  async function submitCreate(e: Event) {
    e.preventDefault();
    const trimmed = newName.trim();
    if (!trimmed || creating) return;
    await oncreate(trimmed, newColor);
    newName = '';
    showForm = false;
  }

  const showFooterHint = $derived(sortedCategories.length > 0 && !showForm);
</script>

<svelte:window onmousemove={onSidebarResizeMove} onmouseup={endSidebarResize} />

<aside id="tags-sidebar" class="tag-sidebar" class:is-resizing={sidebarResize != null} aria-label={tagsSidebarAria}>
  <header class="sidebar-header">
    <div class="sidebar-heading">
      <h2 class="type-section-head">{tagsSidebarTitle}</h2>
      {#if categories.length > 0}
        <div class="sidebar-meta-row">
          {#if filterNarrowed}
            <span class="sidebar-meta type-caption type-data tabular-nums">
              {tagsSidebarFilterMeta(visibleFilterCount)}
            </span>
            <button type="button" class="sidebar-link-btn type-caption" onclick={() => void onshowall()}>
              {tagsSidebarShowAll}
            </button>
          {:else}
            <span class="sidebar-meta type-caption type-meta">{tagsSidebarAllShown}</span>
          {/if}
        </div>
      {/if}
    </div>
    <div class="sidebar-header-actions">
      <button
        type="button"
        class="btn btn-sm preset-tonal toolbar-icon-btn shrink-0"
        onclick={onhide}
        title={tagsSidebarHideTitle}
        aria-label={tagsSidebarHideAria}
        aria-controls="tags-sidebar"
        aria-expanded={true}
      >
        <PanelLeftClose size={14} aria-hidden="true" />
      </button>
      <button
        type="button"
        class="btn btn-sm preset-tonal shrink-0"
        onclick={() => (showForm ? closeCreateForm() : openCreateForm())}
        aria-expanded={showForm}
        aria-controls="tag-create-panel"
      >
        {showForm ? tagsSidebarCancel : tagsSidebarNew}
      </button>
    </div>
  </header>

  {#if showForm}
    <form id="tag-create-panel" class="create-panel layout-stack-sm motion-reveal" onsubmit={submitCreate}>
      <label class="label">
        <span class="label-text">{tagsSidebarTagNameLabel}</span>
        <input
          bind:this={nameInput}
          class="input input-sm w-full min-w-0"
          bind:value={newName}
          placeholder={tagsSidebarTagNamePlaceholder}
          maxlength="80"
          required
          disabled={creating}
        />
      </label>

      <fieldset class="color-fieldset">
        <legend class="type-label label-text">{tagsSidebarColorLegend}</legend>
        <div class="color-swatches" role="radiogroup" aria-label={tagsSidebarColorGroupLabel}>
          {#each TAG_COLOR_PRESETS as preset (preset.hex)}
            <button
              type="button"
              class="color-swatch"
              class:color-swatch--selected={newColor === preset.hex}
              style:--swatch-color={preset.hex}
              role="radio"
              aria-checked={newColor === preset.hex}
              aria-label={tagColorAria(preset.label)}
              title={preset.label}
              disabled={creating}
              onclick={() => (newColor = preset.hex)}
            ></button>
          {/each}
        </div>
      </fieldset>

      <button
        type="submit"
        class="btn btn-sm preset-filled-primary-500 w-full"
        disabled={creating || !newName.trim()}
        aria-busy={creating}
      >
        {creating ? tagsSidebarCreating : tagsSidebarCreateTag}
      </button>
    </form>
  {/if}

  <div class="tag-scroll">
    {#if sortedCategories.length === 0 && !showForm}
      <div class="empty-state layout-stack-sm">
        <p class="empty-state-title type-ui">{tagsSidebarEmptyTitle}</p>
        <p class="empty-state-hint type-caption type-meta type-prose">
          {tagsSidebarEmptyHint}
        </p>
        <button type="button" class="btn btn-sm preset-tonal w-full" onclick={openCreateForm}>
          {tagsSidebarCreateFirst}
        </button>
      </div>
    {:else}
      <ul class="tag-list" role="list">
        {#each sortedCategories as cat (cat.id)}
          <li
            class="tag-row"
            class:tag-row--filter-on={cat.visible}
            style:--chip-color={cat.color || DEFAULT_TAG_COLOR}
          >
            <label
              class="filter-control"
              title={tagsFilterToggleTitle(cat.name, cat.visible)}
            >
              <input
                type="checkbox"
                class="filter-input"
                checked={cat.visible}
                onchange={(e) => ontoggle(cat.id, (e.currentTarget as HTMLInputElement).checked)}
                aria-label={tagsFilterToggleAria(cat.name, cat.visible)}
              />
              <span class="filter-track" aria-hidden="true"></span>
            </label>

            <div class="tag-main">
              <span class="tag-dot" aria-hidden="true"></span>
              {#if editingId === cat.id}
                <input
                  bind:this={editInput}
                  class="input input-sm tag-rename-input"
                  bind:value={editingName}
                  maxlength="80"
                  disabled={renameBusy}
                  aria-label={tagsRenameLabel}
                  onblur={() => submitRename(cat.id)}
                  onkeydown={(e) => {
                    if (e.key === 'Enter') { e.preventDefault(); void submitRename(cat.id); }
                    if (e.key === 'Escape') { e.preventDefault(); cancelRename(); }
                  }}
                />
              {:else}
                <span class="tag-name type-ui">{cat.name}</span>
                <span class="tag-count type-caption type-data tabular-nums">{cat.modIds?.length ?? 0}</span>
              {/if}
            </div>

            <div class="tag-actions">
              {#if editingId !== cat.id}
                <button
                  type="button"
                  class="tag-action-btn"
                  title={tagsRenameLabel}
                  aria-label={tagsRenameAria(cat.name)}
                  onclick={() => startRename(cat)}
                >
                  <Pencil size={13} aria-hidden="true" />
                </button>
              {/if}
              <button
                type="button"
                class="delete-x"
                title={tagsDeleteLabel}
                aria-label={tagsDeleteAria(cat.name)}
                onclick={() => ondelete(cat.id)}
              >
                ×
              </button>
            </div>
          </li>
        {/each}
      </ul>
    {/if}
  </div>

  {#if showFooterHint}
    <footer class="sidebar-footer type-caption type-meta type-prose">
      {tagsSidebarFooterHint}
    </footer>
  {/if}

  <button
    type="button"
    class="sidebar-resize-handle"
    aria-label={tagsResizeAria}
    onmousedown={startSidebarResize}
  ></button>
</aside>

<style>
  .tag-sidebar {
    position: relative;
    display: flex;
    flex-direction: column;
    width: 100%;
    min-width: 0;
    height: 100%;
    padding: var(--space-3);
    gap: var(--space-3);
    border-right: 1px solid var(--sdvm-border);
    background-color: var(--sdvm-shell);
    overflow: hidden;
  }

  .tag-sidebar.is-resizing {
    cursor: col-resize;
    user-select: none;
  }

  .sidebar-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--space-2);
    flex-shrink: 0;
  }

  .sidebar-heading {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    min-width: 0;
    flex: 1;
  }

  .sidebar-header-actions {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    flex-shrink: 0;
  }

  .sidebar-meta-row {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: var(--space-2);
  }

  .sidebar-link-btn {
    padding: 0;
    border: 0;
    background: transparent;
    color: var(--color-primary-400);
    font: inherit;
    cursor: pointer;
    text-decoration: underline;
    text-underline-offset: 0.15em;
  }

  .sidebar-link-btn:hover,
  .sidebar-link-btn:focus-visible {
    color: var(--color-primary-300);
  }

  .sidebar-link-btn:focus-visible {
    outline: 2px solid var(--color-primary-400);
    outline-offset: 2px;
    border-radius: var(--radius-base, 0.25rem);
  }

  .sidebar-meta {
    color: var(--color-surface-500);
  }

  .create-panel {
    flex-shrink: 0;
    padding: var(--space-3);
    border: 1px solid var(--sdvm-border);
    border-radius: var(--radius-lg, 0.5rem);
    background-color: var(--sdvm-panel);
  }

  .color-fieldset {
    margin: 0;
    padding: 0;
    border: 0;
  }

  .color-fieldset legend {
    margin-bottom: var(--space-2);
  }

  .color-swatches {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
  }

  .color-swatch {
    width: 1.375rem;
    height: 1.375rem;
    padding: 0;
    border: 2px solid transparent;
    border-radius: 999px;
    background-color: var(--swatch-color);
    cursor: pointer;
    transition: transform var(--motion-fast) var(--ease-out-quart);
  }

  .color-swatch--selected {
    box-shadow:
      0 0 0 2px var(--sdvm-shell),
      0 0 0 4px var(--swatch-color);
  }

  .color-swatch:focus-visible {
    outline: 2px solid var(--color-primary-400);
    outline-offset: 2px;
  }

  .tag-scroll {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    overflow-x: hidden;
  }

  .tag-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .tag-row {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr) auto;
    align-items: center;
    gap: var(--space-2);
    min-width: 0;
    padding: var(--space-1);
    border-radius: var(--radius-lg, 0.5rem);
    transition: background-color var(--motion-fast) var(--ease-out-quart);
  }

  .tag-row:hover,
  .tag-row:focus-within {
    background-color: color-mix(in oklab, var(--chip-color) 8%, var(--sdvm-raised));
  }

  .tag-row--filter-on {
    background-color: color-mix(in oklab, var(--chip-color) 5%, transparent);
  }

  .filter-control {
    position: relative;
    display: flex;
    align-items: center;
    flex-shrink: 0;
    cursor: pointer;
  }

  .filter-input {
    position: absolute;
    width: 1px;
    height: 1px;
    margin: -1px;
    padding: 0;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    border: 0;
  }

  .filter-track {
    display: block;
    width: 1.75rem;
    height: 1rem;
    border-radius: 999px;
    background-color: var(--color-surface-700);
    transition: background-color var(--motion-fast) var(--ease-out-quart);
  }

  .filter-track::after {
    content: '';
    display: block;
    width: 0.75rem;
    height: 0.75rem;
    margin: 0.125rem;
    border-radius: 999px;
    background-color: var(--color-surface-300);
    transition: transform var(--motion-fast) var(--ease-out-quart);
  }

  .filter-input:checked + .filter-track {
    background-color: color-mix(in oklab, var(--chip-color) 55%, var(--color-surface-800));
  }

  .filter-input:checked + .filter-track::after {
    transform: translateX(0.75rem);
    background-color: var(--color-surface-50);
  }

  .filter-input:focus-visible + .filter-track {
    outline: 2px solid var(--color-primary-400);
    outline-offset: 2px;
  }

  .tag-main {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    min-width: 0;
    padding: var(--space-2);
  }

  .tag-dot {
    width: 0.5rem;
    height: 0.5rem;
    flex-shrink: 0;
    border-radius: 999px;
    background-color: var(--chip-color);
  }

  .tag-name {
    flex: 1;
    min-width: 0;
    font-weight: var(--weight-medium);
    color: var(--color-surface-100);
    overflow-wrap: anywhere;
    word-break: break-word;
  }

  .tag-count {
    flex-shrink: 0;
    color: var(--color-surface-500);
  }

  .tag-actions {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    flex-shrink: 0;
  }

  .tag-rename-input {
    flex: 1;
    min-width: 0;
    font-size: var(--type-ui);
    height: 1.5rem;
    padding-inline: var(--space-1);
  }

  .tag-action-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 1.5rem;
    height: 1.5rem;
    padding: 0;
    border: 0;
    border-radius: var(--radius-base, 0.25rem);
    background: transparent;
    color: var(--color-surface-500);
    font-size: var(--type-ui);
    line-height: 1;
    cursor: pointer;
    opacity: 0.55;
    transition:
      color var(--motion-fast) var(--ease-out-quart),
      background-color var(--motion-fast) var(--ease-out-quart),
      opacity var(--motion-fast) var(--ease-out-quart);
  }

  .tag-row:hover .tag-action-btn,
  .tag-action-btn:focus-visible {
    opacity: 1;
  }

  .tag-action-btn:hover,
  .tag-action-btn:focus-visible {
    color: var(--color-surface-100);
    background-color: var(--color-surface-700);
  }

  .tag-action-btn:focus-visible {
    outline: 2px solid var(--color-primary-400);
    outline-offset: 1px;
  }

  .delete-x {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 1.5rem;
    height: 1.5rem;
    padding: 0;
    border: 0;
    border-radius: var(--radius-base, 0.25rem);
    background: transparent;
    color: var(--color-surface-500);
    font-size: var(--type-subhead);
    line-height: 1;
    cursor: pointer;
    opacity: 0.75;
    transition:
      color var(--motion-fast) var(--ease-out-quart),
      background-color var(--motion-fast) var(--ease-out-quart),
      opacity var(--motion-fast) var(--ease-out-quart);
  }

  .tag-row:hover .delete-x,
  .delete-x:focus-visible {
    opacity: 1;
  }

  .delete-x:hover,
  .delete-x:focus-visible {
    color: var(--sdvm-error-fg);
    background-color: var(--sdvm-error-bg);
  }

  .delete-x:focus-visible {
    outline: 2px solid color-mix(in oklab, var(--color-error-500) 50%, transparent);
    outline-offset: 1px;
  }

  .empty-state {
    padding: var(--space-6) var(--space-2);
    text-align: center;
  }

  .empty-state-title {
    font-weight: var(--weight-semibold);
    color: var(--color-surface-200);
    text-wrap: balance;
  }

  .empty-state-hint {
    line-height: var(--leading-relaxed, 1.5);
    text-wrap: pretty;
  }

  .sidebar-footer {
    flex-shrink: 0;
    padding-top: var(--space-2);
    border-top: 1px solid color-mix(in oklab, var(--sdvm-border) 80%, transparent);
    line-height: var(--leading-relaxed, 1.5);
    text-wrap: pretty;
  }

  .sidebar-resize-handle {
    position: absolute;
    top: 0;
    right: 0;
    z-index: 1;
    width: 0.375rem;
    height: 100%;
    padding: 0;
    border: 0;
    background: transparent;
    cursor: col-resize;
    touch-action: none;
  }

  .sidebar-resize-handle::after {
    content: '';
    position: absolute;
    top: 10%;
    bottom: 10%;
    left: 50%;
    width: 1px;
    transform: translateX(-50%);
    background-color: color-mix(in oklab, var(--color-surface-700) 70%, transparent);
    transition: background-color var(--motion-fast) var(--ease-out-quart);
  }

  .sidebar-resize-handle:hover::after,
  .tag-sidebar.is-resizing .sidebar-resize-handle::after {
    background-color: var(--color-primary-500);
    width: 2px;
  }

  .sidebar-resize-handle:focus-visible::after {
    background-color: var(--color-primary-500);
    width: 2px;
  }

  @media (prefers-reduced-motion: reduce) {
    .tag-row,
    .filter-track,
    .filter-track::after,
    .color-swatch,
    .delete-x,
    .sidebar-resize-handle::after {
      transition: none;
    }
  }
</style>
