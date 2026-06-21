<script lang="ts">
  import { onMount } from "svelte";
  import { Events } from "@wailsio/runtime";
  import * as API from "$lib/api";
  import JsonCodeEditor from "$lib/components/JsonCodeEditor.svelte";
  import ConfigEditorFileTree, {
    type ModJsonFileNode,
  } from "$lib/components/ConfigEditorFileTree.svelte";
  import { parseJsoncState } from "$lib/mods/jsonc";
  import {
    flattenFileTree,
    treeFocusKeyForPath,
  } from "$lib/mods/configEditorTree";
  import ConfirmDialog from "$lib/components/ConfirmDialog.svelte";
  import WindowControls from "$lib/components/WindowControls.svelte";
  import {
    formatUserError,
    configEditorDiscard,
    configEditorDiscardConfirmBody,
    configEditorDiscardConfirmTitle,
    configEditorInvalidJson,
    configEditorJsoncHint,
    configEditorLoadFailed,
    configEditorLoadFailedFor,
    configEditorLoadingMods,
    configEditorLoadingFile,
    configEditorLoadingFileAria,
    configEditorTitleFallback,
    configEditorMissingModId,
    configEditorNoModsWithJson,
    configEditorModMissing,
    configEditorNoJsonInMod,
    configEditorEmptyLibraryHint,
    configEditorSelectModHint,
    configEditorSaveAndSwitch,
    configEditorOpenExternal,
    configEditorParseError,
    configEditorProfileBanner,
    configEditorSave,
    configEditorSaved,
    configEditorSaveFailed,
    configEditorSaving,
    configEditorSearchModsPlaceholder,
    configEditorSidebarFilesHeading,
    configEditorSidebarModsHeading,
    configEditorUnsaved,
    configEditorUnsavedCloseBody,
    configEditorUnsavedCloseTitle,
    configEditorUnsavedFileSwitchBody,
    configEditorUnsavedFileSwitchTitle,
    configEditorUnsavedSwitchBody,
    configEditorUnsavedSwitchTitle,
    configEditorValidJson,
    configEditorWindowTitle,
    dialogCancelLabel,
  } from "$lib/copy";
  import { closeWindow } from "$lib/wails/windowApi";

  type ModConfigView = Awaited<ReturnType<typeof API.GetModConfigFile>>;
  type ModJsonSummary = Awaited<
    ReturnType<typeof API.ListModsWithJsonFiles>
  >[number];

  type PendingAction =
    | { kind: "close" }
    | { kind: "switch-mod"; modId: string; relPath?: string }
    | { kind: "switch-file"; relPath: string }
    | { kind: "discard" };

  function queryFromLocation() {
    const params = new URLSearchParams(window.location.search);
    return {
      modId: params.get("modId")?.trim() ?? "",
      relPath: params.get("file")?.trim() ?? "",
    };
  }

  function dirKeysForPath(relPath: string): string[] {
    const parts = relPath.split("/");
    if (parts.length <= 1) return [];
    const keys: string[] = [];
    for (let i = 0; i < parts.length - 1; i++) {
      keys.push(parts.slice(0, i + 1).join("/") + "/");
    }
    return keys;
  }

  function parseOpenModEvent(data: unknown): {
    modId: string;
    relPath: string;
  } {
    if (typeof data === "string") {
      return { modId: data.trim(), relPath: "" };
    }
    if (data && typeof data === "object") {
      const record = data as { modId?: string; relPath?: string };
      return {
        modId: String(record.modId ?? "").trim(),
        relPath: String(record.relPath ?? "").trim(),
      };
    }
    return { modId: "", relPath: "" };
  }

  const initial = queryFromLocation();

  let modId = $state(initial.modId);
  let activeRelPath = $state(initial.relPath);
  let modSummaries = $state<ModJsonSummary[]>([]);
  let modSearch = $state("");
  let fileTree = $state<ModJsonFileNode[]>([]);
  let expandedDirs = $state(new Set<string>());
  let loadingMods = $state(true);
  let loadingFile = $state(true);
  let loadError = $state<string | null>(null);
  let view = $state<ModConfigView | null>(null);
  let draft = $state("");
  let saved = $state("");
  let editorRevision = $state(0);
  let saving = $state(false);
  let saveFlash = $state(false);
  let footerError = $state<string | null>(null);

  let pendingAction = $state<PendingAction | null>(null);
  let confirmOpen = $state(false);
  let confirmBusy = $state(false);

  let modFocusIndex = $state(0);
  let fileFocusKey = $state<string | null>(null);
  let fileListEl = $state<HTMLDivElement | undefined>();

  const SIDEBAR_MODS_MIN = 0.28;
  const SIDEBAR_MODS_MAX = 0.72;
  let sidebarModsFr = $state(0.4);
  let sidebarSplitResize = $state<{
    startY: number;
    startFr: number;
    totalPx: number;
  } | null>(null);

  const filteredMods = $derived.by(() => {
    const q = modSearch.trim().toLowerCase();
    if (!q) return modSummaries;
    return modSummaries.filter(
      (m) =>
        m.modName.toLowerCase().includes(q) ||
        m.folderPath.toLowerCase().includes(q),
    );
  });

  const jsonState = $derived(parseJsoncState(draft));
  const dirty = $derived(draft !== saved);
  const canSave = $derived(
    dirty && jsonState.valid && !saving && !loadError && !!activeRelPath,
  );
  const loading = $derived(loadingMods || loadingFile);

  const confirmVariant = $derived(
    pendingAction?.kind === "discard" || pendingAction?.kind === "close"
      ? "danger"
      : "default",
  );

  const flatFileRows = $derived(flattenFileTree(fileTree, expandedDirs));

  const showSaveAndSwitch = $derived(
    canSave &&
      (pendingAction?.kind === "switch-mod" ||
        pendingAction?.kind === "switch-file"),
  );

  const sidebarFilesFr = $derived(Math.max(0.15, 1 - sidebarModsFr));

  function modFileCountLabel(count: number): string {
    return count === 1 ? "1 file" : `${count} files`;
  }

  $effect(() => {
    void API.SetConfigEditorDirty(dirty);
  });

  $effect(() => {
    if (view?.modName) {
      document.title = configEditorWindowTitle(
        view.modName,
        activeRelPath.split("/").pop() ?? "config.json",
      );
    }
  });

  $effect(() => {
    const idx = filteredMods.findIndex((m) => m.modId === modId);
    if (idx >= 0) modFocusIndex = idx;
  });

  $effect(() => {
    if (activeRelPath) {
      fileFocusKey = treeFocusKeyForPath(activeRelPath);
    } else if (flatFileRows.length > 0) {
      fileFocusKey = flatFileRows[0].focusKey;
    } else {
      fileFocusKey = null;
    }
  });

  async function loadModSummaries() {
    loadingMods = true;
    try {
      modSummaries = (await API.ListModsWithJsonFiles()) ?? [];
    } finally {
      loadingMods = false;
    }
  }

  async function loadFileTree(id: string) {
    try {
      fileTree = (await API.ListModJsonFiles(id)) ?? [];
    } catch {
      fileTree = [];
    }
  }

  async function loadFile(id: string, relPath: string) {
    if (!id) {
      loadError = configEditorMissingModId;
      loadingFile = false;
      view = null;
      return;
    }
    loadingFile = true;
    loadError = null;
    footerError = null;
    try {
      const data = await API.GetModConfigFile(id, relPath);
      modId = id;
      activeRelPath = data.relPath;
      view = data;
      draft = data.content;
      saved = data.content;
      editorRevision += 1;
      expandedDirs = new Set([
        ...expandedDirs,
        ...dirKeysForPath(data.relPath),
      ]);
    } catch (error) {
      const message = formatUserError(error);
      loadError =
        message.toLowerCase().includes("not found") ||
        message.toLowerCase().includes("no longer")
          ? configEditorModMissing
          : message || configEditorLoadFailedFor(relPath);
      view = null;
    } finally {
      loadingFile = false;
    }
  }

  async function selectMod(id: string, relPath = "") {
    await loadFileTree(id);
    await loadFile(id, relPath);
  }

  function requestAction(action: PendingAction) {
    if (action.kind === "discard" && !dirty) {
      void loadFile(modId, activeRelPath);
      return;
    }
    if (
      (action.kind === "close" ||
        action.kind === "switch-mod" ||
        action.kind === "switch-file") &&
      dirty
    ) {
      pendingAction = action;
      confirmOpen = true;
      return;
    }
    if (action.kind === "discard" && dirty) {
      pendingAction = action;
      confirmOpen = true;
      return;
    }
    void executeAction(action);
  }

  async function executeAction(action: PendingAction) {
    if (action.kind === "close") {
      await API.SetConfigEditorDirty(false);
      await closeWindow();
      return;
    }
    if (action.kind === "discard") {
      footerError = null;
      await loadFile(modId, activeRelPath);
      return;
    }
    if (action.kind === "switch-mod") {
      await selectMod(action.modId, action.relPath ?? "");
      return;
    }
    if (action.kind === "switch-file") {
      await loadFile(modId, action.relPath);
    }
  }

  async function saveConfig(): Promise<boolean> {
    if (!canSave || !modId || !activeRelPath) return false;
    saving = true;
    footerError = null;
    try {
      await API.SaveModConfigFile(modId, activeRelPath, draft);
      saved = draft;
      saveFlash = true;
      setTimeout(() => {
        saveFlash = false;
      }, 1800);
      return true;
    } catch (error) {
      footerError = configEditorSaveFailed(formatUserError(error));
      return false;
    } finally {
      saving = false;
    }
  }

  async function saveAndSwitch() {
    if (!pendingAction || !canSave) return;
    confirmBusy = true;
    try {
      const ok = await saveConfig();
      if (!ok) return;
      const action = pendingAction;
      pendingAction = null;
      confirmOpen = false;
      await executeAction(action);
    } finally {
      confirmBusy = false;
    }
  }

  async function openExternal() {
    if (!modId || !activeRelPath) return;
    try {
      await API.OpenModConfigExternalFile(modId, activeRelPath);
    } catch (error) {
      footerError = formatUserError(error);
    }
  }

  function toggleDir(dirKey: string) {
    const next = new Set(expandedDirs);
    if (next.has(dirKey)) next.delete(dirKey);
    else next.add(dirKey);
    expandedDirs = next;
  }

  $effect(() => {
    if (filteredMods.length === 0) {
      modFocusIndex = 0;
      return;
    }
    if (modFocusIndex >= filteredMods.length) {
      modFocusIndex = filteredMods.length - 1;
    }
  });

  function focusFileTreeItem() {
    const key = fileFocusKey;
    if (!key) return;
    fileListEl
      ?.querySelector<HTMLElement>(`[data-focus-key="${CSS.escape(key)}"]`)
      ?.focus();
  }

  function onFileListFocus(e: FocusEvent) {
    if (e.target === fileListEl) focusFileTreeItem();
  }

  function scrollFileRowIntoView() {
    const key = fileFocusKey;
    if (!key) return;
    queueMicrotask(() => {
      fileListEl
        ?.querySelector<HTMLElement>(`[data-focus-key="${CSS.escape(key)}"]`)
        ?.scrollIntoView({ block: "nearest" });
    });
  }

  function scrollModIntoView(index: number) {
    const mod = filteredMods[index];
    if (mod) document.getElementById(`config-mod-${mod.modId}`)?.scrollIntoView({ block: "nearest" });
  }

  function onModListKeydown(e: KeyboardEvent) {
    if (filteredMods.length === 0) return;
    if (e.key === "ArrowDown") {
      e.preventDefault();
      modFocusIndex = Math.min(filteredMods.length - 1, modFocusIndex + 1);
      scrollModIntoView(modFocusIndex);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      modFocusIndex = Math.max(0, modFocusIndex - 1);
      scrollModIntoView(modFocusIndex);
    } else if (e.key === "Home") {
      e.preventDefault();
      modFocusIndex = 0;
      scrollModIntoView(0);
    } else if (e.key === "End") {
      e.preventDefault();
      modFocusIndex = filteredMods.length - 1;
      scrollModIntoView(modFocusIndex);
    } else if (e.key === "Enter") {
      e.preventDefault();
      const mod = filteredMods[modFocusIndex];
      if (mod && mod.modId !== modId) {
        requestAction({ kind: "switch-mod", modId: mod.modId });
      }
    }
  }

  function onFileListKeydown(e: KeyboardEvent) {
    const rows = flatFileRows;
    if (!rows.length) return;
    let idx = rows.findIndex((row) => row.focusKey === fileFocusKey);
    if (idx < 0) idx = 0;

    if (e.key === "ArrowDown") {
      e.preventDefault();
      idx = Math.min(rows.length - 1, idx + 1);
      fileFocusKey = rows[idx].focusKey;
      scrollFileRowIntoView();
      focusFileTreeItem();
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      idx = Math.max(0, idx - 1);
      fileFocusKey = rows[idx].focusKey;
      scrollFileRowIntoView();
      focusFileTreeItem();
    } else if (e.key === "ArrowRight") {
      const row = rows[idx];
      if (row?.type === "dir" && !row.expanded) {
        e.preventDefault();
        toggleDir(row.dirKey);
      }
    } else if (e.key === "ArrowLeft") {
      const row = rows[idx];
      if (row?.type === "dir" && row.expanded) {
        e.preventDefault();
        toggleDir(row.dirKey);
      }
    } else if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      const row = rows[idx];
      if (!row) return;
      if (row.type === "dir") {
        toggleDir(row.dirKey);
      } else if (row.relPath !== activeRelPath) {
        requestAction({ kind: "switch-file", relPath: row.relPath });
      }
    }
  }

  function onSplitPointerDown(e: PointerEvent) {
    const sidebar = (e.currentTarget as HTMLElement).closest(
      ".config-editor-sidebar",
    ) as HTMLElement | null;
    if (!sidebar) return;
    sidebarSplitResize = {
      startY: e.clientY,
      startFr: sidebarModsFr,
      totalPx: sidebar.clientHeight,
    };
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
  }

  function onSplitPointerMove(e: PointerEvent) {
    if (!sidebarSplitResize) return;
    const delta = e.clientY - sidebarSplitResize.startY;
    const next =
      sidebarSplitResize.startFr + delta / sidebarSplitResize.totalPx;
    sidebarModsFr = Math.min(
      SIDEBAR_MODS_MAX,
      Math.max(SIDEBAR_MODS_MIN, next),
    );
  }

  function onSplitPointerUp(e: PointerEvent) {
    if (!sidebarSplitResize) return;
    sidebarSplitResize = null;
    try {
      (e.currentTarget as HTMLElement).releasePointerCapture(e.pointerId);
    } catch {
      // pointer already released
    }
  }

  function onKeydown(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key === "s") {
      e.preventDefault();
      void saveConfig();
    }
  }

  function confirmCopy(action: PendingAction | null): {
    title: string;
    message: string;
    confirmLabel: string;
  } {
    if (action?.kind === "switch-mod") {
      return {
        title: configEditorUnsavedSwitchTitle,
        message: configEditorUnsavedSwitchBody,
        confirmLabel: "Switch mod",
      };
    }
    if (action?.kind === "switch-file") {
      return {
        title: configEditorUnsavedFileSwitchTitle,
        message: configEditorUnsavedFileSwitchBody,
        confirmLabel: "Switch file",
      };
    }
    if (action?.kind === "discard") {
      return {
        title: configEditorDiscardConfirmTitle,
        message: configEditorDiscardConfirmBody,
        confirmLabel: configEditorDiscard,
      };
    }
    return {
      title: configEditorUnsavedCloseTitle,
      message: configEditorUnsavedCloseBody,
      confirmLabel: "Close",
    };
  }

  const confirmText = $derived(confirmCopy(pendingAction));

  onMount(() => {
    void (async () => {
      await loadModSummaries();
      if (modId) await selectMod(modId, initial.relPath);
      else loadingFile = false;
    })();

    Events.On("config-editor-open-mod", (ev) => {
      const { modId: nextId, relPath } = parseOpenModEvent(ev.data);
      if (!nextId) return;
      if (nextId === modId && (!relPath || relPath === activeRelPath)) return;
      requestAction({ kind: "switch-mod", modId: nextId, relPath });
    });

    Events.On("config-editor-reload", () => {
      if (modId) void loadFile(modId, activeRelPath);
    });
  });
</script>

<svelte:window onkeydown={onKeydown} />

<div class="config-editor-shell app-shell">
  <header class="config-editor-header app-panel app-border">
    <div class="config-editor-drag" data-wails-drag="true">
      <div class="config-editor-title-block">
        {#if view}
          <h1 class="type-headline text-surface-50">{view.modName}</h1>
          <p class="type-mono type-meta text-surface-400">{view.displayPath}</p>
        {:else}
          <h1 class="type-headline text-surface-50">
            {configEditorTitleFallback}
          </h1>
        {/if}
      </div>
    </div>
    <WindowControls onclose={() => requestAction({ kind: "close" })} />
  </header>

  {#if view?.profileSpecificConfigs}
    <div class="config-editor-profile-banner" role="status">
      <span class="state-badge state-badge--info">
        {configEditorProfileBanner(view.profileName)}
      </span>
    </div>
  {/if}

  <div class="config-editor-meta app-panel app-border">
    <div class="config-editor-meta-start">
      {#if loadError}
        <span class="state-badge state-badge--error">{loadError}</span>
      {:else if loadingFile}
        <span class="state-badge state-badge--muted"
          >{configEditorLoadingFile}</span
        >
      {:else if jsonState.valid}
        <span class="state-badge state-badge--success"
          >{configEditorValidJson}</span
        >
      {:else}
        <span class="state-badge state-badge--error"
          >{configEditorInvalidJson}</span
        >
      {/if}
      {#if dirty && !loadingFile}
        <span class="state-badge state-badge--warning"
          >{configEditorUnsaved}</span
        >
      {/if}
      {#if saveFlash}
        <span class="state-badge state-badge--success motion-status-in"
          >{configEditorSaved}</span
        >
      {/if}
      {#if !loadError && !loadingFile}
        <span class="config-editor-jsonc-hint type-caption type-meta"
          >{configEditorJsoncHint}</span
        >
      {/if}
    </div>
    <button
      type="button"
      class="btn btn-sm preset-tonal"
      disabled={!modId || loadingFile || !!loadError}
      onclick={() => void openExternal()}
    >
      {configEditorOpenExternal}
    </button>
  </div>

  <div class="config-editor-body">
    <aside
      class="config-editor-sidebar app-panel app-border"
      class:is-resizing={sidebarSplitResize != null}
      style="grid-template-rows: minmax(0, {sidebarModsFr}fr) 4px minmax(0, {sidebarFilesFr}fr)"
    >
      <section class="config-editor-sidebar-section config-editor-sidebar-mods">
        <h2 class="type-label config-editor-sidebar-heading">
          {configEditorSidebarModsHeading}
        </h2>
        <input
          type="search"
          class="input input-sm w-full"
          placeholder={configEditorSearchModsPlaceholder}
          bind:value={modSearch}
        />
        <div
          class="config-editor-mod-list"
          role="listbox"
          aria-label="Mods with JSON files"
          tabindex={filteredMods.length > 0 ? 0 : -1}
          onkeydown={onModListKeydown}
        >
          {#if loadingMods}
            <p class="type-ui type-meta config-editor-sidebar-empty">
              {configEditorLoadingMods}
            </p>
          {:else if filteredMods.length === 0}
            <div class="config-editor-sidebar-empty layout-stack-sm">
              <p class="type-ui type-meta">{configEditorNoModsWithJson}</p>
              <p class="type-caption type-meta type-prose">
                {configEditorEmptyLibraryHint}
              </p>
            </div>
          {:else}
              {#each filteredMods as mod, i (mod.modId)}
                <button
                  id="config-mod-{mod.modId}"
                  type="button"
                  class="config-mod-row"
                  class:active={mod.modId === modId}
                  class:keyboard-focused={i === modFocusIndex}
                  role="option"
                  aria-selected={mod.modId === modId}
                  tabindex={-1}
                  onclick={() => {
                    if (mod.modId !== modId) {
                      requestAction({ kind: "switch-mod", modId: mod.modId });
                    }
                  }}
                >
                  <span class="config-mod-row-top">
                    <span class="config-mod-row-name truncate"
                      >{mod.modName}</span
                    >
                    <span class="config-mod-row-count type-caption"
                      >{modFileCountLabel(mod.jsonFileCount)}</span
                    >
                  </span>
                  <span class="config-mod-row-path type-mono truncate"
                    >{mod.folderPath}</span
                  >
                </button>
              {/each}
          {/if}
        </div>
      </section>

      <div
        class="config-editor-split-handle"
        role="separator"
        aria-orientation="horizontal"
        aria-label="Resize mods and files panels"
        onpointerdown={onSplitPointerDown}
        onpointermove={onSplitPointerMove}
        onpointerup={onSplitPointerUp}
        onpointercancel={onSplitPointerUp}
      ></div>

      <section
        class="config-editor-sidebar-section config-editor-sidebar-files"
      >
        <h2 class="type-label config-editor-sidebar-heading">
          {configEditorSidebarFilesHeading}
        </h2>
        <div
          class="config-editor-file-list"
          bind:this={fileListEl}
          role="group"
          aria-label="JSON file tree navigation"
          tabindex={flatFileRows.length > 0 ? 0 : -1}
          onfocus={onFileListFocus}
          onkeydown={onFileListKeydown}
        >
          {#if !modId}
            <p class="type-ui type-meta config-editor-sidebar-empty">
              {configEditorSelectModHint}
            </p>
          {:else if fileTree.length === 0}
            <p class="type-ui type-meta config-editor-sidebar-empty">
              {configEditorNoJsonInMod}
            </p>
          {:else}
            <ConfigEditorFileTree
              nodes={fileTree}
              {activeRelPath}
              {expandedDirs}
              focusedKey={fileFocusKey}
              onselectfile={(relPath) => {
                if (relPath === activeRelPath) return;
                requestAction({ kind: "switch-file", relPath });
              }}
              ontoggledir={toggleDir}
            />
          {/if}
        </div>
      </section>
    </aside>

    <main class="config-editor-main">
      {#if loadingFile}
        <div
          class="config-editor-loading"
          role="status"
          aria-busy="true"
          aria-label={configEditorLoadingFileAria}
        >
          <div class="config-editor-loading-skeleton" aria-hidden="true">
            <div class="config-editor-loading-line"></div>
            <div
              class="config-editor-loading-line config-editor-loading-line--wide"
            ></div>
            <div
              class="config-editor-loading-line config-editor-loading-line--medium"
            ></div>
            <div
              class="config-editor-loading-line config-editor-loading-line--short"
            ></div>
          </div>
        </div>
      {:else if loadError}
        <div class="config-editor-loading type-ui type-meta">{loadError}</div>
      {:else}
        <JsonCodeEditor
          value={draft}
          revision={editorRevision}
          onchange={(value) => {
            draft = value;
            footerError = null;
          }}
        />
      {/if}
    </main>
  </div>

  <footer class="config-editor-footer app-panel app-border">
    <div class="config-editor-footer-status type-ui type-meta">
      {#if footerError}
        <span class="text-error-400">{footerError}</span>
      {:else if !jsonState.valid && draft.trim()}
        <span class="text-error-400">
          {configEditorParseError(
            jsonState.line,
            jsonState.column,
            jsonState.message,
          )}
        </span>
      {/if}
    </div>
    <div class="config-editor-footer-actions">
      <button
        type="button"
        class="btn btn-sm preset-tonal"
        disabled={!dirty || saving || loadingFile}
        onclick={() => requestAction({ kind: "discard" })}
      >
        {configEditorDiscard}
      </button>
      <button
        type="button"
        class="btn btn-sm preset-filled-primary-500"
        disabled={!canSave}
        aria-busy={saving}
        onclick={() => void saveConfig()}
      >
        {saving ? configEditorSaving : configEditorSave}
      </button>
    </div>
  </footer>
</div>

<ConfirmDialog
  open={confirmOpen}
  title={confirmText.title}
  message={confirmText.message}
  confirmLabel={confirmText.confirmLabel}
  cancelLabel={dialogCancelLabel}
  variant={confirmVariant}
  extraLabel={showSaveAndSwitch ? configEditorSaveAndSwitch : undefined}
  extraDisabled={saving}
  onextra={saveAndSwitch}
  busy={confirmBusy}
  oncancel={() => {
    pendingAction = null;
    confirmOpen = false;
  }}
  onconfirm={async () => {
    if (!pendingAction) return;
    confirmBusy = true;
    try {
      const action = pendingAction;
      pendingAction = null;
      confirmOpen = false;
      await executeAction(action);
    } finally {
      confirmBusy = false;
    }
  }}
/>

<style>
  .config-editor-shell {
    display: grid;
    grid-template-rows: auto auto auto 1fr auto;
    height: 100dvh;
    overflow: hidden;
  }

  .config-editor-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--space-3);
    padding: var(--space-3) var(--space-3) var(--space-2);
    border-bottom-width: 1px;
    border-bottom-style: solid;
  }

  .config-editor-drag {
    flex: 1;
    min-width: 0;
    padding-left: var(--space-2);
  }

  .config-editor-title-block {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    min-width: 0;
  }

  .config-editor-title-block h1 {
    text-wrap: balance;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .config-editor-profile-banner {
    padding: var(--space-2) var(--space-4);
    background: color-mix(
      in oklch,
      var(--color-primary-500) 8%,
      var(--sdvm-panel)
    );
    border-bottom: 1px solid var(--sdvm-border);
  }

  .config-editor-meta {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    padding: var(--space-2) var(--space-4);
    border-bottom-width: 1px;
    border-bottom-style: solid;
  }

  .config-editor-meta-start {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
    min-width: 0;
    flex: 1;
  }

  .config-editor-jsonc-hint {
    color: var(--color-surface-500);
    flex-basis: 100%;
  }

  @media (min-width: 720px) {
    .config-editor-jsonc-hint {
      flex-basis: auto;
      margin-left: auto;
    }
  }

  .config-editor-body {
    display: grid;
    grid-template-columns: 240px 1fr;
    min-height: 0;
    overflow: hidden;
  }

  .config-editor-sidebar {
    display: grid;
    min-height: 0;
    border-right-width: 1px;
    border-right-style: solid;
  }

  .config-editor-sidebar.is-resizing {
    cursor: row-resize;
    user-select: none;
  }

  .config-editor-sidebar-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    min-height: 0;
    padding: var(--space-3);
  }

  .config-editor-sidebar-mods {
    min-height: 0;
  }

  .config-editor-split-handle {
    position: relative;
    flex-shrink: 0;
    height: 4px;
    margin: 0 calc(var(--space-3) * -1);
    cursor: row-resize;
    touch-action: none;
  }

  .config-editor-split-handle::after {
    content: "";
    position: absolute;
    left: var(--space-3);
    right: var(--space-3);
    top: 50%;
    height: 1px;
    transform: translateY(-50%);
    background: var(--sdvm-border);
    transition: background-color var(--motion-fast) var(--ease-out-quart);
  }

  .config-editor-split-handle:hover::after,
  .config-editor-sidebar.is-resizing .config-editor-split-handle::after {
    background: color-mix(
      in oklch,
      var(--color-primary-500) 45%,
      var(--sdvm-border)
    );
  }

  .config-editor-sidebar-heading {
    color: var(--color-surface-400);
  }

  .config-editor-mod-list,
  .config-editor-file-list {
    flex: 1;
    min-height: 0;
    overflow: auto;
    position: relative;
  }


  .config-editor-sidebar-empty {
    padding: var(--space-2);
    text-align: center;
  }

  .config-mod-row {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 0.125rem;
    width: 100%;
    height: 100%;
    padding: var(--space-2);
    border: none;
    border-radius: var(--radius-base);
    background: transparent;
    color: var(--color-surface-300);
    text-align: left;
    transition: background-color var(--motion-fast) var(--ease-out-quart);
  }

  .config-mod-row-top {
    display: flex;
    align-items: baseline;
    gap: var(--space-2);
    width: 100%;
    min-width: 0;
  }

  .config-mod-row:hover {
    background: var(--sdvm-raised);
    color: var(--color-surface-50);
  }

  .config-mod-row.active {
    background: color-mix(
      in oklch,
      var(--color-primary-500) 16%,
      var(--sdvm-raised)
    );
    color: var(--color-surface-50);
  }

  .config-mod-row.keyboard-focused,
  .config-mod-row:focus-visible,
  .config-editor-mod-list:focus-visible {
    outline: none;
  }

  .config-mod-row.keyboard-focused,
  .config-editor-mod-list:focus-visible .config-mod-row.keyboard-focused {
    box-shadow: inset 0 0 0 2px
      color-mix(in oklch, var(--color-primary-500) 55%, transparent);
  }

  .config-editor-mod-list:focus-visible {
    box-shadow: inset 0 0 0 2px
      color-mix(in oklch, var(--color-primary-500) 35%, transparent);
    border-radius: var(--radius-base);
  }

  .config-mod-row-name {
    font-size: var(--type-ui);
    font-weight: 600;
    flex: 1;
    min-width: 0;
  }

  .config-mod-row-count {
    flex-shrink: 0;
    color: var(--color-surface-500);
  }

  .config-mod-row.active .config-mod-row-count {
    color: var(--color-surface-400);
  }

  .config-mod-row-path {
    font-size: var(--type-caption);
    color: var(--color-surface-500);
    width: 100%;
  }

  .config-mod-row.active .config-mod-row-path {
    color: var(--color-surface-400);
  }

  .config-editor-main {
    min-height: 0;
    overflow: hidden;
  }

  .config-editor-loading {
    display: grid;
    place-items: start center;
    height: 100%;
    padding: var(--space-6);
  }

  .config-editor-loading-skeleton {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    width: min(100%, 42rem);
    padding: var(--space-4);
    border: 1px solid var(--sdvm-border);
    border-radius: var(--radius-base);
    background-color: color-mix(
      in oklab,
      var(--sdvm-panel) 92%,
      var(--color-surface-950)
    );
  }

  .config-editor-loading-line {
    height: 0.875rem;
    border-radius: var(--radius-base);
    background-color: color-mix(
      in oklab,
      var(--sdvm-raised) 70%,
      var(--color-surface-700)
    );
    animation: config-editor-skeleton-pulse 1.4s ease-in-out infinite;
  }

  .config-editor-loading-line--wide {
    width: 100%;
  }

  .config-editor-loading-line--medium {
    width: 82%;
  }

  .config-editor-loading-line--short {
    width: 48%;
  }

  @keyframes config-editor-skeleton-pulse {
    0%,
    100% {
      opacity: 0.55;
    }
    50% {
      opacity: 1;
    }
  }

  .config-editor-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
    padding: var(--space-3) var(--space-4);
    border-top-width: 1px;
    border-top-style: solid;
  }

  .config-editor-footer-status {
    flex: 1;
    min-width: 0;
  }

  .config-editor-footer-actions {
    display: flex;
    gap: var(--space-2);
    flex-shrink: 0;
  }

  @media (prefers-reduced-motion: reduce) {
    .config-mod-row,
    .config-editor-split-handle::after {
      transition: none;
    }

    .config-editor-loading-line {
      animation: none;
      opacity: 0.75;
    }
  }
</style>
