<script lang="ts">
  import { onMount } from 'svelte';
  import { Minus, Square, Copy, X } from '@lucide/svelte';
  import {
    windowMinimizeLabel,
    windowMaximizeLabel,
    windowRestoreLabel,
    windowCloseLabel,
  } from '$lib/copy';
  import {
    isWailsHost,
    minimiseWindow,
    toggleWindowMaximise,
    closeWindow,
    queryWindowMaximised,
    queryWindowFocused,
  } from '$lib/wails/windowApi';

  let maximised = $state(false);
  let focused = $state(true);

  async function syncWindowState() {
    maximised = await queryWindowMaximised();
    focused = await queryWindowFocused();
  }

  onMount(() => {
    if (!isWailsHost()) return;

    void syncWindowState();

    const onFocus = () => {
      focused = true;
      void syncWindowState();
    };
    const onBlur = () => {
      focused = false;
    };
    const onResize = () => {
      void syncWindowState();
    };

    window.addEventListener('focus', onFocus);
    window.addEventListener('blur', onBlur);
    window.addEventListener('resize', onResize);

    return () => {
      window.removeEventListener('focus', onFocus);
      window.removeEventListener('blur', onBlur);
      window.removeEventListener('resize', onResize);
    };
  });

  async function onMaximiseClick() {
    await toggleWindowMaximise();
    await syncWindowState();
  }
</script>

{#if isWailsHost()}
  <div
    class="window-controls-capsule"
    class:window-controls-capsule--unfocused={!focused}
    role="group"
    aria-label="Window controls"
  >
    <button
      type="button"
      class="window-controls-btn"
      aria-label={windowMinimizeLabel}
      onclick={() => void minimiseWindow()}
    >
      <Minus size={14} strokeWidth={2} aria-hidden="true" />
    </button>
    <button
      type="button"
      class="window-controls-btn"
      aria-label={maximised ? windowRestoreLabel : windowMaximizeLabel}
      onclick={() => void onMaximiseClick()}
    >
      {#if maximised}
        <Copy size={13} strokeWidth={2} aria-hidden="true" />
      {:else}
        <Square size={13} strokeWidth={2} aria-hidden="true" />
      {/if}
    </button>
    <button
      type="button"
      class="window-controls-btn window-controls-btn--close"
      aria-label={windowCloseLabel}
      onclick={() => void closeWindow()}
    >
      <X size={14} strokeWidth={2} aria-hidden="true" />
    </button>
  </div>
{/if}
