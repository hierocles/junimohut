<script lang="ts">
  import BrandMark from "$lib/components/BrandMark.svelte";
  import * as m from "$lib/paraglide/messages.js";

  interface Props {
    open: boolean;
    onclose: () => void;
  }

  let { open, onclose }: Props = $props();

  let dialogEl = $state<HTMLDialogElement | undefined>();
  let closeBtn = $state<HTMLButtonElement | undefined>();

  $effect(() => {
    const el = dialogEl;
    if (!el) return;
    if (open && !el.open) {
      el.showModal();
      queueMicrotask(() => closeBtn?.focus());
    } else if (!open && el.open) {
      el.close();
    }
  });

  function handleClose() {
    onclose();
  }

  function onDialogCancel(e: Event) {
    e.preventDefault();
    handleClose();
  }
</script>

<dialog
  bind:this={dialogEl}
  class="about-dialog overlay-dialog"
  aria-labelledby="about-title"
  aria-describedby="about-desc"
  onclose={handleClose}
  oncancel={onDialogCancel}
>
  <div
    class="about-dialog-panel card app-panel border app-border layout-stack-sm motion-dialog-enter"
  >
    <div class="about-dialog-brand">
      <BrandMark />
      <h2 id="about-title" class="type-title text-surface-50 m-0">
        {m.about_dialog_title()}
      </h2>
    </div>
    <p id="about-desc" class="type-ui type-meta type-prose m-0">
      {m.about_dialog_tagline()}<br />
      {m.about_dialog_disclaimer()}
    </p>
    <div class="about-dialog-actions">
      <button
        bind:this={closeBtn}
        type="button"
        class="btn preset-filled-primary-500 w-full"
        onclick={handleClose}
      >
        {m.about_close_label()}
      </button>
    </div>
  </div>
</dialog>

<style>
  .about-dialog {
    padding: 0;
    margin: auto;
    border: none;
    background: transparent;
    max-width: min(22rem, calc(100vw - var(--space-8)));
    z-index: var(--z-modal);
  }

  .about-dialog::backdrop {
    background-color: var(--overlay-backdrop);
  }

  .about-dialog[open]::backdrop {
    animation: motion-backdrop-enter var(--motion-medium) var(--ease-out-quart)
      both;
  }

  .about-dialog-panel {
    padding: var(--space-6);
    margin: 0;
  }

  .about-dialog-brand {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .about-dialog-brand :global(.brand-mark svg) {
    width: 2rem;
    height: 2rem;
  }

  .about-dialog-actions {
    padding-top: var(--space-2);
  }

  @media (prefers-reduced-motion: reduce) {
    .about-dialog[open]::backdrop {
      animation: none;
    }
  }
</style>
