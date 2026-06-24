<script lang="ts">
  import { X } from "@lucide/svelte";
  import * as API from "$lib/api";
  import { applyDocumentTheme } from "$lib/themes/applyDocumentTheme";
  import type { Settings } from "$lib/api/client";
  import * as m from "$lib/paraglide/messages.js";
  import { formatUserError } from "$lib/errors/formatUserError";
  import {
    settingsHideDisabledOptions,
    themeDropdownOptions,
  } from "$lib/i18n/helpers";
  import DropdownList from "$lib/components/DropdownList.svelte";

  interface Props {
    drawerOpen?: boolean;
    settings: Settings;
    nexusConnected: boolean;
    onclose: () => void;
    onsave: (s: Settings) => void | Promise<void>;
    onnexus: (key: string) => void | Promise<void>;
    oninstallsmapi: () => void | Promise<void>;
    onregisternxm: () => void | Promise<void>;
    onerror: (message: string) => void;
  }

  let {
    drawerOpen = $bindable(false),
    settings,
    nexusConnected,
    onclose,
    onsave,
    onnexus,
    oninstallsmapi,
    onregisternxm,
    onerror,
  }: Props = $props();

  type PathBaseline = Pick<Settings, "gamePath" | "smapiPath" | "modsRoot">;

  function cloneSettingsDraft(source: Settings): Settings {
    return {
      ...source,
      gamePath: source.gamePath ?? "",
      smapiPath: source.smapiPath ?? "",
      modsRoot: source.modsRoot ?? "",
      theme: source.theme ?? "stardew-dark",
      hideDisabledFilter: source.hideDisabledFilter ?? "none",
      ignoreHiddenFolders: source.ignoreHiddenFolders ?? true,
      profileSpecificConfigs: source.profileSpecificConfigs ?? false,
      autoEnableOnInstall: source.autoEnableOnInstall ?? true,
      alwaysAskDeleteOnUpdate: source.alwaysAskDeleteOnUpdate ?? false,
      showInstallSummary: source.showInstallSummary ?? true,
    };
  }

  const emptyDraft = (): Settings =>
    cloneSettingsDraft({
      gamePath: "",
      smapiPath: "",
      modsRoot: "",
    } as Settings);

  let draft = $state<Settings>(emptyDraft());
  let pathBaseline = $state<PathBaseline>({
    gamePath: "",
    smapiPath: "",
    modsRoot: "",
  });
  let nexusKey = $state("");
  let saving = $state(false);
  let nexusBusy = $state(false);
  let smapiBusy = $state(false);
  let nxmBusy = $state(false);

  const pathsDirty = $derived(
    draft.gamePath !== pathBaseline.gamePath ||
      draft.smapiPath !== pathBaseline.smapiPath ||
      draft.modsRoot !== pathBaseline.modsRoot,
  );

  let lastOpen = false;
  let openedAt = 0;
  let drawerEl = $state<HTMLDivElement | null>(null);

  $effect(() => {
    if (!drawerEl) return;
    if (drawerOpen) {
      drawerEl.style.display = "flex";
      drawerEl.removeAttribute("inert");
      drawerEl.setAttribute("aria-hidden", "false");
    } else {
      drawerEl.style.display = "none";
      drawerEl.setAttribute("inert", "");
      drawerEl.setAttribute("aria-hidden", "true");
    }
  });

  $effect.pre(() => {
    if (drawerOpen && !lastOpen) {
      openedAt = performance.now();
      draft = cloneSettingsDraft(settings);
      pathBaseline = {
        gamePath: settings.gamePath ?? "",
        smapiPath: settings.smapiPath ?? "",
        modsRoot: settings.modsRoot ?? "",
      };
    }
    lastOpen = drawerOpen;
  });

  $effect(() => {
    if (!drawerOpen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return;
      // Let an open dropdown consume Escape first.
      if (document.querySelector(".dropdown-layer")) return;
      closeDrawer();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  function closeDrawer() {
    if (pathsDirty) {
      draft.gamePath = pathBaseline.gamePath;
      draft.smapiPath = pathBaseline.smapiPath;
      draft.modsRoot = pathBaseline.modsRoot;
    }
    onclose();
    drawerOpen = false;
    if (drawerEl) {
      drawerEl.style.display = "none";
      drawerEl.setAttribute("inert", "");
      drawerEl.setAttribute("aria-hidden", "true");
    }
  }

  function onBackdropClick(e: MouseEvent) {
    if (e.button !== 0) return;
    // Ignore the opening click when the toolbar Settings control sits under the panel.
    if (performance.now() - openedAt < 500) return;
    closeDrawer();
  }

  async function persistDraft() {
    if (saving) return;
    saving = true;
    try {
      await onsave(draft);
    } finally {
      saving = false;
    }
  }

  async function savePaths() {
    if (saving || !pathsDirty) return;
    saving = true;
    try {
      await onsave(draft);
      pathBaseline = {
        gamePath: draft.gamePath,
        smapiPath: draft.smapiPath,
        modsRoot: draft.modsRoot,
      };
      closeDrawer();
    } catch {
      /* parent reports via status line */
    } finally {
      saving = false;
    }
  }

  async function onThemeChange() {
    applyDocumentTheme(draft.theme);
    try {
      await persistDraft();
    } catch {
      /* parent reports via status line */
    }
  }

  async function onAutoFieldChange() {
    try {
      await persistDraft();
    } catch {
      /* parent reports via status line */
    }
  }

  async function browsePath(field: keyof PathBaseline) {
    try {
      let picked = "";
      if (field === "gamePath") picked = await API.BrowseGameFolder();
      else if (field === "smapiPath") picked = await API.BrowseSMAPIPath();
      else picked = await API.BrowseModsRoot();
      if (picked) draft[field] = picked;
    } catch (e) {
      onerror(formatUserError(e));
    }
  }

  async function connectNexus() {
    if (nexusBusy) return;
    nexusBusy = true;
    try {
      await onnexus(nexusKey);
    } finally {
      nexusBusy = false;
    }
  }
</script>

<div
  bind:this={drawerEl}
  class="settings-drawer overlay-scrim overlay-scrim--drawer fixed inset-0 flex"
  style="display: none"
  aria-hidden="true"
  inert
>
  <button
    type="button"
    class="settings-drawer-backdrop overlay-backdrop-tint motion-backdrop-enter flex-1"
    onclick={onBackdropClick}
    aria-label="Close settings"
  ></button>

  <div
    class="settings-drawer-panel app-panel motion-drawer-enter border-l app-border"
  >
    <div class="settings-drawer-header overlay-panel-header">
      <h2 class="type-title text-surface-50 m-0">Settings</h2>
      <button
        type="button"
        class="btn btn-sm preset-tonal toolbar-icon-btn"
        onclick={closeDrawer}
        aria-label="Close settings"
      >
        <X size={14} />
      </button>
    </div>

    <div class="settings-drawer-body">
      <section class="settings-section">
        <h3 class="settings-section-title type-section-head">
          {m.settings_section_paths()}
        </h3>
        <p class="settings-section-hint type-caption type-meta type-prose">
          {m.settings_paths_hint()}
        </p>

        <label class="label">
          <span class="label-text">Game folder</span>
          <div class="field-path-row">
            <input
              class="input type-mono"
              bind:value={draft.gamePath}
              placeholder="Path to Stardew Valley"
              maxlength="512"
            />
            <button
              type="button"
              class="btn preset-tonal field-browse-btn"
              onclick={() => browsePath("gamePath")}
            >
              {m.settings_browse_label()}
            </button>
          </div>
        </label>

        <label class="label">
          <span class="label-text">SMAPI launcher</span>
          <div class="field-path-row">
            <input
              class="input type-mono"
              bind:value={draft.smapiPath}
              placeholder="StardewModdingAPI.exe"
              maxlength="512"
            />
            <button
              type="button"
              class="btn preset-tonal field-browse-btn"
              onclick={() => browsePath("smapiPath")}
            >
              {m.settings_browse_label()}
            </button>
          </div>
          <button
            type="button"
            class="btn preset-tonal field-secondary-action"
            disabled={smapiBusy}
            aria-busy={smapiBusy}
            onclick={async () => {
              if (smapiBusy) return;
              smapiBusy = true;
              try {
                await oninstallsmapi();
              } finally {
                smapiBusy = false;
              }
            }}
          >
            {smapiBusy ? m.settings_opening_smapi() : m.settings_install_smapi()}
          </button>
        </label>

        <label class="label">
          <span class="label-text">Mod library</span>
          <div class="field-path-row">
            <input
              class="input type-mono"
              bind:value={draft.modsRoot}
              placeholder="AppData\…\mod-library"
              maxlength="512"
            />
            <button
              type="button"
              class="btn preset-tonal field-browse-btn"
              onclick={() => browsePath("modsRoot")}
            >
              {m.settings_browse_label()}
            </button>
          </div>
        </label>
      </section>

      <section class="settings-section">
        <h3 class="settings-section-title type-section-head">
          {m.settings_section_library()}
        </h3>

        <label class="field-check-row">
          <input
            type="checkbox"
            class="checkbox"
            bind:checked={draft.ignoreHiddenFolders}
            onchange={onAutoFieldChange}
          />
          <span class="type-ui"
            >Skip hidden folders when scanning (SMAPI default)</span
          >
        </label>

        <label class="field-check-row">
          <input
            type="checkbox"
            class="checkbox"
            bind:checked={draft.profileSpecificConfigs}
            onchange={onAutoFieldChange}
          />
          <span class="type-ui">Keep separate config files per profile</span>
        </label>

        <label class="field-check-row">
          <input
            type="checkbox"
            class="checkbox"
            bind:checked={draft.autoEnableOnInstall}
            onchange={onAutoFieldChange}
          />
          <span class="type-ui">Enable new mods after install</span>
        </label>

        <label class="field-check-row">
          <input
            type="checkbox"
            class="checkbox"
            bind:checked={draft.alwaysAskDeleteOnUpdate}
            onchange={onAutoFieldChange}
          />
          <span class="type-ui"
            >Ask before deleting old files when updating a mod</span
          >
        </label>

        <label class="field-check-row">
          <input
            type="checkbox"
            class="checkbox"
            bind:checked={draft.showInstallSummary}
            onchange={onAutoFieldChange}
          />
          <span class="type-ui">{m.settings_show_install_summary()}</span>
        </label>

        <div class="label">
          <span class="label-text" id="settings-hide-disabled-label"
            >Disabled mods</span
          >
          <DropdownList
            layer="elevated"
            labelledById="settings-hide-disabled-label"
            bind:value={draft.hideDisabledFilter}
            options={[...settingsHideDisabledOptions]}
            onchange={onAutoFieldChange}
          />
        </div>
      </section>

      <section class="settings-section">
        <h3 class="settings-section-title type-section-head">
          {m.settings_section_appearance()}
        </h3>
        <div class="label">
          <span class="label-text" id="settings-theme-label">Theme</span>
          <DropdownList
            layer="elevated"
            labelledById="settings-theme-label"
            bind:value={draft.theme}
            options={themeDropdownOptions()}
            onchange={onThemeChange}
          />
        </div>
      </section>

      <section class="settings-section">
        <h3 class="settings-section-title type-section-head">
          {m.settings_section_nexus()}
        </h3>
        {#if nexusConnected}
          <p class="state-badge state-badge--success type-ui w-fit">
            Connected to Nexus Mods
          </p>
        {:else}
          <p class="type-caption type-meta settings-nexus-hint">
            {m.settings_nexus_hint()}
          </p>
          <input
            class="input"
            type="password"
            bind:value={nexusKey}
            placeholder="Nexus Mods API key"
            aria-label="Nexus Mods API key"
            autocomplete="off"
          />
          <button
            class="btn preset-tonal w-full"
            onclick={connectNexus}
            disabled={nexusBusy || !nexusKey.trim()}
            aria-busy={nexusBusy}
          >
            {nexusBusy ? "Connecting…" : "Connect Nexus Mods"}
          </button>
        {/if}
        <button
          class="btn preset-tonal w-full"
          disabled={nxmBusy}
          aria-busy={nxmBusy}
          onclick={async () => {
            if (nxmBusy) return;
            nxmBusy = true;
            try {
              await onregisternxm();
            } finally {
              nxmBusy = false;
            }
          }}
        >
          {nxmBusy ? "Registering…" : "Register nxm:// links (Windows)"}
        </button>
      </section>
    </div>

    <div class="settings-drawer-footer app-border">
      {#if pathsDirty}
        <p class="settings-footer-meta type-caption type-meta">
          {m.settings_unsaved_paths()}
        </p>
        <div class="settings-footer-actions">
          <button
            type="button"
            class="btn preset-filled-primary-500 flex-1"
            onclick={savePaths}
            disabled={saving}
            aria-busy={saving}
          >
            {saving ? "Saving…" : m.settings_save_paths()}
          </button>
          <button
            type="button"
            class="btn preset-tonal"
            onclick={closeDrawer}
            disabled={saving}>{m.dialog_cancel_label()}</button
          >
        </div>
      {:else}
        <div class="settings-footer-actions">
          <button
            type="button"
            class="btn preset-filled-primary-500 flex-1"
            onclick={closeDrawer}>{m.settings_done()}</button
          >
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .settings-drawer-panel {
    display: flex;
    flex-direction: column;
    width: min(100%, 30rem);
    max-height: 100%;
    overflow: hidden;
  }

  .settings-drawer-header {
    padding: var(--space-4) var(--space-5);
    border-bottom: 1px solid var(--sdvm-divider);
    flex-shrink: 0;
  }

  .settings-drawer-body {
    flex: 1 1 auto;
    min-height: 0;
    overflow-y: auto;
    padding: var(--space-5);
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
  }

  .settings-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .settings-section-title {
    margin: 0;
  }

  .settings-section-hint {
    margin: 0;
  }

  .settings-nexus-hint {
    margin: 0 0 var(--space-1);
  }

  .settings-drawer-footer {
    flex-shrink: 0;
    padding: var(--space-4) var(--space-5);
    border-top: 1px solid var(--sdvm-divider);
    background-color: var(--sdvm-panel);
  }

  .settings-footer-meta {
    margin: 0 0 var(--space-2);
  }

  .settings-footer-actions {
    display: flex;
    gap: var(--space-2);
  }
</style>
