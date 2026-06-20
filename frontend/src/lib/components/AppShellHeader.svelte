<script lang="ts">
  import type { Snippet } from 'svelte';
  import BrandMark from '$lib/components/BrandMark.svelte';
  import WindowControls from '$lib/components/WindowControls.svelte';
  import { appVersionLabel, brandWordmark, brandWordmarkTitle } from '$lib/copy';
  import { onDragRegionDoubleClick } from '$lib/wails/windowApi';

  interface Props {
    onTitleClick?: () => void;
    toolbar?: Snippet;
  }

  let { onTitleClick, toolbar }: Props = $props();
</script>

<header class="app-header app-header--frameless border-b app-border shrink-0">
  <div
    class="app-header-chrome"
    ondblclick={onDragRegionDoubleClick}
    role="presentation"
  >
    <div class="app-header-brand">
      <BrandMark />
      <h1 class="toolbar-brand-title m-0">
        <button
          type="button"
          class="brand-wordmark type-title"
          onclick={onTitleClick}
          title={brandWordmarkTitle}
        >
          {brandWordmark}
        </button>
      </h1>
      <span class="brand-version-pill type-caption">{appVersionLabel()}</span>
    </div>
    <div class="app-header-chrome-fill wails-drag" aria-hidden="true"></div>
    <WindowControls />
  </div>

  {#if toolbar}
    <div class="app-header-toolbar">
      {@render toolbar()}
    </div>
  {/if}
</header>
