<script lang="ts">
  import { Pencil } from "@lucide/svelte";
  import type { Category, Mod } from "$lib/api/client";
  import { modStatusInfo } from "$lib/mods/modStatus";
  import {
    dependencyRowsForMod,
    dependentRowsForMod,
  } from "$lib/mods/dependencies";
  import {
    displayModName,
    hasUserCustomName,
    officialModName,
  } from "$lib/mods/names";
  import {
    dependenciesMissingSummary,
    dependentsSummary,
    dependentViewMod,
    dependencyDisabled,
    dependencyEnableMod,
    dependencyInstalled,
    dependencyLoadOrderLabel,
    dependencyNotInstalled,
    dependencyOpenNexus,
    dependencyOptionalAbsent,
    dependencySearchNexus,
    dependencyVersionTooLow,
    modClearDisplayName,
    modClearDisplayNameAria,
    modDisplayNameAria,
    modDisplayNameLabel,
    modContainsOverwritesLabel,
    modContainsOverwritesTooltip,
    modOfficialNameLabel,
    modRenameLabel,
    configEditorEditConfig,
  } from "$lib/copy";
  import { nexusModPageUrl, nexusSearchUrl } from "$lib/mods/dependencies";
  import { openExternalUrl } from "$lib/wails/openExternalUrl";

  type Tab = "general" | "dependencies" | "dependents" | "update";

  interface Props {
    selectedModId: string | null;
    mods: Mod[];
    /** Full unfiltered library — used for dep resolution so providers absent from
     *  the current search/filter are still recognised as installed. */
    libraryMods: Mod[];
    categories: Category[];
    onclose: () => void;
    ondownloadupdate: (mod: Mod) => Promise<void>;
    onenabledependency?: (modId: string) => void | Promise<void>;
    onselectmod?: (modId: string) => void;
    onsetcustomname?: (modId: string, name: string) => void | Promise<void>;
    oneditconfig?: (mod: Mod) => void | Promise<void>;
  }

  let {
    selectedModId,
    mods,
    libraryMods,
    categories,
    onclose,
    ondownloadupdate,
    onenabledependency,
    onselectmod,
    onsetcustomname,
    oneditconfig,
  }: Props = $props();

  const mod = $derived.by(() => {
    const id = selectedModId;
    if (!id) return null;
    return mods.find((m) => m.id === id) ?? null;
  });

  let activeTab = $state<Tab>("general");
  let downloading = $state(false);
  let copiedField = $state<string | null>(null);
  let editingDisplayName = $state(false);
  let displayNameDraft = $state("");
  let displayNameBusy = $state(false);
  let displayNameInput = $state<HTMLInputElement | undefined>();
  let focusDisplayNameOnOpen = $state(false);

  const categoryById = $derived(new Map(categories.map((c) => [c.id, c])));

  const modCategories = $derived(
    mod
      ? (mod.categoryIds ?? [])
          .map((id) => categoryById.get(id))
          .filter((c): c is Category => c != null)
      : [],
  );

  const dependencyRows = $derived(mod ? dependencyRowsForMod(mod, libraryMods) : []);
  const hasDependencies = $derived(dependencyRows.length > 0);
  const dependentRows = $derived(mod ? dependentRowsForMod(mod, libraryMods) : []);
  const dependencyIssueCount = $derived(
    mod?.missingDependencyCount ?? mod?.dependencyIssues?.length ?? 0,
  );
  const hasDependents = $derived(dependentRows.length > 0);
  const hasUpdateTab = $derived(
    mod != null &&
      (mod.updateStatus?.state === "update" ||
        mod.updateStatus?.state === "update_available" ||
        mod.updateStatus?.state === "incompatible" ||
        mod.updateStatus?.state === "unofficial" ||
        !!mod.updateStatus?.message?.trim() ||
        !!mod.updateStatus?.modPageUrl),
  );

  const canDownload = $derived(
    mod != null &&
      (mod.updateStatus?.state === "update" ||
        mod.updateStatus?.state === "update_available") &&
      (mod.manifest?.UpdateKeys?.some((k) => k.startsWith("Nexus:")) ?? false),
  );

  const modStatus = $derived(
    mod ? modStatusInfo(mod) : { text: "", badge: "" },
  );

  $effect(() => {
    selectedModId;
    activeTab = "general";
    editingDisplayName = false;
    displayNameDraft = "";
  });

  $effect(() => {
    if (!focusDisplayNameOnOpen || !mod) return;
    focusDisplayNameOnOpen = false;
    startDisplayNameEdit();
  });

  export function focusDisplayNameInput() {
    focusDisplayNameOnOpen = true;
  }

  function startDisplayNameEdit() {
    if (!mod) return;
    editingDisplayName = true;
    displayNameDraft = mod.customName?.trim() ?? "";
    queueMicrotask(() => {
      displayNameInput?.focus();
      displayNameInput?.select();
    });
  }

  function cancelDisplayNameEdit() {
    editingDisplayName = false;
    displayNameDraft = "";
  }

  async function submitDisplayNameEdit() {
    if (!mod || displayNameBusy || !onsetcustomname)
      return cancelDisplayNameEdit();
    const trimmed = displayNameDraft.trim();
    const current = mod.customName?.trim() ?? "";
    if (trimmed === current) return cancelDisplayNameEdit();
    displayNameBusy = true;
    try {
      await onsetcustomname(mod.id, trimmed);
    } finally {
      displayNameBusy = false;
      editingDisplayName = false;
      displayNameDraft = "";
    }
  }

  async function clearDisplayName() {
    if (!mod || displayNameBusy || !onsetcustomname || !mod.customName?.trim())
      return;
    displayNameBusy = true;
    try {
      await onsetcustomname(mod.id, "");
    } finally {
      displayNameBusy = false;
      editingDisplayName = false;
      displayNameDraft = "";
    }
  }

  function formatDate(ts: number): string {
    if (!ts) return "—";
    return new Date(ts * 1000).toLocaleString();
  }

  function dependencyStatusBadge(
    state:
      | "satisfied"
      | "missing"
      | "version_too_low"
      | "disabled"
      | "optional",
  ): {
    text: string;
    badge: string;
  } {
    if (state === "satisfied") {
      return {
        text: dependencyInstalled(),
        badge: "state-badge state-badge--success",
      };
    }
    if (state === "optional") {
      return {
        text: dependencyOptionalAbsent(),
        badge: "state-badge state-badge--muted",
      };
    }
    if (state === "version_too_low") {
      return {
        text: dependencyVersionTooLow(),
        badge: "state-badge state-badge--error",
      };
    }
    if (state === "disabled") {
      return {
        text: dependencyDisabled(),
        badge: "state-badge state-badge--error",
      };
    }
    return {
      text: dependencyNotInstalled(),
      badge: "state-badge state-badge--error",
    };
  }

  function openDependencySearch(uniqueID: string) {
    void openExternalUrl(nexusSearchUrl(uniqueID));
  }

  function openDependencyPage(nexusModId: string) {
    void openExternalUrl(nexusModPageUrl(nexusModId));
  }

  async function enableDependency(modId: string) {
    if (!onenabledependency) return;
    await onenabledependency(modId);
  }

  function goToDependenciesTab() {
    activeTab = "dependencies";
  }

  function goToDependentsTab() {
    activeTab = "dependents";
  }

  function selectDependentMod(modId: string) {
    onselectmod?.(modId);
  }

  async function copyValue(field: string, value: string) {
    try {
      await navigator.clipboard.writeText(value);
      copiedField = field;
      setTimeout(() => {
        if (copiedField === field) copiedField = null;
      }, 1500);
    } catch {
      /* clipboard unavailable */
    }
  }

  async function downloadUpdate() {
    if (!mod || downloading) return;
    downloading = true;
    try {
      await ondownloadupdate(mod);
    } finally {
      downloading = false;
    }
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === "Escape" && mod) {
      e.preventDefault();
      onclose();
    }
  }

  const tabs = $derived([
    { id: "general" as const, label: "General" },
    ...(hasDependencies
      ? [{ id: "dependencies" as const, label: "Dependencies" }]
      : []),
    ...(hasDependents
      ? [{ id: "dependents" as const, label: "Dependents" }]
      : []),
    ...(hasUpdateTab ? [{ id: "update" as const, label: "Update" }] : []),
  ]);
</script>

<svelte:window onkeydown={onKeydown} />

{#if mod}
  <aside
    class="detail-pane app-panel motion-panel-enter flex shrink-0 flex-col border-t app-border"
    aria-label="Mod details"
  >
    <div class="detail-tabbar flex items-center border-b app-border">
      <div class="detail-tablist" role="tablist">
        {#each tabs as tab (tab.id)}
          <button
            type="button"
            role="tab"
            class="detail-tab"
            class:active={activeTab === tab.id}
            aria-selected={activeTab === tab.id}
            onclick={() => (activeTab = tab.id)}
          >
            {tab.label}
          </button>
        {/each}
      </div>
      <button
        type="button"
        class="detail-close btn btn-sm preset-tonal shrink-0"
        aria-label="Close details pane"
        onclick={onclose}
      >
        ×
      </button>
    </div>

    <div
      class="detail-body type-ui"
      role="tabpanel"
      style="padding: var(--space-4);"
    >
      {#key activeTab}
        <div class="detail-tab-panel motion-fade-in">
          {#if activeTab === "general"}
            <div
              class="summary-row grid sm:grid-cols-2 lg:grid-cols-4"
              style="gap: var(--space-4); margin-bottom: var(--space-4);"
            >
              <div class="min-w-0">
                <span class="field-label type-label"
                  >{modOfficialNameLabel}</span
                >
                <p class="summary-name type-subhead truncate text-surface-50">
                  {officialModName(mod)}
                </p>
              </div>
              <div class="min-w-0">
                <span class="field-label type-label">{modDisplayNameLabel}</span
                >
                <div class="display-name-row flex min-w-0 items-center gap-1">
                  {#if editingDisplayName}
                    <input
                      bind:this={displayNameInput}
                      class="input input-sm display-name-input min-w-0 flex-1"
                      bind:value={displayNameDraft}
                      maxlength="200"
                      disabled={displayNameBusy}
                      placeholder={officialModName(mod)}
                      aria-label={modDisplayNameAria}
                      onblur={() => void submitDisplayNameEdit()}
                      onkeydown={(e) => {
                        if (e.key === "Enter") {
                          e.preventDefault();
                          void submitDisplayNameEdit();
                        }
                        if (e.key === "Escape") {
                          e.preventDefault();
                          cancelDisplayNameEdit();
                        }
                      }}
                    />
                  {:else}
                    <p
                      class="summary-name type-subhead min-w-0 flex-1 truncate text-surface-50"
                    >
                      {displayModName(mod)}
                    </p>
                    {#if onsetcustomname}
                      <button
                        type="button"
                        class="display-name-action"
                        title={modRenameLabel}
                        aria-label={modRenameLabel}
                        disabled={displayNameBusy}
                        onclick={() => startDisplayNameEdit()}
                      >
                        <Pencil size={13} aria-hidden="true" />
                      </button>
                      {#if hasUserCustomName(mod)}
                        <button
                          type="button"
                          class="display-name-action"
                          title={modClearDisplayName}
                          aria-label={modClearDisplayNameAria(
                            displayModName(mod),
                          )}
                          disabled={displayNameBusy}
                          onclick={() => void clearDisplayName()}
                        >
                          ×
                        </button>
                      {/if}
                    {/if}
                  {/if}
                </div>
              </div>
              <div class="min-w-0">
                <span class="field-label type-label">Version</span>
                <p class="text-surface-300">{mod.manifest?.Version || "—"}</p>
              </div>
              <div class="min-w-0 sm:col-span-2 lg:col-span-2">
                <span class="field-label type-label">Unique ID</span>
                <div class="flex min-w-0 items-center gap-1">
                  <p class="type-mono truncate text-surface-300">
                    {mod.manifest?.UniqueID || "—"}
                  </p>
                  {#if mod.manifest?.UniqueID}
                    <button
                      type="button"
                      class="copy-btn"
                      class:motion-copy-pop={copiedField === "uid"}
                      title="Copy Unique ID"
                      onclick={() => copyValue("uid", mod.manifest!.UniqueID)}
                    >
                      {copiedField === "uid" ? "✓" : "⎘"}
                    </button>
                  {/if}
                </div>
              </div>
              <div class="min-w-0 lg:col-span-2">
                <span class="field-label type-label">Folder</span>
                <div class="flex min-w-0 items-center gap-1">
                  <p class="truncate type-meta">{mod.folderPath}</p>
                  <button
                    type="button"
                    class="copy-btn"
                    class:motion-copy-pop={copiedField === "folder"}
                    title="Copy folder path"
                    onclick={() => copyValue("folder", mod.folderPath)}
                  >
                    {copiedField === "folder" ? "✓" : "⎘"}
                  </button>
                </div>
              </div>
              <div class="min-w-0 lg:col-span-2">
                <span class="field-label type-label">Path</span>
                <div class="flex min-w-0 items-center gap-1">
                  <p class="type-mono truncate text-surface-400">
                    {mod.absolutePath}
                  </p>
                  <button
                    type="button"
                    class="copy-btn"
                    class:motion-copy-pop={copiedField === "path"}
                    title="Copy absolute path"
                    onclick={() => copyValue("path", mod.absolutePath)}
                  >
                    {copiedField === "path" ? "✓" : "⎘"}
                  </button>
                </div>
              </div>
            </div>

            <hr class="detail-divider" />

            <div class="info-columns">
              <section class="info-col">
                <h3 class="col-heading type-label">Manifest</h3>
                <dl class="field-list">
                  <div class="field-row">
                    <dt>Author</dt>
                    <dd>{mod.manifest?.Author || "—"}</dd>
                  </div>
                  <div class="field-row">
                    <dt>Entry DLL</dt>
                    <dd class="type-mono">{mod.manifest?.EntryDll || "—"}</dd>
                  </div>
                  {#if mod.manifest?.ContentPackFor}
                    <div class="field-row">
                      <dt>Content pack for</dt>
                      <dd class="type-mono">
                        {mod.manifest.ContentPackFor.UniqueID}
                        {#if mod.manifest.ContentPackFor.MinimumVersion}
                          <span class="text-surface-500">
                            ≥ {mod.manifest.ContentPackFor.MinimumVersion}</span
                          >
                        {/if}
                      </dd>
                    </div>
                  {/if}
                  {#if mod.manifest?.Description?.trim()}
                    <div class="field-row field-row-block">
                      <dt>Description</dt>
                      <dd class="type-prose type-meta text-surface-300">
                        {mod.manifest.Description}
                      </dd>
                    </div>
                  {/if}
                </dl>
              </section>

              <section class="info-col">
                <h3 class="col-heading type-label">Status</h3>
                <dl class="field-list">
                  <div class="field-row">
                    <dt>Enabled</dt>
                    <dd>
                      {#if mod.enabled}
                        <span class="state-badge state-badge--success">Yes</span
                        >
                      {:else}
                        <span class="state-badge state-badge--muted">No</span>
                      {/if}
                    </dd>
                  </div>
                  <div class="field-row">
                    <dt>Update</dt>
                    <dd class="detail-status-badges">
                      <span class={modStatus.badge}>{modStatus.text}</span>
                      {#if mod.containsOverwrites}
                        <span
                          class="state-badge state-badge--patch"
                          title={modContainsOverwritesTooltip()}
                        >
                          {modContainsOverwritesLabel()}
                        </span>
                      {/if}
                    </dd>
                  </div>
                  {#if dependencyIssueCount > 0}
                    <div class="field-row">
                      <dt>Dependencies</dt>
                      <dd>
                        <button
                          type="button"
                          class="dep-summary-link state-badge state-badge--error"
                          onclick={goToDependenciesTab}
                        >
                          {dependenciesMissingSummary(dependencyIssueCount)}
                        </button>
                      </dd>
                    </div>
                  {/if}
                  {#if hasDependents}
                    <div class="field-row">
                      <dt>Dependents</dt>
                      <dd>
                        <button
                          type="button"
                          class="dep-summary-link state-badge state-badge--info"
                          onclick={goToDependentsTab}
                        >
                          {dependentsSummary(dependentRows.length)}
                        </button>
                      </dd>
                    </div>
                  {/if}
                  <div class="field-row">
                    <dt>Core mod</dt>
                    <dd>{mod.isCoreMod ? "Yes" : "No"}</dd>
                  </div>
                  <div class="field-row">
                    <dt>JSON files</dt>
                    <dd class="detail-config-cell">
                      {#if mod.hasJsonFiles}
                        <span class="state-badge state-badge--success">
                          {mod.jsonFileCount === 1
                            ? "1 file"
                            : `${mod.jsonFileCount} files`}
                        </span>
                        {#if oneditconfig}
                          <button
                            type="button"
                            class="btn btn-sm preset-tonal"
                            onclick={() => void oneditconfig(mod)}
                          >
                            {configEditorEditConfig}
                          </button>
                        {/if}
                      {:else}
                        <span class="state-badge state-badge--muted">None</span>
                      {/if}
                    </dd>
                  </div>
                  <div class="field-row">
                    <dt>Has config</dt>
                    <dd>{mod.hasConfig ? "Yes" : "No"}</dd>
                  </div>
                  {#if modCategories.length > 0}
                    <div class="field-row field-row-block">
                      <dt>Tags</dt>
                      <dd class="flex flex-wrap gap-1">
                        {#each modCategories as cat (cat.id)}
                          <span
                            class="category-tag chip-colored"
                            style:--chip-color={cat.color ||
                              "var(--color-primary-500)"}
                          >
                            <span
                              class="dot"
                              style:background={cat.color ||
                                "var(--color-primary-500)"}
                            ></span>
                            {cat.name}
                          </span>
                        {/each}
                      </dd>
                    </div>
                  {/if}
                </dl>
              </section>

              <section class="info-col">
                <h3 class="col-heading type-label">Group</h3>
                <dl class="field-list">
                  <div class="field-row">
                    <dt>Label</dt>
                    <dd>{mod.groupLabel || "—"}</dd>
                  </div>
                  <div class="field-row">
                    <dt>Key</dt>
                    <dd class="type-mono text-surface-400">
                      {mod.groupKey || "—"}
                    </dd>
                  </div>
                </dl>
              </section>

              <section class="info-col">
                <h3 class="col-heading type-label">Dates</h3>
                <dl class="field-list">
                  <div class="field-row">
                    <dt>Installed</dt>
                    <dd>{formatDate(mod.installTime)}</dd>
                  </div>
                  <div class="field-row">
                    <dt>Last updated</dt>
                    <dd>{formatDate(mod.lastUpdated)}</dd>
                  </div>
                </dl>
              </section>
            </div>
          {:else if activeTab === "dependencies" && dependencyRows.length}
            <ul class="space-y-2">
              {#each dependencyRows as row (row.uniqueID + (row.isContentPack ? ":cp" : ""))}
                {@const status = dependencyStatusBadge(row.state)}
                <li class="dep-row">
                  <div class="dep-row-main min-w-0">
                    <span class="type-mono text-surface-200"
                      >{row.uniqueID}</span
                    >
                    {#if row.minimumVersion}
                      <span class="text-surface-500"
                        >≥ {row.minimumVersion}</span
                      >
                    {/if}
                    {#if row.isContentPack}
                      <span class="type-caption type-meta">content pack</span>
                    {:else if row.isRequired}
                      <span class="type-caption type-meta">required</span>
                    {:else}
                      <span class="type-caption type-meta"
                        >{dependencyLoadOrderLabel()}</span
                      >
                    {/if}
                  </div>
                  <div class="dep-row-status flex flex-wrap items-center gap-2">
                    <span class={status.badge}>{status.text}</span>
                    {#if row.installedName && row.state !== "missing" && row.state !== "optional"}
                      <span class="type-meta text-surface-400">
                        {row.installedName}{#if row.installedVersion}
                          v{row.installedVersion}{/if}
                      </span>
                    {/if}
                    <div
                      class="dep-row-actions flex flex-wrap items-center gap-1"
                    >
                      {#if row.state === "disabled" && row.providerModId && onenabledependency}
                        <button
                          type="button"
                          class="btn btn-xs preset-tonal"
                          onclick={() => enableDependency(row.providerModId!)}
                        >
                          {dependencyEnableMod()}
                        </button>
                      {/if}
                      {#if row.nexusModId}
                        <button
                          type="button"
                          class="btn btn-xs preset-tonal"
                          onclick={() => openDependencyPage(row.nexusModId!)}
                        >
                          {dependencyOpenNexus()}
                        </button>
                      {:else if (row.state === "missing" || row.state === "version_too_low") && row.isRequired}
                        <button
                          type="button"
                          class="btn btn-xs preset-tonal"
                          onclick={() => openDependencySearch(row.uniqueID)}
                        >
                          {dependencySearchNexus()}
                        </button>
                      {/if}
                    </div>
                  </div>
                </li>
              {/each}
            </ul>
          {:else if activeTab === "dependents" && dependentRows.length}
            <ul class="space-y-2">
              {#each dependentRows as row (row.modId)}
                <li class="dep-row">
                  <div class="dep-row-main min-w-0">
                    <span class="text-surface-100 font-medium">{row.name}</span>
                    {#if row.uniqueID}
                      <span class="type-mono text-surface-400"
                        >{row.uniqueID}</span
                      >
                    {/if}
                    {#if row.version}
                      <span class="text-surface-500">v{row.version}</span>
                    {/if}
                    {#if row.minimumVersion}
                      <span class="text-surface-500"
                        >requires ≥ {row.minimumVersion}</span
                      >
                    {/if}
                    {#if row.isContentPack}
                      <span class="type-caption type-meta">content pack</span>
                    {:else if row.isRequired}
                      <span class="type-caption type-meta">required</span>
                    {:else}
                      <span class="type-caption type-meta"
                        >{dependencyLoadOrderLabel()}</span
                      >
                    {/if}
                  </div>
                  <div class="dep-row-status flex flex-wrap items-center gap-2">
                    {#if row.enabled}
                      <span class="state-badge state-badge--success"
                        >Enabled</span
                      >
                    {:else}
                      <span class="state-badge state-badge--muted"
                        >Disabled</span
                      >
                    {/if}
                    <div
                      class="dep-row-actions flex flex-wrap items-center gap-1"
                    >
                      {#if onselectmod}
                        <button
                          type="button"
                          class="btn btn-xs preset-tonal"
                          onclick={() => selectDependentMod(row.modId)}
                        >
                          {dependentViewMod()}
                        </button>
                      {/if}
                      {#if row.nexusModId}
                        <button
                          type="button"
                          class="btn btn-xs preset-tonal"
                          onclick={() => openDependencyPage(row.nexusModId!)}
                        >
                          {dependencyOpenNexus()}
                        </button>
                      {/if}
                    </div>
                  </div>
                </li>
              {/each}
            </ul>
          {:else if activeTab === "update"}
            <dl class="field-list max-w-2xl">
              <div class="field-row">
                <dt>State</dt>
                <dd><span class={modStatus.badge}>{modStatus.text}</span></dd>
              </div>
              {#if mod.manifest?.Version}
                <div class="field-row">
                  <dt>Installed version</dt>
                  <dd>{mod.manifest.Version}</dd>
                </div>
              {/if}
              {#if mod.updateStatus?.latestVersion}
                <div class="field-row">
                  <dt>Latest version</dt>
                  <dd class="state-update">{mod.updateStatus.latestVersion}</dd>
                </div>
              {/if}
              {#if mod.updateStatus?.message?.trim()}
                <div class="field-row field-row-block">
                  <dt>Note</dt>
                  <dd class="type-prose type-meta text-surface-300">
                    {mod.updateStatus.message}
                  </dd>
                </div>
              {/if}
              {#if mod.updateStatus?.modPageUrl}
                <div class="field-row">
                  <dt>Mod page</dt>
                  <dd>
                    <a
                      class="anchor"
                      href={mod.updateStatus.modPageUrl}
                      onclick={(event) => {
                        event.preventDefault();
                        void openExternalUrl(mod.updateStatus!.modPageUrl!);
                      }}
                    >
                      Open on Nexus Mods
                    </a>
                  </dd>
                </div>
              {/if}
            </dl>
            {#if canDownload}
              <button
                type="button"
                class="btn btn-sm preset-filled-primary-500 font-medium"
                style="margin-top: var(--space-3);"
                disabled={downloading}
                aria-busy={downloading}
                onclick={downloadUpdate}
              >
                {downloading ? "Downloading…" : "Download update"}
              </button>
            {/if}
          {/if}
        </div>
      {/key}
    </div>
  </aside>
{/if}

<style>
  .detail-pane {
    max-height: var(--detail-pane-max);
    overflow: hidden;
  }

  .detail-tabbar {
    flex-shrink: 0;
    gap: var(--space-2);
    padding-inline: var(--space-4);
    overflow: hidden;
  }

  .detail-tablist {
    display: flex;
    flex: 1;
    min-width: 0;
    gap: var(--space-1);
    overflow: hidden;
  }

  .detail-close {
    margin-block: var(--space-1);
  }

  .detail-divider {
    margin: 0 0 var(--space-4);
    border: 0;
    border-top: 1px solid var(--sdvm-border);
  }

  .detail-body {
    flex: 1 1 auto;
    min-height: 0;
    overflow-y: auto;
  }

  .detail-tab {
    flex-shrink: 0;
    padding: var(--space-2) var(--space-3);
    font-size: var(--type-meta);
    font-weight: var(--weight-medium);
    color: var(--color-surface-400);
    background: transparent;
    border: 0;
    border-bottom: 2px solid transparent;
    line-height: var(--leading-snug);
    white-space: nowrap;
    cursor: pointer;
    transition:
      color var(--motion-fast) var(--ease-out-quart),
      border-color var(--motion-fast) var(--ease-out-quart);
  }

  .detail-tab:hover {
    color: var(--color-surface-50);
  }

  .detail-tab.active {
    color: var(--color-surface-50);
    font-weight: var(--weight-semibold);
    border-bottom-color: var(--color-primary-500);
    margin-bottom: -1px;
  }

  .detail-tab:focus-visible {
    outline: 2px solid
      color-mix(in oklab, var(--color-primary-500) 50%, transparent);
    outline-offset: -2px;
  }

  .field-label {
    display: block;
    margin-bottom: var(--space-1);
  }

  .info-columns {
    display: grid;
    gap: var(--space-4) var(--space-6);
    grid-template-columns: repeat(auto-fit, minmax(11rem, 1fr));
  }

  .col-heading {
    margin-bottom: var(--space-2);
  }

  .field-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .field-row {
    display: grid;
    grid-template-columns: minmax(5.5rem, 7rem) 1fr;
    gap: var(--space-2);
    align-items: baseline;
  }

  .field-row-block {
    grid-template-columns: 1fr;
    gap: var(--space-1);
  }

  .field-row dt {
    font-size: var(--type-caption);
    color: var(--color-surface-400);
  }

  .field-row dd {
    color: var(--color-surface-300);
    min-width: 0;
  }

  .detail-status-badges {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
  }

  .detail-config-cell {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
  }

  .copy-btn {
    flex-shrink: 0;
    padding: var(--space-1) var(--space-2);
    font-size: var(--type-caption);
    line-height: 1;
    color: var(--color-surface-400);
    border-radius: var(--radius-base, 0.25rem);
  }

  .copy-btn:hover {
    color: var(--color-surface-50);
    background-color: color-mix(
      in oklab,
      var(--color-surface-800) 100%,
      transparent
    );
  }

  .copy-btn:focus-visible {
    outline: 2px solid
      color-mix(in oklab, var(--color-primary-500) 50%, transparent);
    outline-offset: 1px;
  }

  .display-name-row {
    min-height: 1.75rem;
  }

  .display-name-input {
    font-size: var(--type-subhead);
  }

  .display-name-action {
    display: inline-flex;
    flex-shrink: 0;
    align-items: center;
    justify-content: center;
    width: 1.5rem;
    height: 1.5rem;
    padding: 0;
    color: var(--color-surface-400);
    background: transparent;
    border: 0;
    border-radius: var(--radius-base, 0.25rem);
    cursor: pointer;
  }

  .display-name-action:hover:not(:disabled) {
    color: var(--color-surface-50);
    background-color: color-mix(
      in oklab,
      var(--color-surface-800) 100%,
      transparent
    );
  }

  .display-name-action:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .category-tag {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-1) var(--space-2);
    font-size: var(--type-caption);
    border-radius: var(--radius-base, 0.25rem);
  }

  .category-tag .dot {
    width: 0.375rem;
    height: 0.375rem;
    border-radius: 9999px;
    flex-shrink: 0;
  }

  .dep-row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-3);
    background-color: color-mix(
      in oklab,
      var(--color-surface-800) 50%,
      transparent
    );
    border-radius: var(--radius-base, 0.25rem);
  }

  .dep-row-main {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
  }

  .dep-summary-link {
    cursor: pointer;
    border: 0;
    font: inherit;
  }

  .dep-summary-link:hover {
    filter: brightness(1.08);
  }

  @media (prefers-reduced-motion: reduce) {
    .detail-tab {
      transition: none;
    }
  }
</style>
