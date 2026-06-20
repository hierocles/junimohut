<script lang="ts">
  import type { Mod } from '$lib/api/client';
  import { contextMenuReinstallSavedLabel } from '$lib/copy';

  interface Props {
    mod: Mod | null;
    x: number;
    y: number;
    onaction: (action: string) => void;
    onclose: () => void;
  }

  let { mod, x, y, onaction, onclose }: Props = $props();

  let menuEl = $state<HTMLDivElement | undefined>();

  const hasSavedDownload = $derived(!!mod?.savedDownloadPath?.trim());
  const hasNexus = $derived(mod?.manifest?.UpdateKeys?.some((k) => k.startsWith('Nexus:')) ?? false);

  const menuPos = $derived({
    left: Math.min(x, Math.max(0, window.innerWidth - 220)),
    top: Math.min(y, Math.max(0, window.innerHeight - 280)),
  });

  const menuItems = $derived(
    [
      { action: 'openFolder', label: 'Open mod folder' },
      { action: 'openManifest', label: 'Open manifest.json' },
      ...(hasSavedDownload
        ? [{ action: 'reinstallSaved', label: contextMenuReinstallSavedLabel }]
        : []),
      ...(hasNexus
        ? [
            { action: 'openPage', label: 'View on Nexus Mods' },
            { action: 'endorse', label: 'Endorse on Nexus Mods' },
            { action: 'downloadUpdate', label: 'Download update' },
          ]
        : []),
      { action: 'delete', label: 'Delete mod…', danger: true },
    ] as const,
  );

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
    if (!mod) return;
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

{#if mod}
  <div class="overlay-scrim overlay-scrim--menu" role="presentation" onclick={onclose}>
    <div
      bind:this={menuEl}
      class="overlay-menu-panel"
      style:left="{menuPos.left}px"
      style:top="{menuPos.top}px"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.stopPropagation()}
      role="menu"
      tabindex="-1"
    >
      {#each menuItems as item (item.action)}
        <button
          type="button"
          class="overlay-menu-item truncate"
          class:overlay-menu-item--danger={item.danger}
          role="menuitem"
          onclick={() => onaction(item.action)}
        >
          {item.label}
        </button>
      {/each}
    </div>
  </div>
{/if}
