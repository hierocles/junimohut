<script lang="ts">
  import type { Mod } from "$lib/api/client";
  import { resolvedNexusModId } from "$lib/mods/resolveNexus";
  import * as m from "$lib/paraglide/messages.js";

  interface Props {
    mod: Mod | null;
    x: number;
    y: number;
    /** Hide Nexus update actions on bundle part rows (parent owns updates). */
    suppressUpdateActions?: boolean;
    onaction: (action: string) => void;
    onclose: () => void;
  }

  let {
    mod,
    x,
    y,
    suppressUpdateActions = false,
    onaction,
    onclose,
  }: Props = $props();

  let menuEl = $state<HTMLDivElement | undefined>();

  const hasSavedDownload = $derived(!!mod?.savedDownloadPath?.trim());
  const hasNexus = $derived(mod != null && resolvedNexusModId(mod) > 0);
  const hasUpdate = $derived(
    !suppressUpdateActions &&
      (mod?.updateStatus?.state === "update" ||
        mod?.updateStatus?.state === "update_available"),
  );
  const updateIgnored = $derived(
    !suppressUpdateActions && mod?.updateStatus?.state === "update_ignored",
  );

  const menuPos = $derived({
    left: Math.min(x, Math.max(0, window.innerWidth - 220)),
    top: Math.min(y, Math.max(0, window.innerHeight - 280)),
  });

  const menuItems = $derived([
    { action: "openFolder", label: m.context_menu_open_folder() },
    { action: "openManifest", label: m.context_menu_open_manifest() },
    ...(mod?.hasJsonFiles
      ? [{ action: "editConfig", label: m.context_menu_edit_config() }]
      : []),
    { action: "rename", label: m.mod_rename_label() },
    ...(hasSavedDownload
      ? [{ action: "reinstallSaved", label: m.context_menu_reinstall_saved_label() }]
      : []),
    ...(hasNexus
      ? [
          { action: "openPage", label: m.context_menu_view_nexus() },
          { action: "endorse", label: m.context_menu_endorse() },
          ...(hasUpdate
            ? [{ action: "ignoreUpdate", label: m.context_menu_ignore_update() }]
            : []),
          ...(updateIgnored
            ? [{ action: "resumeUpdate", label: m.context_menu_resume_update() }]
            : []),
          ...(hasUpdate
            ? [{ action: "downloadUpdate", label: m.context_menu_download_update() }]
            : []),
        ]
      : []),
    { action: "delete", label: m.context_menu_delete_mod(), danger: true },
  ] as const);

  function focusMenuItem(delta: number) {
    const buttons = menuEl?.querySelectorAll<HTMLButtonElement>(
      "button.overlay-menu-item",
    );
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
      if (e.key === "Escape") {
        e.preventDefault();
        onclose();
        return;
      }
      if (e.key === "ArrowDown") {
        e.preventDefault();
        focusMenuItem(1);
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        focusMenuItem(-1);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });
</script>

{#if mod}
  <div
    class="overlay-scrim overlay-scrim--menu"
    role="presentation"
    onclick={onclose}
  >
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
          class:overlay-menu-item--danger={"danger" in item && item.danger}
          role="menuitem"
          onclick={() => onaction(item.action)}
        >
          {item.label}
        </button>
      {/each}
    </div>
  </div>
{/if}
