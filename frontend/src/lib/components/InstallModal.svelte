<script lang="ts">
  import { ChevronDown } from "@lucide/svelte";
  import type {
    Category,
    InstallResult,
    InstallOptions,
    Mod,
  } from "$lib/api/client";
  import { USE_MOCK_DATA } from "$lib/api/client";
  import * as API from "$lib/api/index";
  import {
    pathBasename,
    normalizeArchivePaths,
    installCompleteLine,
    installDependencyWarningTitle,
    installDependencyWarningBody,
    installAnywayLabel,
    installSuggestedTagsHint,
    installDisplayNameLegend,
    installDisplayNameHint,
    installDisplayNameOfficial,
    installDisplayNameFolder,
    installDisplayNamePreviewLabel,
    installRowBadgePatch,
    installRowBadgeBlocked,
    installAddMoreArchives,
    installNamingDisclosure,
    dependencyNotInstalled,
    dependencyVersionTooLow,
    dependencyDisabled,
    installOverwriteWarningBody,
    installOverwriteTargetLegend,
    installOverwriteTargetHint,
    installOverwriteMultiTargetHint,
    installOverwriteMatchSummary,
    installOverwriteSamplePathsLabel,
    installOverwriteExistingModHint,
    installOverwriteExistingModMergeIntro,
    installOverwriteSelectRequiredHint,
    installOverwriteConfirmLabel,
  } from "$lib/copy";
  import {
    INSTALL_MODAL_DROP_ID,
    pathsFromDataTransfer,
    pathsFromFileList,
    useNativeArchiveFileDrop,
  } from "$lib/wails/archiveFileDrop";
  import type { InstallDependencyPreview } from "$lib/mods/dependencies";
  import type { InstallOverwritePreview } from "../../../bindings/junimohut/internal/mods/models.js";

  export type InstallNamePreview = {
    archivePath: string;
    needsDisplayNameChoice?: boolean;
    mods: Array<{
      officialName: string;
      folderLabel: string;
      destFolder: string;
      uniqueID: string;
    }>;
  };

  interface Props {
    open: boolean;
    paths?: string[];
    updateTarget?: Mod | null;
    manualQueue?: boolean;
    alwaysAskDeleteOnUpdate?: boolean;
    showInstallSummary?: boolean;
    categories: Category[];
    modsRoot: string;
    suggestedTagIds?: string[];
    onclose: () => void;
    oninstall: (
      paths: string[],
      tagIds: string[],
      options: InstallOptions,
    ) => Promise<InstallResult[]>;
    onpreview?: (paths: string[]) => Promise<InstallDependencyPreview[]>;
  }

  let {
    open,
    paths = $bindable([]),
    updateTarget = null,
    manualQueue = true,
    alwaysAskDeleteOnUpdate = false,
    showInstallSummary = true,
    categories,
    modsRoot,
    suggestedTagIds = [],
    onclose,
    oninstall,
    onpreview,
  }: Props = $props();

  let dialogEl = $state<HTMLDialogElement | undefined>();
  let fileInput = $state<HTMLInputElement | undefined>();
  let browseBtn = $state<HTMLButtonElement | undefined>();
  let installing = $state(false);
  let previewBusy = $state(false);
  let dependencyConfirmOpen = $state(false);
  let dependencyPreviews = $state<InstallDependencyPreview[]>([]);
  let results = $state<InstallResult[] | null>(null);
  let dragOver = $state(false);
  let closing = $state(false);
  let selectedTagIds = $state(new Set<string>());
  let tagsTouched = $state(false);
  let installMode = $state<"replace" | "install">("install");
  let deleteOldFiles = $state(false);
  let nameDisplayMode = $state<"official" | "folder">("official");
  let namePreviewBusy = $state(false);
  let namePreviews = $state<InstallNamePreview[]>([]);
  let installedUseFolderNames = $state(false);
  let overwritePreviewBusy = $state(false);
  let overwritePreviews = $state<InstallOverwritePreview[]>([]);
  let overwriteTargets = $state<Record<string, string[]>>({});
  let expandedPath = $state<string | null>(null);
  let namingOpen = $state(false);
  let dropZoneExpanded = $state(false);

  const isUpdateFlow = $derived(updateTarget != null);
  const needsMergeConfirmation = $derived(
    !isUpdateFlow &&
      overwritePreviews.some((preview) => preview.state === "confirm"),
  );
  const installActionLabel = $derived(
    isUpdateFlow && installMode === "replace"
      ? "Update mod"
      : needsMergeConfirmation
        ? installOverwriteConfirmLabel()
        : "Install",
  );

  const sortedCategories = $derived(
    [...categories].sort(
      (a, b) => a.sortOrder - b.sortOrder || a.name.localeCompare(b.name),
    ),
  );

  const showResults = $derived(results != null && showInstallSummary);
  const hasBlockedOverwrite = $derived(
    overwritePreviews.some((preview) => preview.state === "blocked"),
  );
  const hasValidOverwriteSelection = $derived(
    overwritePreviews.every((preview) => {
      if (preview.state !== "confirm") return true;
      return (overwriteTargets[preview.archivePath]?.length ?? 0) > 0;
    }),
  );
  const canInstall = $derived(
    paths.length > 0 &&
      !installing &&
      !hasBlockedOverwrite &&
      hasValidOverwriteSelection,
  );
  const showFullDropZone = $derived(
    manualQueue && (paths.length === 0 || dropZoneExpanded),
  );

  const resultSummary = $derived.by(() => {
    if (!results) return { ok: 0, fail: 0 };
    const fail = results.filter((r) => r.error).length;
    return { ok: results.length - fail, fail };
  });

  const appliedSuggestedTagIds = $derived.by(() => {
    const known = new Set(categories.map((c) => c.id));
    return [...new Set(suggestedTagIds)].filter((id) => known.has(id));
  });

  const showSuggestedTagHint = $derived(
    appliedSuggestedTagIds.length > 0 &&
      !tagsTouched &&
      appliedSuggestedTagIds.every((id) => selectedTagIds.has(id)),
  );

  const flatNamePreviewMods = $derived(
    namePreviews.flatMap((preview) =>
      preview.mods.map((mod) => ({
        ...mod,
        archivePath: preview.archivePath,
      })),
    ),
  );

  const hasNameDisplayChoice = $derived(
    namePreviews.some((preview) => preview.needsDisplayNameChoice),
  );

  const namingModCount = $derived(
    namePreviews
      .filter((preview) => preview.needsDisplayNameChoice)
      .reduce((count, preview) => count + preview.mods.length, 0),
  );

  function overwriteForPath(path: string): InstallOverwritePreview | undefined {
    return overwritePreviews.find((preview) => preview.archivePath === path);
  }

  function previewDisplayName(
    mod: (typeof flatNamePreviewMods)[number],
  ): string {
    if (hasNameDisplayChoice && nameDisplayMode === "folder") {
      return mod.folderLabel;
    }
    return mod.officialName;
  }

  function resultDisplayName(result: InstallResult): string {
    if (result.name?.trim()) {
      return result.name;
    }
    const preview = flatNamePreviewMods.find(
      (m) =>
        result.folderPath === m.destFolder ||
        (!!result.modId && !!m.uniqueID && result.modId.includes(m.uniqueID)),
    );
    if (preview) {
      return installedUseFolderNames
        ? preview.folderLabel
        : preview.officialName;
    }
    return pathBasename(result.folderPath) || "Archive";
  }

  function applySuggestedTags() {
    if (tagsTouched || appliedSuggestedTagIds.length === 0) return;
    selectedTagIds = new Set(appliedSuggestedTagIds);
  }

  function isMultiTargetPreview(preview: InstallOverwritePreview): boolean {
    const candidates = preview.candidates ?? [];
    if (candidates.length <= 1) return false;
    const uniqueIds = new Set(
      candidates.map((c) => c.uniqueID).filter((id): id is string => !!id),
    );
    return uniqueIds.size > 1;
  }

  function previewHasExistingModMatch(
    preview: InstallOverwritePreview,
  ): boolean {
    return preview.candidates?.some((item) => item.uniqueID) ?? false;
  }

  function overwriteTargetHint(
    preview: InstallOverwritePreview,
    multiTarget: boolean,
  ): string {
    if (previewHasExistingModMatch(preview) && !multiTarget) {
      return installOverwriteExistingModHint();
    }
    if (multiTarget) {
      return installOverwriteMultiTargetHint();
    }
    return installOverwriteTargetHint();
  }

  function syncOverwriteTargets(previews: InstallOverwritePreview[]) {
    const next: Record<string, string[]> = { ...overwriteTargets };
    for (const preview of previews) {
      if (preview.state !== "confirm") continue;
      const current = next[preview.archivePath] ?? [];
      const candidates = preview.candidates ?? [];
      const valid = new Set(
        candidates.map((c) => c.folderPath).filter(Boolean),
      );
      const filtered = current.filter((folderPath) => valid.has(folderPath));
      if (filtered.length > 0) {
        next[preview.archivePath] = filtered;
      } else {
        delete next[preview.archivePath];
      }
    }
    for (const path of Object.keys(next)) {
      if (!previews.some((preview) => preview.archivePath === path)) {
        delete next[path];
      }
    }
    overwriteTargets = next;
  }

  function buildOverwriteTargetsForInstall(): Record<string, string[]> {
    const targets: Record<string, string[]> = {};
    for (const preview of overwritePreviews) {
      if (preview.state !== "confirm") continue;
      const selected = (overwriteTargets[preview.archivePath] ?? [])
        .map((target) => target.trim())
        .filter(Boolean);
      if (selected.length > 0) {
        targets[preview.archivePath] = selected;
      }
    }
    return targets;
  }

  async function refreshOverwritePreview() {
    if (paths.length === 0 || isUpdateFlow) {
      overwritePreviews = [];
      overwriteTargets = {};
      return;
    }
    overwritePreviewBusy = true;
    try {
      if (USE_MOCK_DATA) {
        overwritePreviews = [];
      } else {
        overwritePreviews =
          (await API.PreviewInstallOverwrites([...paths])) ?? [];
      }
      syncOverwriteTargets(overwritePreviews);
      if (
        overwritePreviews.some(
          (preview) =>
            preview.state === "confirm" &&
            (preview.candidates?.length ?? 0) > 0,
        ) &&
        expandedPath == null &&
        paths.length === 1
      ) {
        expandedPath = paths[0];
      }
    } catch {
      overwritePreviews = [];
      overwriteTargets = {};
    } finally {
      overwritePreviewBusy = false;
    }
  }

  async function proceedToDependencyCheckAndInstall() {
    if (onpreview) {
      previewBusy = true;
      try {
        const previews = (await onpreview([...paths])) ?? [];
        const withIssues = previews.filter((p) => (p.issues?.length ?? 0) > 0);
        if (withIssues.length > 0) {
          dependencyPreviews = withIssues;
          dependencyConfirmOpen = true;
          return;
        }
      } catch {
        /* proceed without blocking install */
      } finally {
        previewBusy = false;
      }
    }
    await runInstallConfirmed();
  }

  function setOverwriteTarget(archivePath: string, folderPath: string) {
    overwriteTargets = { ...overwriteTargets, [archivePath]: [folderPath] };
  }

  function toggleOverwriteTarget(archivePath: string, folderPath: string) {
    const current = [...(overwriteTargets[archivePath] ?? [])];
    const idx = current.indexOf(folderPath);
    if (idx >= 0) current.splice(idx, 1);
    else current.push(folderPath);
    overwriteTargets = { ...overwriteTargets, [archivePath]: current };
  }

  function isOverwriteTargetSelected(
    archivePath: string,
    folderPath: string,
  ): boolean {
    return (overwriteTargets[archivePath] ?? []).includes(folderPath);
  }

  function onHtml5Drop(e: DragEvent) {
    if (useNativeArchiveFileDrop) return;
    e.preventDefault();
    dragOver = false;
    addPaths(pathsFromDataTransfer(e.dataTransfer));
    dropZoneExpanded = false;
  }

  function onHtml5DragEnter(e: DragEvent) {
    if (useNativeArchiveFileDrop || !manualQueue) return;
    if (!e.dataTransfer?.types.includes("Files")) return;
    e.preventDefault();
    dragOver = true;
  }

  function onHtml5DragOver(e: DragEvent) {
    if (useNativeArchiveFileDrop || !manualQueue) return;
    if (!e.dataTransfer?.types.includes("Files")) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = "copy";
    dragOver = true;
  }

  function onHtml5DragLeave(e: DragEvent) {
    if (useNativeArchiveFileDrop) return;
    const related = e.relatedTarget as Node | null;
    const current = e.currentTarget as HTMLElement;
    if (related && current.contains(related)) return;
    dragOver = false;
  }

  function toggleRowExpand(path: string) {
    expandedPath = expandedPath === path ? null : path;
  }

  function requestClose() {
    if (installing || closing) return;
    closing = true;
    onclose();
  }

  function onDialogCancel(e: Event) {
    e.preventDefault();
    requestClose();
  }

  function addPaths(incoming: string[]) {
    const next = normalizeArchivePaths([...paths, ...incoming]);
    if (next.length !== paths.length || next.some((p, i) => p !== paths[i])) {
      paths = next;
    }
  }

  function removePath(path: string) {
    paths = paths.filter((p) => p !== path);
    if (expandedPath === path) expandedPath = null;
  }

  function clearQueue() {
    paths = [];
    expandedPath = null;
    dropZoneExpanded = false;
  }

  function clearTags() {
    tagsTouched = true;
    selectedTagIds = new Set();
  }

  function toggleTag(categoryId: string) {
    tagsTouched = true;
    const next = new Set(selectedTagIds);
    if (next.has(categoryId)) next.delete(categoryId);
    else next.add(categoryId);
    selectedTagIds = next;
  }

  function isTagSelected(categoryId: string): boolean {
    return selectedTagIds.has(categoryId);
  }

  function onBrowsePick(e: Event) {
    const input = e.currentTarget as HTMLInputElement;
    addPaths(pathsFromFileList(input.files ?? []));
    input.value = "";
    dropZoneExpanded = false;
  }

  async function browseArchives() {
    if (USE_MOCK_DATA) {
      fileInput?.click();
      return;
    }
    try {
      const selected = (await API.SelectArchives()) ?? [];
      if (selected.length > 0) {
        addPaths(selected);
        dropZoneExpanded = false;
      }
    } catch {
      // dialog cancelled or unavailable
    }
  }

  function onDrop(e: DragEvent) {
    onHtml5Drop(e);
  }

  async function runInstallConfirmed() {
    if (!canInstall) return;
    installing = true;
    try {
      const batch = [...paths];
      const options: InstallOptions = isUpdateFlow
        ? {
            mode: installMode,
            deleteOld: installMode === "replace" ? deleteOldFiles : undefined,
            useFolderDisplayNames:
              hasNameDisplayChoice && nameDisplayMode === "folder",
          }
        : {
            mode: "install",
            useFolderDisplayNames:
              hasNameDisplayChoice && nameDisplayMode === "folder",
            overwriteTargets: buildOverwriteTargetsForInstall(),
          };
      installedUseFolderNames = options.useFolderDisplayNames ?? false;
      const installResults = await oninstall(
        batch,
        [...selectedTagIds],
        options,
      );
      const failed = installResults.filter((r) => r.error).length;
      if (!showInstallSummary && failed === 0) {
        paths = [];
        dependencyConfirmOpen = false;
        dependencyPreviews = [];
        overwritePreviews = [];
        overwriteTargets = {};
        requestClose();
        return;
      }
      results = installResults;
      paths = [];
      dependencyConfirmOpen = false;
      dependencyPreviews = [];
      overwritePreviews = [];
      overwriteTargets = {};
    } catch {
      results = null;
    } finally {
      installing = false;
    }
  }

  function dependencyIssueLabel(state: string): string {
    if (state === "version_too_low") return dependencyVersionTooLow();
    if (state === "disabled") return dependencyDisabled();
    return dependencyNotInstalled();
  }

  async function handleInstall() {
    if (!canInstall || installing || previewBusy || overwritePreviewBusy)
      return;
    await proceedToDependencyCheckAndInstall();
  }

  function cancelDependencyConfirm() {
    dependencyConfirmOpen = false;
    dependencyPreviews = [];
  }

  async function refreshNamePreview() {
    if (paths.length === 0 || isUpdateFlow) {
      namePreviews = [];
      return;
    }
    namePreviewBusy = true;
    try {
      if (USE_MOCK_DATA) {
        const { getMockInstallNamePreview } = await import(
          "$lib/mock/designData"
        );
        namePreviews = getMockInstallNamePreview([...paths]);
      } else {
        namePreviews = ((await API.PreviewInstallNames([...paths])) ?? []).map(
          (p) => ({ ...p, mods: p.mods ?? [] }),
        );
      }
    } catch {
      namePreviews = [];
    } finally {
      namePreviewBusy = false;
    }
  }

  function installMore() {
    results = null;
    if (!manualQueue) return;
    queueMicrotask(() => browseBtn?.focus());
  }

  function rowHasPatch(path: string): boolean {
    const preview = overwriteForPath(path);
    return preview != null && preview.state !== undefined;
  }

  function rowIsBlocked(path: string): boolean {
    return overwriteForPath(path)?.state === "blocked";
  }

  $effect(() => {
    const el = dialogEl;
    if (!el) return;
    const shouldOpen = open;
    paths;
    if (shouldOpen) {
      closing = false;
      queueMicrotask(() => {
        if (!dialogEl || !open) return;
        if (!dialogEl.open) {
          dialogEl.showModal();
          if (manualQueue && paths.length === 0) browseBtn?.focus();
        }
      });
    } else if (el.open) {
      closing = true;
      el.close();
    }
  });

  $effect(() => {
    if (!open) {
      results = null;
      dragOver = false;
      selectedTagIds = new Set();
      tagsTouched = false;
      installMode = "install";
      deleteOldFiles = false;
      nameDisplayMode = "official";
      namePreviews = [];
      dependencyConfirmOpen = false;
      dependencyPreviews = [];
      overwritePreviews = [];
      overwriteTargets = {};
      expandedPath = null;
      namingOpen = false;
      dropZoneExpanded = false;
      return;
    }
    closing = false;
    if (updateTarget) {
      installMode = "replace";
    }
    applySuggestedTags();
  });

  $effect(() => {
    if (!open || isUpdateFlow) return;
    paths;
    void refreshNamePreview();
    void refreshOverwritePreview();
  });

  $effect(() => {
    if (!open || tagsTouched) return;
    appliedSuggestedTagIds;
    applySuggestedTags();
  });
</script>

<dialog
  bind:this={dialogEl}
  class="install-dialog overlay-dialog"
  aria-labelledby="install-title"
  onclose={() => {
    if (!installing && !closing) onclose();
  }}
  oncancel={onDialogCancel}
>
  <div
    class="install-panel card app-panel border app-border motion-dialog-enter"
  >
    {#if showResults && results}
      <header class="install-header">
        <div class="min-w-0">
          <h2 id="install-title" class="type-title text-surface-50">
            Install complete
          </h2>
        </div>
        <button
          type="button"
          class="install-close"
          aria-label="Close"
          onclick={requestClose}
        >
          ×
        </button>
      </header>

      <div
        class="install-results layout-stack-sm"
        role="status"
        aria-live="polite"
      >
        <p class="type-ui install-results-summary">
          {installCompleteLine(resultSummary.ok, resultSummary.fail)}
        </p>
        <ul class="result-list" role="list">
          {#each results as result, i (result.modId || result.name || i)}
            <li
              class="result-row delight-pop"
              class:result-row--error={!!result.error}
              style:animation-delay="{Math.min(i, 8) * 40}ms"
            >
              <span
                class="result-name type-ui truncate"
                title={resultDisplayName(result)}
              >
                {resultDisplayName(result)}
              </span>
              {#if result.error}
                <span
                  class="state-badge state-badge--error type-caption shrink-0"
                  >Failed</span
                >
                <p class="result-error type-caption type-meta type-prose">
                  {result.error}
                </p>
              {:else}
                <span
                  class="state-badge state-badge--success type-caption shrink-0 delight-pop"
                  >Installed</span
                >
              {/if}
            </li>
          {/each}
        </ul>
      </div>

      <footer class="install-footer install-footer--actions">
        {#if manualQueue}
          <button
            type="button"
            class="btn preset-tonal flex-1"
            onclick={installMore}>Install more</button
          >
        {/if}
        <button
          type="button"
          class="btn preset-filled-primary-500 flex-1"
          onclick={requestClose}>Done</button
        >
      </footer>
    {:else}
      <header class="install-header">
        <div class="min-w-0">
          <h2 id="install-title" class="type-title text-surface-50">
            {isUpdateFlow ? "Install update" : "Install mods"}
          </h2>
          <p
            class="type-caption type-meta install-dest truncate"
            title={modsRoot}
          >
            {#if isUpdateFlow && updateTarget}
              Updating <span class="type-data"
                >{updateTarget.manifest.Name}</span
              >
              in
              <span class="type-data">{modsRoot || "your mod library"}</span>
            {:else}
              Installs to <span class="type-data"
                >{modsRoot || "your mod library"}</span
              >
            {/if}
          </p>
        </div>
        <button
          type="button"
          class="install-close"
          aria-label="Close"
          disabled={installing}
          onclick={requestClose}
        >
          ×
        </button>
      </header>

      <div
        id={open && !showResults ? INSTALL_MODAL_DROP_ID : undefined}
        data-file-drop-target={open && !showResults && manualQueue
          ? true
          : undefined}
        class="install-body layout-stack-sm"
        class:install-body--drag={!useNativeArchiveFileDrop &&
          dragOver &&
          manualQueue}
        role="region"
        aria-label="Install queue and options"
        ondragenter={onHtml5DragEnter}
        ondragover={onHtml5DragOver}
        ondragleave={onHtml5DragLeave}
        ondrop={onDrop}
      >
        {#if isUpdateFlow && updateTarget}
          <div
            class="update-mode-compact"
            role="radiogroup"
            aria-label="Update method"
          >
            <label class="update-segment">
              <input
                type="radio"
                bind:group={installMode}
                value="replace"
                disabled={installing}
              />
              <span class="type-ui">Replace folder</span>
            </label>
            <label class="update-segment">
              <input
                type="radio"
                bind:group={installMode}
                value="install"
                disabled={installing}
              />
              <span class="type-ui">New folder</span>
            </label>
          </div>
          {#if installMode === "replace" && alwaysAskDeleteOnUpdate}
            <label class="update-delete-row">
              <input
                type="checkbox"
                bind:checked={deleteOldFiles}
                disabled={installing}
              />
              <span class="type-caption type-meta"
                >Delete old mod files before updating (keeps config.json)</span
              >
            </label>
          {/if}
        {/if}

        {#if manualQueue}
          {#if showFullDropZone}
            <div
              class="install-dropzone"
              class:install-dropzone--compact={paths.length > 0}
              role="presentation"
            >
              <p class="dropzone-title type-ui text-surface-100">
                {paths.length > 0
                  ? installAddMoreArchives
                  : "Drop mod archives here"}
              </p>
              {#if paths.length === 0}
                <p class="type-caption type-meta">.zip, .7z, or .rar</p>
              {/if}
              <button
                bind:this={browseBtn}
                type="button"
                class="btn btn-sm preset-tonal"
                disabled={installing}
                onclick={browseArchives}
              >
                Choose files…
              </button>
              <input
                bind:this={fileInput}
                type="file"
                class="sr-only"
                accept=".zip,.7z,.rar"
                multiple
                onchange={onBrowsePick}
              />
            </div>
            {#if paths.length > 0}
              <button
                type="button"
                class="anchor type-caption type-meta dropzone-collapse"
                disabled={installing}
                onclick={() => (dropZoneExpanded = false)}
              >
                Hide
              </button>
            {/if}
          {:else}
            <button
              type="button"
              class="btn btn-sm preset-tonal add-more-btn"
              disabled={installing}
              onclick={() => (dropZoneExpanded = true)}
            >
              {installAddMoreArchives}
            </button>
          {/if}
        {/if}

        {#if paths.length > 0}
          <div class="queue-section">
            <div class="queue-header">
              <span class="type-label"
                >{manualQueue ? "Ready to install" : "Downloaded mod"}</span
              >
              <div class="queue-header-actions">
                {#if overwritePreviewBusy && !isUpdateFlow}
                  <span class="type-caption type-meta">Checking patches…</span>
                {/if}
                {#if manualQueue}
                  <button
                    type="button"
                    class="anchor type-caption type-meta"
                    disabled={installing}
                    onclick={clearQueue}
                  >
                    Clear all
                  </button>
                {/if}
              </div>
            </div>
            <ul class="file-queue" role="list">
              {#each paths as path (path)}
                {@const preview = overwriteForPath(path)}
                {@const blocked = rowIsBlocked(path)}
                {@const patch = rowHasPatch(path)}
                <li class="queue-item">
                  <div
                    class="queue-row"
                    class:queue-row--expanded={expandedPath === path}
                  >
                    <span class="file-name type-ui truncate" title={path}
                      >{pathBasename(path)}</span
                    >
                    <div class="queue-row-badges">
                      {#if blocked}
                        <span
                          class="state-badge state-badge--error type-caption queue-badge"
                          >{installRowBadgeBlocked}</span
                        >
                      {:else if preview?.state === "confirm"}
                        <span
                          class="state-badge state-badge--warning type-caption queue-badge"
                          >{installRowBadgePatch}</span
                        >
                      {/if}
                    </div>
                    {#if patch && !isUpdateFlow}
                      <button
                        type="button"
                        class="queue-expand"
                        aria-expanded={expandedPath === path}
                        aria-label="{expandedPath === path
                          ? 'Hide'
                          : 'Show'} patch options for {pathBasename(path)}"
                        disabled={installing}
                        onclick={() => toggleRowExpand(path)}
                      >
                        <span
                          class="queue-expand-icon"
                          class:queue-expand-icon--open={expandedPath === path}
                        >
                          <ChevronDown size={14} />
                        </span>
                      </button>
                    {/if}
                    {#if manualQueue}
                      <button
                        type="button"
                        class="file-remove"
                        aria-label="Remove {pathBasename(path)}"
                        disabled={installing}
                        onclick={() => removePath(path)}
                      >
                        ×
                      </button>
                    {/if}
                  </div>
                  {#if expandedPath === path && preview}
                    <div class="queue-row-panel layout-stack-sm">
                      <p
                        class="type-caption type-meta type-prose queue-panel-intro"
                      >
                        {previewHasExistingModMatch(preview)
                          ? installOverwriteExistingModMergeIntro(
                              preview.candidates?.length ?? 0,
                            )
                          : installOverwriteWarningBody(preview.fileCount)}
                      </p>
                      {#if preview.state === "blocked"}
                        <p
                          class="type-caption type-meta type-prose overwrite-blocked"
                        >
                          {preview.blockReason}
                        </p>
                      {:else if preview.candidates?.length}
                        {@const multiTarget = isMultiTargetPreview(preview)}
                        {@const selectedTargets =
                          overwriteTargets[preview.archivePath] ?? []}
                        {@const selectedCandidate =
                          preview.candidates.find((item) =>
                            selectedTargets.includes(item.folderPath),
                          )}
                        <p class="type-caption type-meta type-prose">
                          {overwriteTargetHint(preview, multiTarget)}
                        </p>
                        {#if selectedTargets.length === 0}
                          <p
                            class="type-caption type-prose overwrite-select-required"
                          >
                            {installOverwriteSelectRequiredHint()}
                          </p>
                        {/if}
                        <span class="type-caption type-label"
                          >{installOverwriteTargetLegend()}</span
                        >
                        <ul class="overwrite-target-list" role="list">
                          {#each preview.candidates as candidate (candidate.folderPath + candidate.uniqueID)}
                            {@const selected = isOverwriteTargetSelected(
                              preview.archivePath,
                              candidate.folderPath,
                            )}
                            <li class="overwrite-target-item">
                              <label class="overwrite-target-option">
                                <input
                                  type={multiTarget ? "checkbox" : "radio"}
                                  name={multiTarget
                                    ? undefined
                                    : `overwrite-target-${preview.archivePath}`}
                                  checked={selected}
                                  disabled={installing || overwritePreviewBusy}
                                  onchange={() =>
                                    multiTarget
                                      ? toggleOverwriteTarget(
                                          preview.archivePath,
                                          candidate.folderPath,
                                        )
                                      : setOverwriteTarget(
                                          preview.archivePath,
                                          candidate.folderPath,
                                        )}
                                />
                                <span class="overwrite-target-copy">
                                  <span class="type-ui"
                                    >{candidate.modName}</span
                                  >
                                  <span
                                    class="type-caption type-meta truncate"
                                    title={candidate.folderPath}
                                  >
                                    {candidate.folderPath}
                                  </span>
                                  <span class="type-caption type-meta">
                                    {installOverwriteMatchSummary(
                                      candidate.matchedFiles,
                                      candidate.totalFiles,
                                    )}
                                  </span>
                                </span>
                              </label>
                            </li>
                          {/each}
                        </ul>
                        {#if selectedCandidate?.samplePaths?.length}
                          <details class="overwrite-sample-details">
                            <summary class="type-caption type-meta anchor">
                              {installOverwriteSamplePathsLabel()}
                            </summary>
                            <ul class="overwrite-sample-list" role="list">
                              {#each selectedCandidate.samplePaths as samplePath (samplePath)}
                                <li
                                  class="type-caption type-meta type-mono truncate"
                                  title={samplePath}
                                >
                                  {samplePath}
                                </li>
                              {/each}
                            </ul>
                          </details>
                        {/if}
                      {/if}
                    </div>
                  {/if}
                </li>
              {/each}
            </ul>
          </div>
        {:else if manualQueue}
          <p class="type-caption type-meta type-prose queue-empty">
            No files selected yet. Drop archives above or choose files from
            disk.
          </p>
        {/if}

        {#if !isUpdateFlow && paths.length > 0 && hasNameDisplayChoice}
          <div class="naming-disclosure">
            <button
              type="button"
              class="naming-disclosure-toggle"
              aria-expanded={namingOpen}
              disabled={installing}
              onclick={() => (namingOpen = !namingOpen)}
            >
              <span class="type-label"
                >{installNamingDisclosure(namingModCount)}</span
              >
              <span
                class="naming-disclosure-icon"
                class:naming-disclosure-icon--open={namingOpen}
              >
                <ChevronDown size={14} />
              </span>
            </button>
            {#if namingOpen}
              <div class="naming-disclosure-panel layout-stack-sm">
                <p class="type-caption type-meta type-prose name-display-hint">
                  {installDisplayNameHint}
                </p>
                <fieldset class="name-display-mode layout-stack-sm">
                  <legend class="sr-only">{installDisplayNameLegend}</legend>
                  <label class="name-display-option">
                    <input
                      type="radio"
                      bind:group={nameDisplayMode}
                      value="official"
                      disabled={installing || namePreviewBusy}
                    />
                    <span class="type-ui">{installDisplayNameOfficial}</span>
                  </label>
                  <label class="name-display-option">
                    <input
                      type="radio"
                      bind:group={nameDisplayMode}
                      value="folder"
                      disabled={installing || namePreviewBusy}
                    />
                    <span class="type-ui">{installDisplayNameFolder}</span>
                  </label>
                </fieldset>
                {#if namePreviewBusy}
                  <p class="type-caption type-meta">Checking mod names…</p>
                {:else if flatNamePreviewMods.length > 0}
                  <div class="name-preview-list" aria-live="polite">
                    <span class="type-caption type-label"
                      >{installDisplayNamePreviewLabel}</span
                    >
                    <ul class="name-preview-items" role="list">
                      {#each flatNamePreviewMods as mod (mod.uniqueID + mod.destFolder + mod.archivePath)}
                        <li class="name-preview-item">
                          <span
                            class="type-ui truncate"
                            title={previewDisplayName(mod)}
                          >
                            {previewDisplayName(mod)}
                          </span>
                          {#if mod.officialName !== previewDisplayName(mod)}
                            <span
                              class="type-caption type-meta truncate"
                              title={mod.officialName}
                            >
                              manifest: {mod.officialName}
                            </span>
                          {/if}
                        </li>
                      {/each}
                    </ul>
                  </div>
                {/if}
              </div>
            {/if}
          </div>
        {/if}

        <div class="tag-section layout-stack-sm">
          <div class="queue-header">
            <span class="type-label">Tags to apply</span>
            {#if selectedTagIds.size > 0}
              <button
                type="button"
                class="anchor type-caption type-meta"
                disabled={installing}
                onclick={clearTags}
              >
                Clear
              </button>
            {/if}
          </div>
          {#if sortedCategories.length === 0}
            <p class="type-caption type-meta type-prose tag-empty">
              Create tags in the sidebar to label mods during install.
            </p>
          {:else}
            <p class="type-caption type-meta type-prose tag-hint">
              Optional — applied to every mod installed from this batch.
            </p>
            {#if showSuggestedTagHint}
              <p
                class="type-caption type-meta type-prose tag-hint tag-hint--suggested"
              >
                {installSuggestedTagsHint(appliedSuggestedTagIds.length)}
              </p>
            {/if}
            <ul
              class="tag-select-list"
              role="listbox"
              aria-label="Tags to apply"
              aria-multiselectable="true"
            >
              {#each sortedCategories as cat (cat.id)}
                {@const selected = isTagSelected(cat.id)}
                <li>
                  <button
                    type="button"
                    class="tag-select-item"
                    class:tag-select-item--on={selected}
                    style:--chip-color={cat.color || "#6366f1"}
                    role="option"
                    aria-selected={selected}
                    disabled={installing}
                    onclick={() => toggleTag(cat.id)}
                  >
                    <span class="tag-select-dot" aria-hidden="true"></span>
                    <span class="tag-select-name type-ui truncate"
                      >{cat.name}</span
                    >
                    {#if selected}
                      <span
                        class="tag-select-check type-caption"
                        aria-hidden="true">✓</span
                      >
                    {/if}
                  </button>
                </li>
              {/each}
            </ul>
          {/if}
        </div>
      </div>

      <footer class="install-footer">
        {#if dependencyConfirmOpen}
          <div class="dependency-confirm layout-stack-sm">
            <p class="type-ui text-surface-100">
              {installDependencyWarningTitle()}
            </p>
            <p class="type-caption type-meta type-prose">
              {installDependencyWarningBody(dependencyPreviews.length)}
            </p>
            <ul class="dependency-warning-list" role="list">
              {#each dependencyPreviews as preview (preview.uniqueID + preview.archivePath)}
                <li class="dependency-warning-item">
                  <span class="type-ui text-surface-200">{preview.modName}</span
                  >
                  <ul class="dependency-warning-sublist" role="list">
                    {#each preview.issues as issue (issue.uniqueID + issue.state)}
                      <li class="type-caption type-meta">
                        <span class="type-mono">{issue.uniqueID}</span>
                        — {dependencyIssueLabel(issue.state)}
                      </li>
                    {/each}
                  </ul>
                </li>
              {/each}
            </ul>
          </div>
        {/if}

        <div
          class="install-footer-actions"
          class:install-footer-actions--confirm={dependencyConfirmOpen}
        >
          {#if dependencyConfirmOpen}
            <button
              type="button"
              class="btn preset-tonal flex-1"
              disabled={installing}
              onclick={cancelDependencyConfirm}
            >
              Cancel
            </button>
            <button
              type="button"
              class="btn preset-filled-primary-500 flex-1"
              disabled={installing}
              aria-busy={installing}
              onclick={runInstallConfirmed}
            >
              {installing ? "Installing…" : installAnywayLabel()}
            </button>
          {:else}
            <button
              type="button"
              class="btn preset-tonal flex-1"
              disabled={installing}
              onclick={requestClose}
            >
              Cancel
            </button>
            <button
              type="button"
              class="btn preset-filled-primary-500 flex-1"
              disabled={!canInstall || previewBusy || overwritePreviewBusy}
              aria-busy={installing || previewBusy || overwritePreviewBusy}
              onclick={handleInstall}
            >
              {installing
                ? installMode === "replace" && isUpdateFlow
                  ? "Updating…"
                  : "Installing…"
                : previewBusy || overwritePreviewBusy
                  ? "Checking…"
                  : paths.length === 1
                    ? `${installActionLabel} 1 mod`
                    : `${installActionLabel} ${paths.length} mods`}
            </button>
          {/if}
        </div>
      </footer>
    {/if}
  </div>
</dialog>

<style>
  .install-results {
    padding: 0 var(--space-6) var(--space-4);
  }

  .install-results-summary {
    text-wrap: pretty;
    margin: 0;
  }

  .install-dialog {
    padding: 0;
    margin: auto;
    border: none;
    background: transparent;
    width: min(36rem, calc(100vw - var(--space-8)));
    max-height: calc(100vh - var(--space-8));
    overflow: visible;
    z-index: var(--z-modal);
  }

  .install-dialog::backdrop {
    background-color: var(--overlay-backdrop);
  }

  .install-dialog[open]::backdrop {
    animation: motion-backdrop-enter var(--motion-medium) var(--ease-out-quart)
      both;
  }

  .install-panel {
    display: flex;
    flex-direction: column;
    padding: 0;
    margin: 0;
    max-height: calc(100vh - var(--space-8));
    min-height: 0;
    overflow: hidden;
  }

  .install-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--space-3);
    padding: var(--space-6) var(--space-6) var(--space-4);
    flex-shrink: 0;
  }

  .install-body {
    flex: 1;
    min-height: 8rem;
    overflow-y: auto;
    padding: 0 var(--space-6) var(--space-4);
  }

  .install-body[data-file-drop-target]:global(.file-drop-target-active) {
    outline: 1px dashed var(--color-primary-500);
    outline-offset: -2px;
    background-color: color-mix(
      in oklab,
      var(--color-primary-500) 6%,
      var(--sdvm-panel)
    );
  }

  .install-body--drag {
    outline: 1px dashed var(--color-primary-500);
    outline-offset: -2px;
    background-color: color-mix(
      in oklab,
      var(--color-primary-500) 6%,
      var(--sdvm-panel)
    );
  }

  .install-footer {
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    padding: var(--space-3) var(--space-6) var(--space-6);
    border-top: 1px solid var(--sdvm-border);
    background-color: var(--sdvm-panel);
  }

  .install-footer--actions {
    flex-direction: row;
    flex-wrap: wrap;
    border-top: 0;
    padding-top: 0;
  }

  .install-footer-actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
  }

  .install-footer-actions--confirm {
    flex-direction: column;
  }

  .install-dest {
    margin-top: var(--space-1);
  }

  .install-close {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 1.75rem;
    height: 1.75rem;
    padding: 0;
    border: 0;
    border-radius: var(--radius-base, 0.25rem);
    background: transparent;
    color: var(--color-surface-400);
    font-size: var(--type-subhead);
    line-height: 1;
    cursor: pointer;
    flex-shrink: 0;
  }

  .install-close:hover:not(:disabled),
  .install-close:focus-visible {
    color: var(--color-surface-100);
    background-color: var(--color-surface-800);
  }

  .install-close:focus-visible {
    outline: 2px solid var(--color-primary-400);
    outline-offset: 1px;
  }

  .install-close:disabled {
    opacity: 0.5;
    cursor: default;
  }

  .update-mode-compact {
    display: inline-flex;
    flex-wrap: wrap;
    gap: var(--space-1);
    padding: var(--space-1);
    border: 1px solid var(--sdvm-border);
    border-radius: var(--radius-base, 0.25rem);
    background-color: color-mix(
      in oklab,
      var(--color-surface-900) 35%,
      var(--sdvm-panel)
    );
    width: fit-content;
  }

  .update-segment {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-1) var(--space-3);
    border-radius: var(--radius-base, 0.25rem);
    cursor: pointer;
  }

  .update-segment:has(input:checked) {
    background-color: color-mix(
      in oklab,
      var(--color-primary-500) 14%,
      var(--sdvm-raised)
    );
  }

  .update-delete-row {
    display: flex;
    align-items: flex-start;
    gap: var(--space-2);
    cursor: pointer;
  }

  .install-dropzone {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    padding: var(--space-8) var(--space-4);
    border: 1px dashed
      color-mix(in oklab, var(--color-surface-600) 80%, transparent);
    border-radius: var(--radius-lg, 0.5rem);
    background-color: color-mix(
      in oklab,
      var(--color-surface-900) 40%,
      var(--sdvm-panel)
    );
    text-align: center;
    transition:
      border-color 150ms cubic-bezier(0.25, 1, 0.5, 1),
      background-color 150ms cubic-bezier(0.25, 1, 0.5, 1);
  }

  .install-dropzone--compact {
    padding: var(--space-4);
  }

  .dropzone-title {
    font-weight: var(--weight-medium);
    text-wrap: balance;
  }

  .dropzone-collapse,
  .add-more-btn {
    align-self: flex-start;
  }

  .queue-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
  }

  .queue-empty {
    margin: 0;
    text-wrap: pretty;
  }

  .file-queue {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    max-height: min(40vh, 16rem);
    margin: 0;
    padding: 0;
    list-style: none;
    overflow-y: auto;
  }

  .queue-item {
    display: flex;
    flex-direction: column;
    gap: 0;
    border: 1px solid var(--sdvm-border);
    border-radius: var(--radius-base, 0.25rem);
    background-color: color-mix(
      in oklab,
      var(--color-surface-900) 35%,
      var(--sdvm-panel)
    );
  }

  .queue-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto auto auto;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-3);
  }

  .queue-row--expanded {
    border-bottom: 1px solid var(--sdvm-border);
  }

  .queue-row-badges {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1);
    justify-content: flex-end;
  }

  .queue-header-actions {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .queue-badge {
    white-space: nowrap;
  }

  .queue-expand-icon,
  .naming-disclosure-icon {
    display: inline-flex;
    transition: transform 150ms cubic-bezier(0.25, 1, 0.5, 1);
  }

  .queue-expand-icon--open,
  .naming-disclosure-icon--open {
    transform: rotate(180deg);
  }

  .queue-expand {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 1.5rem;
    height: 1.5rem;
    padding: 0;
    border: 0;
    border-radius: var(--radius-base, 0.25rem);
    background: transparent;
    color: var(--color-surface-400);
    cursor: pointer;
  }

  .queue-expand:hover:not(:disabled),
  .queue-expand:focus-visible {
    color: var(--color-surface-100);
    background-color: var(--color-surface-800);
  }

  .queue-expand:focus-visible {
    outline: 2px solid var(--color-primary-400);
    outline-offset: 1px;
  }

  .queue-row-panel {
    padding: var(--space-3);
    max-height: min(36vh, 14rem);
    overflow-y: auto;
  }

  .queue-panel-intro {
    margin: 0;
  }

  .queue-expand {
    min-width: 0;
    font-weight: var(--weight-medium);
  }

  .file-remove {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 1.5rem;
    height: 1.5rem;
    padding: 0;
    border: 0;
    border-radius: var(--radius-base, 0.25rem);
    background: transparent;
    color: var(--color-surface-500);
    font-size: var(--type-subhead);
    line-height: 1;
    cursor: pointer;
    flex-shrink: 0;
  }

  .file-remove:hover:not(:disabled),
  .file-remove:focus-visible {
    color: var(--sdvm-error-fg);
    background-color: var(--sdvm-error-bg);
  }

  .file-remove:focus-visible {
    outline: 2px solid
      color-mix(in oklab, var(--color-error-500) 50%, transparent);
    outline-offset: 1px;
  }

  .naming-disclosure {
    border: 1px solid var(--sdvm-border);
    border-radius: var(--radius-base, 0.25rem);
    overflow: hidden;
  }

  .naming-disclosure-toggle {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
    width: 100%;
    padding: var(--space-2) var(--space-3);
    border: 0;
    background-color: color-mix(
      in oklab,
      var(--color-surface-900) 30%,
      var(--sdvm-panel)
    );
    color: inherit;
    cursor: pointer;
    text-align: left;
  }

  .naming-disclosure-toggle:hover:not(:disabled),
  .naming-disclosure-toggle:focus-visible {
    background-color: var(--sdvm-raised);
  }

  .naming-disclosure-toggle:focus-visible {
    outline: 2px solid var(--color-primary-400);
    outline-offset: -2px;
  }

  .naming-disclosure-panel {
    padding: var(--space-3);
    border-top: 1px solid var(--sdvm-border);
  }

  .name-display-mode {
    margin: 0;
    padding: 0;
    border: 0;
  }

  .name-display-hint {
    margin: 0;
    text-wrap: pretty;
  }

  .name-display-option {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    cursor: pointer;
  }

  .name-preview-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .name-preview-items {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    max-height: 8rem;
    margin: 0;
    padding: 0;
    list-style: none;
    overflow-y: auto;
  }

  .name-preview-item {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--sdvm-border);
    border-radius: var(--radius-base, 0.25rem);
    background-color: color-mix(
      in oklab,
      var(--color-surface-900) 30%,
      var(--sdvm-panel)
    );
  }

  .overwrite-blocked {
    margin: 0;
    color: var(--sdvm-warning-fg);
  }

  .overwrite-select-required {
    margin: 0;
    color: var(--sdvm-warning-fg);
  }

  .overwrite-target-list,
  .overwrite-sample-list {
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .overwrite-target-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .overwrite-target-option {
    display: flex;
    align-items: flex-start;
    gap: var(--space-2);
    cursor: pointer;
  }

  .overwrite-target-copy {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    min-width: 0;
  }

  .overwrite-sample-details {
    margin-top: var(--space-1);
  }

  .overwrite-sample-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    margin-top: var(--space-2);
  }

  .result-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    max-height: 14rem;
    margin: 0;
    padding: 0 var(--space-6) var(--space-4);
    list-style: none;
    overflow-y: auto;
  }

  .result-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: var(--space-1) var(--space-2);
    align-items: center;
    padding: var(--space-3);
    border: 1px solid var(--sdvm-border);
    border-radius: var(--radius-base, 0.25rem);
    background-color: color-mix(
      in oklab,
      var(--color-surface-900) 30%,
      var(--sdvm-panel)
    );
  }

  .result-row--error {
    border-color: color-mix(
      in oklab,
      var(--sdvm-error-border) 70%,
      var(--sdvm-border)
    );
    background-color: color-mix(
      in oklab,
      var(--sdvm-error-bg) 40%,
      var(--sdvm-panel)
    );
  }

  .result-name {
    font-weight: var(--weight-medium);
    grid-column: 1;
  }

  .result-error {
    grid-column: 1 / -1;
    margin: 0;
    text-wrap: pretty;
  }

  .dependency-confirm {
    width: 100%;
    padding: var(--space-3);
    border: 1px solid
      color-mix(in oklab, var(--color-error-500) 35%, var(--sdvm-border));
    border-radius: var(--radius-base, 0.25rem);
    background: color-mix(
      in oklab,
      var(--sdvm-error-bg) 35%,
      var(--sdvm-panel)
    );
  }

  .dependency-warning-list,
  .dependency-warning-sublist {
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .dependency-warning-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .dependency-warning-item {
    padding: var(--space-2);
    border-radius: var(--radius-base, 0.25rem);
    background: color-mix(in oklab, var(--color-surface-900) 35%, transparent);
  }

  .dependency-warning-sublist {
    margin-top: var(--space-1);
    padding-left: var(--space-2);
  }

  .tag-section {
    padding-top: var(--space-3);
    border-top: 1px solid var(--sdvm-border);
  }

  .tag-hint,
  .tag-empty {
    margin: 0;
    text-wrap: pretty;
  }

  .tag-hint--suggested {
    color: var(--sdvm-info-fg);
  }

  .tag-select-list {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1);
    margin: 0;
    padding: 0;
    list-style: none;
    max-height: 5rem;
    overflow-y: auto;
  }

  .tag-select-item {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    max-width: 100%;
    padding: var(--space-1) var(--space-3);
    border: 1px solid var(--sdvm-border);
    border-radius: var(--radius-base, 0.25rem);
    background-color: color-mix(
      in oklab,
      var(--color-surface-900) 35%,
      var(--sdvm-panel)
    );
    color: inherit;
    cursor: pointer;
    transition:
      background-color 150ms cubic-bezier(0.25, 1, 0.5, 1),
      border-color 150ms cubic-bezier(0.25, 1, 0.5, 1);
  }

  .tag-select-item:hover:not(:disabled) {
    background-color: color-mix(
      in oklab,
      var(--chip-color) 10%,
      var(--sdvm-raised)
    );
    border-color: color-mix(
      in oklab,
      var(--chip-color) 25%,
      var(--sdvm-border)
    );
  }

  .tag-select-item--on {
    background-color: color-mix(
      in oklab,
      var(--chip-color) 14%,
      var(--sdvm-raised)
    );
    border-color: color-mix(in oklab, var(--chip-color) 40%, transparent);
  }

  .tag-select-item:focus-visible {
    outline: 2px solid var(--color-primary-400);
    outline-offset: 1px;
  }

  .tag-select-item:disabled {
    opacity: 0.7;
    cursor: default;
  }

  .tag-select-dot {
    width: 0.5rem;
    height: 0.5rem;
    flex-shrink: 0;
    border-radius: 999px;
    background-color: var(--chip-color);
  }

  .tag-select-name {
    min-width: 0;
    font-weight: var(--weight-medium);
  }

  .tag-select-check {
    flex-shrink: 0;
    color: color-mix(in oklab, var(--chip-color) 75%, var(--color-surface-50));
    font-weight: var(--weight-bold);
  }

  @media (prefers-reduced-motion: reduce) {
    .install-dropzone {
      transition: none;
    }

    .tag-select-item {
      transition: none;
    }

    .install-dialog[open]::backdrop {
      animation: none;
    }

    .queue-expand-icon,
    .naming-disclosure-icon {
      transition: none;
    }
  }
</style>
