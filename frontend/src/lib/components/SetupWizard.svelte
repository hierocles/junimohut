<script lang="ts">
  import * as API from '$lib/api';
  import type { Settings } from '$lib/api/client';
  import { formatUserError, settingsBrowseLabel, setupWelcomeTitle, setupWelcomeBody } from '$lib/copy';
  import setupHeroSvg from '$lib/assets/brand/setup-hero.svg?raw';
  import AppShellHeader from '$lib/components/AppShellHeader.svelte';

  interface Props {
    settings: Settings;
    oncomplete: () => void;
    onerror: (message: string) => void;
  }

  let { settings, oncomplete, onerror }: Props = $props();

  let gamePath = $state('');
  let smapiPath = $state('');
  let modsRoot = $state('');

  $effect(() => {
    gamePath = settings.gamePath ?? '';
    smapiPath = settings.smapiPath ?? '';
    modsRoot = settings.modsRoot ?? '';
  });
  let loading = $state(false);

  async function detect() {
    loading = true;
    try {
      const d = await API.DetectPaths();
      if (d.gamePath) gamePath = d.gamePath;
      if (d.smapiPath) smapiPath = d.smapiPath;
      if (d.modsRoot) modsRoot = d.modsRoot;
    } catch (e) {
      onerror(formatUserError(e));
    } finally {
      loading = false;
    }
  }

  async function browsePath(field: 'gamePath' | 'smapiPath' | 'modsRoot') {
    try {
      let picked = '';
      if (field === 'gamePath') picked = await API.BrowseGameFolder();
      else if (field === 'smapiPath') picked = await API.BrowseSMAPIPath();
      else picked = await API.BrowseModsRoot();
      if (picked) {
        if (field === 'gamePath') gamePath = picked;
        else if (field === 'smapiPath') smapiPath = picked;
        else modsRoot = picked;
      }
    } catch (e) {
      onerror(formatUserError(e));
    }
  }

  async function complete() {
    if (!gamePath || !modsRoot) {
      onerror('Enter your game folder and mod library to continue.');
      return;
    }
    loading = true;
    try {
      await API.CompleteSetup(gamePath, smapiPath, modsRoot);
      oncomplete();
    } catch (e) {
      onerror(formatUserError(e));
    } finally {
      loading = false;
    }
  }
</script>

<AppShellHeader />

<main class="flex-1 overflow-y-auto flex justify-center px-6 py-10">
  <div class="card layout-stack w-full max-w-xl setup-card" style="padding: var(--space-8);">
    <div class="setup-hero mb-4" aria-hidden="true">
      {@html setupHeroSvg}
    </div>
    <header class="layout-stack-sm">
      <h2 class="type-display text-surface-50">{setupWelcomeTitle}</h2>
      <p class="type-body type-meta type-prose">
        {setupWelcomeBody}
      </p>
    </header>

    <label class="label">
      <span class="label-text">Game folder</span>
      <div class="field-path-row">
        <input class="input type-mono" bind:value={gamePath} maxlength="512" placeholder="C:\Program Files (x86)\Steam\steamapps\common\Stardew Valley" />
        <button type="button" class="btn preset-tonal field-browse-btn" onclick={() => browsePath('gamePath')} disabled={loading}>
          {settingsBrowseLabel}
        </button>
      </div>
    </label>

    <label class="label">
      <span class="label-text">SMAPI launcher</span>
      <div class="field-path-row">
        <input class="input type-mono" bind:value={smapiPath} maxlength="512" placeholder="StardewModdingAPI.exe inside your game folder" />
        <button type="button" class="btn preset-tonal field-browse-btn" onclick={() => browsePath('smapiPath')} disabled={loading}>
          {settingsBrowseLabel}
        </button>
      </div>
    </label>

    <label class="label">
      <span class="label-text">Mod library</span>
      <div class="field-path-row">
        <input class="input type-mono" bind:value={modsRoot} maxlength="512" placeholder="AppData\JunimoHut\mod-library" />
        <button type="button" class="btn preset-tonal field-browse-btn" onclick={() => browsePath('modsRoot')} disabled={loading}>
          {settingsBrowseLabel}
        </button>
      </div>
      <span class="type-caption type-meta">Mod files are stored here. Enabled mods appear as links in {gamePath ? `${gamePath}\\Mods` : 'your game Mods folder'}.</span>
    </label>

    <div class="flex" style="gap: var(--space-3);">
      <button class="btn preset-tonal" onclick={detect} disabled={loading} aria-busy={loading}>Detect paths</button>
      <button class="btn preset-filled-primary-500" onclick={complete} disabled={loading} aria-busy={loading}>
        {loading ? 'Saving…' : 'Save and continue'}
      </button>
    </div>
  </div>
</main>

<style>
  .setup-hero {
    border-radius: var(--radius-card-cozy);
    background: var(--jh-illustration-bg);
    overflow: hidden;
  }

  .setup-hero :global(svg) {
    display: block;
    width: 100%;
    height: auto;
    max-height: 10rem;
  }

  .setup-card {
    border-radius: var(--radius-card-cozy);
  }
</style>
