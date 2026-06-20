<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    open: boolean;
    title: string;
    message: string;
    confirmLabel?: string;
    cancelLabel?: string;
    variant?: 'danger' | 'default';
    busy?: boolean;
    children?: Snippet;
    onconfirm: () => void | Promise<void>;
    oncancel: () => void;
  }

  let {
    open,
    title,
    message,
    confirmLabel = 'Confirm',
    cancelLabel = 'Cancel',
    variant = 'default',
    busy = false,
    children,
    onconfirm,
    oncancel,
  }: Props = $props();

  let dialogEl = $state<HTMLDialogElement | undefined>();
  let cancelBtn = $state<HTMLButtonElement | undefined>();

  $effect(() => {
    const el = dialogEl;
    if (!el) return;
    if (open && !el.open) {
      el.showModal();
      queueMicrotask(() => cancelBtn?.focus());
    } else if (!open && el.open) {
      el.close();
    }
  });

  function handleCancel() {
    if (busy) return;
    oncancel();
  }

  async function handleConfirm() {
    if (busy) return;
    await onconfirm();
  }

  function onDialogCancel(e: Event) {
    e.preventDefault();
    handleCancel();
  }
</script>

<dialog
  bind:this={dialogEl}
  class="confirm-dialog overlay-dialog"
  aria-labelledby="confirm-title"
  aria-describedby="confirm-message"
  onclose={() => {
    if (!busy) oncancel();
  }}
  oncancel={onDialogCancel}
>
  <div class="confirm-dialog-panel card app-panel border app-border layout-stack-sm motion-dialog-enter">
    <h2 id="confirm-title" class="type-title text-surface-50">{title}</h2>
    <p id="confirm-message" class="type-ui type-meta type-prose">{message}</p>
    {#if children}
      <div class="confirm-dialog-extra layout-stack-sm">
        {@render children()}
      </div>
    {/if}
    <div class="confirm-dialog-actions flex" style="gap: var(--space-2);">
      <button
        bind:this={cancelBtn}
        type="button"
        class="btn preset-tonal flex-1"
        disabled={busy}
        onclick={handleCancel}
      >
        {cancelLabel}
      </button>
      <button
        type="button"
        class="btn flex-1"
        class:preset-filled-error-500={variant === 'danger'}
        class:preset-filled-primary-500={variant !== 'danger'}
        disabled={busy}
        aria-busy={busy}
        onclick={handleConfirm}
      >
        {busy ? 'Working…' : confirmLabel}
      </button>
    </div>
  </div>
</dialog>

<style>
  .confirm-dialog {
    padding: 0;
    margin: auto;
    border: none;
    background: transparent;
    max-width: min(24rem, calc(100vw - var(--space-8)));
    z-index: var(--z-modal);
  }

  .confirm-dialog::backdrop {
    background-color: var(--overlay-backdrop);
  }

  .confirm-dialog[open]::backdrop {
    animation: motion-backdrop-enter var(--motion-medium) var(--ease-out-quart) both;
  }

  .confirm-dialog-panel {
    padding: var(--space-6);
    margin: 0;
  }

  .confirm-dialog-extra {
    padding-top: var(--space-2);
  }

  .confirm-dialog-actions {
    padding-top: var(--space-2);
  }

  @media (prefers-reduced-motion: reduce) {
    .confirm-dialog[open]::backdrop {
      animation: none;
    }
  }
</style>
