<script lang="ts">
  import { X } from '@lucide/svelte';
  import type { Category, Mod } from '$lib/api/client';

  interface Props {
    mod: Mod;
    categories: Category[];
    x: number;
    y: number;
    busy?: boolean;
    ontoggle: (categoryId: string, assign: boolean) => void | Promise<void>;
    onclose: () => void;
  }

  let { mod, categories, x, y, busy = false, ontoggle, onclose }: Props = $props();

  let panelEl = $state<HTMLDivElement | undefined>();
  let toggleBusy = $state<string | null>(null);

  const sorted = $derived(
    [...categories].sort((a, b) => a.sortOrder - b.sortOrder || a.name.localeCompare(b.name)),
  );

  const assignedIds = $derived(new Set(mod.categoryIds ?? []));

  const panelPos = $derived({
    left: Math.min(x, Math.max(8, window.innerWidth - 240)),
    top: Math.min(y, Math.max(8, window.innerHeight - 320)),
  });

  function isAssigned(cat: Category): boolean {
    return assignedIds.has(cat.id);
  }

  async function handleToggle(cat: Category) {
    if (busy || toggleBusy) return;
    toggleBusy = cat.id;
    try {
      await ontoggle(cat.id, !isAssigned(cat));
    } finally {
      toggleBusy = null;
    }
  }

  function focusItem(delta: number) {
    const buttons = panelEl?.querySelectorAll<HTMLButtonElement>('button.tag-picker-item');
    if (!buttons?.length) return;
    const current = document.activeElement;
    let idx = [...buttons].indexOf(current as HTMLButtonElement);
    if (idx < 0) idx = 0;
    else if (delta > 0) idx = Math.min(idx + 1, buttons.length - 1);
    else idx = Math.max(idx - 1, 0);
    buttons[idx]?.focus();
  }

  $effect(() => {
    queueMicrotask(() => panelEl?.focus());
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        e.stopPropagation();
        onclose();
        return;
      }
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        focusItem(1);
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        focusItem(-1);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  });
</script>

<div class="overlay-scrim overlay-scrim--menu" role="presentation" onclick={onclose}>
  <div
    bind:this={panelEl}
    class="tag-picker-panel overlay-floating-panel app-panel"
    style:left="{panelPos.left}px"
    style:top="{panelPos.top}px"
    onclick={(e) => e.stopPropagation()}
    onkeydown={(e) => e.stopPropagation()}
    role="dialog"
    aria-labelledby="tag-picker-title"
    tabindex="-1"
  >
    <header class="overlay-panel-header tag-picker-header">
      <h3 id="tag-picker-title" class="type-section-head overlay-panel-title">Tags</h3>
      <button
        type="button"
        class="btn btn-sm preset-tonal toolbar-icon-btn"
        aria-label="Close"
        onclick={onclose}
      >
        <X size={14} />
      </button>
    </header>

    {#if sorted.length === 0}
      <p class="type-ui type-meta type-prose tag-picker-empty">No tags yet. Create tags in the sidebar.</p>
    {:else}
      <ul class="tag-picker-list" role="listbox" aria-label="Available tags">
        {#each sorted as cat (cat.id)}
          {@const assigned = isAssigned(cat)}
          {@const rowBusy = toggleBusy === cat.id}
          <li>
            <button
              type="button"
              class="tag-picker-item"
              class:tag-picker-item--on={assigned}
              style:--chip-color={cat.color || '#6366f1'}
              role="option"
              aria-selected={assigned}
              aria-busy={rowBusy}
              disabled={busy || rowBusy}
              onclick={() => handleToggle(cat)}
            >
              <span class="tag-picker-dot" aria-hidden="true"></span>
              <span class="tag-picker-name type-ui truncate">{cat.name}</span>
              {#if assigned}
                <span class="tag-picker-check type-caption" aria-hidden="true">✓</span>
              {/if}
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</div>

<style>
  .tag-picker-panel {
    position: fixed;
    width: min(14rem, calc(100vw - 1rem));
    max-height: min(20rem, calc(100vh - 2rem));
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-3);
    overflow: hidden;
    margin: 0;
    box-shadow: var(--overlay-shadow);
    animation: motion-panel-enter var(--motion-medium) var(--ease-out-quart) both;
  }

  .tag-picker-header {
    margin-bottom: 0;
  }

  .tag-picker-empty {
    margin: 0;
  }

  .tag-picker-list {
    margin: 0;
    padding: 0;
    list-style: none;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .tag-picker-item {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    width: 100%;
    min-width: 0;
    padding: var(--space-2) var(--space-3);
    border: 1px solid transparent;
    border-radius: var(--radius-base, 0.25rem);
    background: transparent;
    color: inherit;
    text-align: left;
    cursor: pointer;
    transition: background-color var(--motion-fast) var(--ease-out-quart);
  }

  .tag-picker-item:hover:not(:disabled) {
    background-color: color-mix(in oklab, var(--chip-color) 10%, var(--sdvm-raised));
  }

  .tag-picker-item--on {
    background-color: color-mix(in oklab, var(--chip-color) 14%, var(--sdvm-raised));
    border-color: color-mix(in oklab, var(--chip-color) 35%, transparent);
  }

  .tag-picker-item:focus-visible {
    outline: 2px solid color-mix(in oklab, var(--color-primary-500) 55%, transparent);
    outline-offset: -2px;
  }

  .tag-picker-item:disabled {
    opacity: 0.7;
    cursor: default;
  }

  .tag-picker-dot {
    width: 0.5rem;
    height: 0.5rem;
    flex-shrink: 0;
    border-radius: 999px;
    background-color: var(--chip-color);
  }

  .tag-picker-name {
    flex: 1;
    min-width: 0;
    font-weight: var(--weight-medium);
  }

  .tag-picker-check {
    flex-shrink: 0;
    color: color-mix(in oklab, var(--chip-color) 75%, var(--color-surface-50));
    font-weight: var(--weight-bold);
  }

  @media (prefers-reduced-motion: reduce) {
    .tag-picker-item {
      transition: none;
    }

    .tag-picker-panel {
      animation: none;
    }
  }
</style>
