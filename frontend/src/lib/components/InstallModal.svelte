<script lang="ts">
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
    dependencyNotInstalled,
    dependencyVersionTooLow,
    dependencyDisabled,
    installOverwriteWarningTitle,
    installOverwriteWarningBody,
    installOverwriteConfirmLabel,
    installOverwriteTargetLegend,
    installOverwriteTargetHint,
    installOverwriteMatchSummary,
    installOverwriteSamplePathsLabel,
  } from "$lib/copy";
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
  let overwriteConfirmOpen = $state(false);
  let overwriteTargets = $state<Record<string, string>>({});

  const isUpdateFlow = $derived(updateTarget != null);
  const installActionLabel = $derived(
    isUpdateFlow && installMode === "replace" ? "Update mod" : "Install",
  );

  const sortedCategories = $derived(
    [...categories].sort(
      (a, b) => a.sortOrder - b.sortOrder || a.name.localeCompare(b.name),
    ),
  );

  const showResults = $derived(results != null);
  const hasBlockedOverwrite = $derived(
    overwritePreviews.some((preview) => preview.state === "blocked"),
  );
  const confirmOverwritePreviews = $derived(
    overwritePreviews.filter((preview) => preview.state === "confirm"),
  );
  const canInstall = $derived(
    paths.length > 0 && !installing && !hasBlockedOverwrite,
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

  function defaultOverwriteTarget(preview: InstallOverwritePreview): string {
    return (
      preview.suggestedTarget?.trim() ||
      preview.candidates?.[0]?.folderPath?.trim() ||
      ""
    );
  }

  function syncOverwriteTargets(previews: InstallOverwritePreview[]) {
    const next: Record<string, string> = { ...overwriteTargets };
    for (const preview of previews) {
      if (preview.state !== "confirm") continue;
      const current = next[preview.archivePath];
      const candidates = preview.candidates ?? [];
      const stillValid = candidates.some(
        (candidate) => candidate.folderPath === current,
      );
      if (!current || !stillValid) {
        next[preview.archivePath] = defaultOverwriteTarget(preview);
      }
    }
    for (const path of Object.keys(next)) {
      if (!previews.some((preview) => preview.archivePath === path)) {
        delete next[path];
      }
    }
    overwriteTargets = next;
  }

  function buildOverwriteTargetsForInstall(): Record<string, string> {
    const targets: Record<string, string> = {};
    for (const preview of confirmOverwritePreviews) {
      const target =
        overwriteTargets[preview.archivePath]?.trim() ||
        defaultOverwriteTarget(preview);
      if (target) {
        targets[preview.archivePath] = target;
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

  function cancelOverwriteConfirm() {
    overwriteConfirmOpen = false;
  }

  async function confirmOverwriteInstall() {
    overwriteConfirmOpen = false;
    await proceedToDependencyCheckAndInstall();
  }

  function setOverwriteTarget(archivePath: string, folderPath: string) {
    overwriteTargets = { ...overwriteTargets, [archivePath]: folderPath };
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
  }

  function clearQueue() {
    paths = [];
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

  function pathsFromFileList(files: FileList | File[]): string[] {
    return [...files]
      .map((f) => (f as File & { path?: string }).path ?? f.name)
      .filter((p) => p.length > 0);
  }

  function onBrowsePick(e: Event) {
    const input = e.currentTarget as HTMLInputElement;
    addPaths(pathsFromFileList(input.files ?? []));
    input.value = "";
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
      }
    } catch {
      // dialog cancelled or unavailable
    }
  }

  function onDrop(e: DragEvent) {
    e.preventDefault();
    dragOver = false;
    const files = e.dataTransfer?.files;
    if (!files?.length) return;
    addPaths(pathsFromFileList(files));
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
      results = await oninstall(batch, [...selectedTagIds], options);
      paths = [];
      dependencyConfirmOpen = false;
      dependencyPreviews = [];
      overwriteConfirmOpen = false;
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
    if (confirmOverwritePreviews.length > 0 && !overwriteConfirmOpen) {
      overwriteConfirmOpen = true;
      return;
    }
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
        namePreviews = (await API.PreviewInstallNames([...paths])) ?? [];
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
          if (manualQueue) browseBtn?.focus();
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
      overwriteConfirmOpen = false;
      overwritePreviews = [];
      overwriteTargets = {};
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
    class="install-panel card app-panel border app-border layout-stack motion-dialog-enter"
  >
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
            Updating <span class="type-data">{updateTarget.manifest.Name}</span>
            in
            <span class="type-data">{modsRoot || "your mod library"}</span>
          {:else}
            Installs to <span class="type-data"
              >{modsRoot || "your mod library"}</span
            >
            <span class="type-meta">
              (enabled mods link into your game Mods folder)</span
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

    {#if showResults && results}
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

      <footer class="install-actions">
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
      {#if isUpdateFlow && updateTarget}
        <fieldset class="update-mode layout-stack-sm">
          <legend class="type-label">Update method</legend>
          <label class="update-mode-option">
            <input
              type="radio"
              bind:group={installMode}
              value="replace"
              disabled={installing}
            />
            <span class="type-ui">Replace existing mod folder</span>
          </label>
          <label class="update-mode-option">
            <input
              type="radio"
              bind:group={installMode}
              value="install"
              disabled={installing}
            />
            <span class="type-ui">Install as new folder</span>
          </label>
          {#if installMode === "replace" && alwaysAskDeleteOnUpdate}
            <label class="flex items-center gap-2">
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
        </fieldset>
      {/if}

      {#if manualQueue}
        <div
          class="install-dropzone"
          class:install-dropzone--active={dragOver}
          role="region"
          aria-label="Drop mod archives here"
          ondragover={(e) => {
            e.preventDefault();
            dragOver = true;
          }}
          ondragleave={() => (dragOver = false)}
          ondrop={onDrop}
        >
          <p class="dropzone-title type-ui text-surface-100">
            Drop mod archives here
          </p>
          <p class="type-caption type-meta">.zip, .7z, or .rar</p>
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
      {/if}

      {#if paths.length > 0}
        <div class="queue-header">
          <span class="type-label"
            >{manualQueue ? "Ready to install" : "Downloaded mod"}</span
          >
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
        <ul class="file-queue" role="list">
          {#each paths as path (path)}
            <li class="file-row">
              <span class="file-name type-ui truncate" title={path}
                >{pathBasename(path)}</span
              >
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
            </li>
          {/each}
        </ul>
      {:else if manualQueue}
        <p class="type-caption type-meta type-prose queue-empty">
          No files selected yet. Drop archives above or choose files from disk.
        </p>
      {/if}

      {#if !isUpdateFlow && paths.length > 0 && (namePreviewBusy || hasNameDisplayChoice)}
        <fieldset class="name-display-mode layout-stack-sm">
          <legend class="type-label">{installDisplayNameLegend}</legend>
          <p class="type-caption type-meta type-prose name-display-hint">
            {installDisplayNameHint}
          </p>
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
        </fieldset>
      {/if}

      {#if !isUpdateFlow && paths.length > 0 && (overwritePreviewBusy || overwritePreviews.length > 0)}
        <div class="overwrite-preview layout-stack-sm">
          {#if overwritePreviewBusy}
            <p class="type-caption type-meta">Checking for file patches…</p>
          {:else}
            {#each overwritePreviews as preview (preview.archivePath)}
              <fieldset class="overwrite-preview-item layout-stack-sm">
                <legend class="type-label"
                  >{installOverwriteWarningTitle()}</legend
                >
                <p class="type-caption type-meta type-prose">
                  <span class="type-ui text-surface-200"
                    >{pathBasename(preview.archivePath)}</span
                  >
                  — {installOverwriteWarningBody(preview.fileCount)}
                </p>
                {#if preview.state === "blocked"}
                  <p
                    class="type-caption type-meta type-prose overwrite-blocked"
                  >
                    {preview.blockReason}
                  </p>
                {:else if preview.candidates?.length}
                  {@const selectedTarget =
                    overwriteTargets[preview.archivePath] ||
                    defaultOverwriteTarget(preview)}
                  {@const selectedCandidate =
                    preview.candidates.find(
                      (item) => item.folderPath === selectedTarget,
                    ) ?? preview.candidates[0]}
                  <p class="type-caption type-meta type-prose">
                    {installOverwriteTargetHint()}
                  </p>
                  <span class="type-caption type-label"
                    >{installOverwriteTargetLegend()}</span
                  >
                  <ul class="overwrite-target-list" role="list">
                    {#each preview.candidates as candidate (candidate.folderPath + candidate.uniqueID)}
                      {@const selected =
                        (overwriteTargets[preview.archivePath] ||
                          defaultOverwriteTarget(preview)) ===
                        candidate.folderPath}
                      <li class="overwrite-target-item">
                        <label class="overwrite-target-option">
                          <input
                            type="radio"
                            name="overwrite-target-{preview.archivePath}"
                            checked={selected}
                            disabled={installing || overwritePreviewBusy}
                            onchange={() =>
                              setOverwriteTarget(
                                preview.archivePath,
                                candidate.folderPath,
                              )}
                          />
                          <span class="overwrite-target-copy">
                            <span class="type-ui">{candidate.modName}</span>
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
                    <div class="overwrite-sample-paths">
                      <span class="type-caption type-label"
                        >{installOverwriteSamplePathsLabel()}</span
                      >
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
                    </div>
                  {/if}
                {/if}
              </fieldset>
            {/each}
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

      <footer
        class="install-actions"
        class:install-actions--confirm={dependencyConfirmOpen ||
          overwriteConfirmOpen}
      >
        {#if overwriteConfirmOpen}
          <div class="dependency-confirm layout-stack-sm">
            <p class="type-ui text-surface-100">
              {installOverwriteWarningTitle()}
            </p>
            <p class="type-caption type-meta type-prose">
              {installOverwriteWarningBody(confirmOverwritePreviews.length)}
            </p>
            <ul class="dependency-warning-list" role="list">
              {#each confirmOverwritePreviews as preview (preview.archivePath)}
                {@const target =
                  overwriteTargets[preview.archivePath] ||
                  defaultOverwriteTarget(preview)}
                {@const candidate = preview.candidates?.find(
                  (item) => item.folderPath === target,
                )}
                <li class="dependency-warning-item">
                  <span class="type-ui text-surface-200"
                    >{pathBasename(preview.archivePath)}</span
                  >
                  <p class="type-caption type-meta type-prose">
                    Merge {preview.fileCount} file{preview.fileCount === 1
                      ? ""
                      : "s"} into
                    <span class="type-ui">{candidate?.modName ?? target}</span>
                    {#if candidate?.folderPath}
                      <span class="type-mono"> ({candidate.folderPath})</span>
                    {/if}
                  </p>
                </li>
              {/each}
            </ul>
          </div>
          <button
            type="button"
            class="btn preset-tonal flex-1"
            disabled={installing}
            onclick={cancelOverwriteConfirm}
          >
            Cancel
          </button>
          <button
            type="button"
            class="btn preset-filled-primary-500 flex-1"
            disabled={installing}
            aria-busy={installing}
            onclick={confirmOverwriteInstall}
          >
            {installing ? "Merging…" : installOverwriteConfirmLabel()}
          </button>
        {:else if dependencyConfirmOpen}
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
      </footer>
    {/if}
  </div>
</dialog>

<style>
  .install-results-summary {
    text-wrap: pretty;
    margin: 0;
  }

  .install-dialog {
    padding: 0;
    margin: auto;
    border: none;
    background: transparent;
    width: min(32rem, calc(100vw - var(--space-8)));
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
    padding: var(--space-6);
    margin: 0;
    gap: var(--space-4);
    max-height: calc(100vh - var(--space-8));
    overflow-y: auto;
  }

  .install-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--space-3);
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

  .install-dropzone--active {
    border-color: var(--color-primary-500);
    background-color: color-mix(
      in oklab,
      var(--color-primary-500) 10%,
      var(--sdvm-panel)
    );
  }

  .dropzone-title {
    font-weight: var(--weight-medium);
    text-wrap: balance;
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
    gap: var(--space-1);
    max-height: 10rem;
    margin: 0;
    padding: 0;
    list-style: none;
    overflow-y: auto;
  }

  .file-row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--sdvm-border);
    border-radius: var(--radius-base, 0.25rem);
    background-color: color-mix(
      in oklab,
      var(--color-surface-900) 35%,
      var(--sdvm-panel)
    );
  }

  .file-name {
    flex: 1;
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

  .result-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    max-height: 14rem;
    margin: 0;
    padding: 0;
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

  .install-actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
    padding-top: var(--space-1);
  }

  .install-actions--confirm {
    flex-direction: column;
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

  .overwrite-preview-item {
    margin: 0;
    padding: var(--space-3);
    border: 1px solid
      color-mix(in oklab, var(--color-primary-500) 25%, var(--sdvm-border));
    border-radius: var(--radius-base, 0.25rem);
    background: color-mix(
      in oklab,
      var(--color-primary-500) 6%,
      var(--sdvm-panel)
    );
  }

  .overwrite-blocked {
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

  .overwrite-sample-paths {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .overwrite-sample-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
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
    padding-top: var(--space-1);
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
    max-height: 8rem;
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

  .update-mode {
    margin: 0;
    padding: var(--space-3);
    border: 1px solid var(--sdvm-border);
    border-radius: var(--radius-md);
  }

  .update-mode-option {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    cursor: pointer;
  }

  .name-display-mode {
    margin: 0;
    padding: var(--space-3);
    border: 1px solid var(--sdvm-border);
    border-radius: var(--radius-md);
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
    margin-top: var(--space-1);
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
  }
</style>
