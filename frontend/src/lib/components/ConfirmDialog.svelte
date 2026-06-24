<script lang="ts">
  import type { Snippet } from "svelte";
  import * as m from "$lib/paraglide/messages.js";

  interface Props {
    open: boolean;
    title: string;
    message: string;
    confirmLabel?: string;
    cancelLabel?: string;
    variant?: "danger" | "default";
    busy?: boolean;
    extraLabel?: string;
    extraDisabled?: boolean;
    children?: Snippet;
    onconfirm: () => void | Promise<void>;
    onextra?: () => void | Promise<void>;
    oncancel: () => void;
  }

  let {
    open,
    title,
    message,
    confirmLabel = m.dialog_confirm_label(),
    cancelLabel = m.dialog_cancel_label(),
    variant = "default",
    busy = false,
    extraLabel,
    extraDisabled = false,
    children,
    onconfirm,
    onextra,
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
  async function handleExtra() {
    if (busy || extraDisabled || !onextra) return;
    await onextra();
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
  <div
    class="confirm-dialog-panel card app-panel border app-border layout-stack-sm motion-dialog-enter"
  >
    <h2 id="confirm-title" class="type-title text-surface-50">{title}</h2>
    <p id="confirm-message" class="type-ui type-meta type-prose">{message}</p>
    {#if children}
      <div class="confirm-dialog-extra layout-stack-sm">
        {@render children()}
      </div>
    {/if}
    <div
      class="confirm-dialog-actions"
      class:confirm-dialog-actions--triple={!!extraLabel}
    >
      <button
        bind:this={cancelBtn}
        type="button"
        class="btn preset-tonal"
        disabled={busy}
        onclick={handleCancel}
      >
        {cancelLabel}
      </button>
      {#if extraLabel}
        <button
          type="button"
          class="btn preset-filled-primary-500"
          disabled={busy || extraDisabled}
          aria-busy={busy}
          onclick={handleExtra}
        >
          {busy ? m.dialog_working_label() : extraLabel}
        </button>
      {/if}
      <button
        type="button"
        class="btn"
        class:preset-filled-error-500={variant === "danger"}
        class:preset-tonal={variant !== "danger"}
        disabled={busy}
        aria-busy={busy}
        onclick={handleConfirm}
      >
        {busy ? m.dialog_working_label() : confirmLabel}
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
    animation: motion-backdrop-enter var(--motion-medium) var(--ease-out-quart)
      both;
  }

  .confirm-dialog-panel {
    padding: var(--space-6);
    margin: 0;
  }

  .confirm-dialog-extra {
    padding-top: var(--space-2);
  }

  .confirm-dialog-actions {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-2);
    padding-top: var(--space-2);
  }

  .confirm-dialog-actions--triple {
    grid-template-columns: 1fr;
  }

  @media (prefers-reduced-motion: reduce) {
    .confirm-dialog[open]::backdrop {
      animation: none;
    }
  }
</style>
