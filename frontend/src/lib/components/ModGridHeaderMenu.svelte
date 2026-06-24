<script lang="ts">
  import {
    GRID_COLUMNS,
    isColumnVisible,
    normalizeVisibleColumns,
    type GridColumnId,
  } from '$lib/mods/gridColumns';
  import * as m from '$lib/paraglide/messages.js';

  interface Props {
    x: number;
    y: number;
    visibleColumns: string[] | null | undefined;
    ontoggle: (id: GridColumnId, visible: boolean) => void | Promise<void>;
    onclose: () => void;
  }

  let { x, y, visibleColumns, ontoggle, onclose }: Props = $props();

  let menuEl = $state<HTMLDivElement | undefined>();

  const activeColumns = $derived(normalizeVisibleColumns(visibleColumns));

  const menuPos = $derived({
    left: Math.min(x, Math.max(0, window.innerWidth - 220)),
    top: Math.min(y, Math.max(0, window.innerHeight - 320)),
  });

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
    queueMicrotask(() => menuEl?.focus());
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        onclose();
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
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  });
</script>

<div class="overlay-scrim overlay-scrim--menu" role="presentation" onclick={onclose}>
  <div
    bind:this={menuEl}
    class="overlay-menu-panel"
    style:left="{menuPos.left}px"
    style:top="{menuPos.top}px"
    onclick={(e) => e.stopPropagation()}
    onkeydown={(e) => e.stopPropagation()}
    role="menu"
    aria-label={m.mod_grid_columns_title()}
    tabindex="-1"
  >
    <p class="overlay-menu-title type-caption type-meta">{m.mod_grid_columns_title()}</p>
    {#each GRID_COLUMNS as column (column.id)}
      {@const checked = isColumnVisible(activeColumns, column.id)}
      <button
        type="button"
        class="overlay-menu-item"
        class:overlay-menu-item--disabled={column.required}
        role="menuitemcheckbox"
        aria-checked={checked}
        aria-disabled={column.required}
        disabled={column.required}
        onclick={() => {
          if (column.required) return;
          void ontoggle(column.id, !checked);
        }}
      >
        <span class="overlay-menu-check" aria-hidden="true">{checked ? '✓' : ''}</span>
        <span class="truncate">{column.label}</span>
      </button>
    {/each}
    <p class="overlay-menu-hint type-caption type-meta">{m.mod_grid_columns_required_hint()}</p>
  </div>
</div>
