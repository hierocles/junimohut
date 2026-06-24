<script lang="ts">
  import ConfirmDialog from "$lib/components/ConfirmDialog.svelte";
  import type { DuplicateModGroup } from "$lib/api/client";
  import {
    duplicateModsDialogTitle,
    duplicateModsDialogMessage,
    duplicateModsCleanupLabel,
    duplicateModsDismissLabel,
    duplicateModsKeepLabel,
  } from "$lib/i18n/helpers";

  interface Props {
    open: boolean;
    groups: DuplicateModGroup[];
    busy?: boolean;
    oncleanup: () => void | Promise<void>;
    onclose: () => void;
  }

  let { open, groups, busy = false, oncleanup, onclose }: Props = $props();
</script>

<ConfirmDialog
  {open}
  title={duplicateModsDialogTitle()}
  message={duplicateModsDialogMessage()}
  confirmLabel={duplicateModsCleanupLabel()}
  cancelLabel={duplicateModsDismissLabel()}
  {busy}
  onconfirm={oncleanup}
  oncancel={onclose}
>
  <ul class="duplicate-mod-list layout-stack-sm" role="list">
    {#each groups as group (group.uniqueID)}
      <li class="duplicate-mod-item">
        <div class="duplicate-mod-item__header">
          <span class="type-ui text-surface-100">{group.modName}</span>
          <span class="type-caption type-meta">{group.uniqueID}</span>
        </div>
        <ul class="duplicate-mod-folder-list layout-stack-xs" role="list">
          {#each group.folders ?? [] as folder (folder)}
            <li class="duplicate-mod-folder-item">
              <span class="type-caption type-mono">{folder}</span>
              {#if folder === group.canonical}
                <span class="duplicate-mod-keep-badge type-caption"
                  >{duplicateModsKeepLabel()}</span
                >
              {/if}
            </li>
          {/each}
        </ul>
      </li>
    {/each}
  </ul>
</ConfirmDialog>

<style>
  .duplicate-mod-list {
    margin: 0;
    padding: 0;
    list-style: none;
    max-height: min(40vh, 20rem);
    overflow: auto;
  }

  .duplicate-mod-item {
    padding: 0.75rem;
    border: 1px solid var(--color-surface-700, rgba(255, 255, 255, 0.08));
    border-radius: 0.5rem;
  }

  .duplicate-mod-item__header {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    margin-bottom: 0.5rem;
  }

  .duplicate-mod-folder-list {
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .duplicate-mod-folder-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .duplicate-mod-keep-badge {
    color: var(--color-accent-300, #9fd89f);
    font-weight: 600;
  }
</style>
