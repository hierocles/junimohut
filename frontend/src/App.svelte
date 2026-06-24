<script lang="ts">
  import { onMount, tick } from "svelte";
  import { Events } from "@wailsio/runtime";
  import {
    INSTALL_MODAL_DROP_ID,
    MOD_GRID_DROP_ID,
  } from "$lib/wails/archiveFileDrop";
  import * as API from "$lib/api";
  import {
    refreshCore,
    refreshFooterStats,
    fetchLibraryMods,
    setModEnabled,
    previewInstallDependencies,
    USE_MOCK_DATA,
    type Mod,
    type Profile,
    type Category,
    type Settings,
    type InstallResult,
    type InstallOptions,
    type UnmanagedMod,
  } from "$lib/api/client";
  import { loadTranslations } from "$lib/i18n";
  import {
    formatUserError,
    modCount,
    installCompleteLine,
    normalizeArchivePaths,
    consumeLibraryMilestone,
    allEnabledMessage,
    launchSentMessage,
    updatesCheckedMessage,
    hutProverb,
    dependencyIssuesFooterMessage,
    dependencyIssueCountLabel,
    unmanagedModCountLabel,
    unmanagedModsDialogTitle,
    unmanagedModsDialogMessage,
    unmanagedModsOpenFolderLabel,
    unmanagedModsDismissLabel,
    updatesFilterFooterMessage,
    statusDismissLabel,
    toolbarProfileAria,
    tagsSidebarShowTitle,
    tagsSidebarShowAria,
    reinstallSavedConfirmTitle,
    reinstallSavedConfirmLabel,
    reinstallSavedConfirmMessage,
    reinstallSavedDeleteOldLabel,
    reinstallSavedMissingError,
    reinstallSavedSuccessLine,
    downloadingModFromNexus,
    downloadNoArchivePath,
    downloadCompleteReview,
    deleteDownloadConfirmTitle,
    deleteDownloadConfirmMessage,
    deletedDownloadMessage,
    launchingSmapi,
    checkingModUpdates,
    bulkToggleStatus,
    updatedModMessage,
    tagsAppliedMessage,
    modEndorsedOnNexus,
    modUpdateIgnoredMessage,
    modUpdateResumedMessage,
    noNexusUpdateKey,
    downloadingUpdateForMessage,
    updateDownloadedForMessage,
    createdTagMessage,
    deletedModMessage,
    deleteModConfirmTitle,
    deleteModConfirmMessage,
    deleteModConfirmLabel,
    deleteModsBatchConfirmTitle,
    deleteModsBatchConfirmMessage,
    deleteModsBatchConfirmLabel,
    deleteBundleConfirmMessage,
    deleteModDeleteArchiveLabel,
    deleteModDeleteArchiveHint,
    deleteModDeleteArchiveNoneHint,
    deletedModsMessage,
    deletedTagMessage,
    deletedProfileMessage,
    renamedTagMessage,
    renamedModMessage,
    clearedModDisplayNameMessage,
    tagAddedToMod,
    tagRemovedFromMod,
    nexusKeyRequired,
    nexusConnectedMessage,
    nexusKeyRejected,
    registerNxmSuccess,
    smapiInstallerOpened,
    filterCleared,
    searchModsPlaceholder,
    searchModsAria,
    downloadsLoadError,
    downloadsBulkReinstallConfirmTitle,
    downloadsBulkReinstallConfirmMessage,
    configEditorUnsavedProfileTitle,
    configEditorUnsavedProfileBody,
  } from "$lib/copy";
  import {
    parseNxmModId,
    nexusModIdFromUpdateKey,
    suggestedTagIdsForInstall,
  } from "$lib/mods/nexusTags";
  import { nexusModPageUrl } from "$lib/mods/dependencies";
  import { openExternalUrl } from "$lib/wails/openExternalUrl";
  import type { GridStatusFilter } from "$lib/mods/filter";
  import { displayModName } from "$lib/mods/names";
  import {
    bundleDeleteFolderPaths,
    bundleUpdateTarget,
    configTargetMod,
    findModInList,
    isBundleChildMod,
    isBundleMod,
  } from "$lib/mods/bundles";
  import type { SavedDownloadRecord } from "$lib/mods/savedDownloads";
  import { applyDocumentTheme } from "$lib/themes/applyDocumentTheme";
  import {
    Settings as SettingsIcon,
    Download,
    RefreshCw,
    X,
    MoreHorizontal,
    Tags,
  } from "@lucide/svelte";
  import SetupWizard from "$lib/components/SetupWizard.svelte";
  import CategorySidebar from "$lib/components/CategorySidebar.svelte";
  import ModGrid from "$lib/components/ModGrid.svelte";
  import ModDetailPane from "$lib/components/ModDetailPane.svelte";
  import SettingsDrawer from "$lib/components/SettingsDrawer.svelte";
  import DropdownList from "$lib/components/DropdownList.svelte";
  import DownloadsPanel from "$lib/components/DownloadsPanel.svelte";
  import ModContextMenu from "$lib/components/ModContextMenu.svelte";
  import ConfirmDialog from "$lib/components/ConfirmDialog.svelte";
  import InstallModal from "$lib/components/InstallModal.svelte";
  import AppShellHeader from "$lib/components/AppShellHeader.svelte";
  import AboutDialog from "$lib/components/AboutDialog.svelte";

  let settings = $state<Settings | null>(null);
  let mods = $state<Mod[]>([]);
  let libraryMods = $state<Mod[]>([]);
  let profiles = $state<Profile[]>([]);
  let categories = $state<Category[]>([]);
  let smapiVersion = $state("");
  let readyCount = $state(0);
  let dependencyIssueCount = $state(0);
  let unmanagedModCount = $state(0);
  let unmanagedMods = $state<UnmanagedMod[]>([]);
  let unmanagedModsOpen = $state(false);
  let gridStatusFilter = $state<GridStatusFilter>("none");
  let search = $state("");
  let selectedModId = $state<string | null>(null);
  let settingsOpen = $state(false);
  let downloadsOpen = $state(false);
  let downloads = $state<
    NonNullable<Awaited<ReturnType<typeof API.ListDownloads>>>
  >([]);
  let savedDownloads = $state<SavedDownloadRecord[]>([]);
  let downloadsPaneWidth = $state(loadDownloadsPaneWidth());
  let nexusConnected = $state(false);
  let statusMessage = $state("");
  let newProfileName = $state("");
  let showNewProfile = $state(false);
  let renamingProfileId = $state<string | null>(null);
  let renamingProfileName = $state("");
  let smapiUpdateAvailable = $state<string | null>(null);
  let contextMod = $state<Mod | null>(null);
  const contextSuppressUpdateActions = $derived(
    contextMod != null && isBundleChildMod(mods, contextMod),
  );
  let detailPane = $state<{ focusDisplayNameInput: () => void } | undefined>();
  let contextPos = $state({ x: 0, y: 0 });
  let loading = $state(false);
  let loadError = $state("");
  let searchDebounced = $state("");
  let confirmBusy = $state(false);
  let categoryCreating = $state(false);
  let sidebarWidth = $state(loadSidebarWidth());
  let tagsSidebarVisible = $state(loadTagsSidebarVisible());
  let launchBusy = $state(false);
  let updatesBusy = $state(false);
  let profileBusy = $state(false);
  let installModalOpen = $state(false);
  let installQueue = $state<string[]>([]);
  let installUpdateTarget = $state<Mod | null>(null);
  let installSuggestedTagIds = $state<string[]>([]);
  let installFromNexus = $state(false);
  let pendingNexusModIds = $state<number[]>([]);
  let nexusDownloadInFlight = $state(false);
  let pendingNexusDownloadPath = $state("");
  let downloadPollTimer: ReturnType<typeof setInterval> | undefined;
  let downloadsFetchSeq = 0;
  let downloadsLoading = $state(false);
  let downloadsRefreshing = $state(false);
  let downloadsFetchError = $state("");
  let downloadsEverLoaded = $state(false);
  let launchReadyFlash = $state(false);
  let titleClickCount = $state(0);
  let aboutOpen = $state(false);
  let statusTone = $state<"default" | "success" | "error">("default");
  let statusProgress = $state(false);
  let statusDismissible = $state(false);
  let statusSparkle = $state(false);
  let statusDismissTimer: ReturnType<typeof setTimeout> | undefined;
  let pendingConfirm = $state<
    | { kind: "delete-mod"; mod: Mod }
    | { kind: "delete-mods-batch"; mods: Mod[] }
    | { kind: "delete-category"; id: string; name: string }
    | { kind: "delete-profile"; id: string; name: string }
    | { kind: "reinstall-saved"; mod: Mod; archivePath: string }
    | {
        kind: "reinstall-saved-batch";
        items: { mod: Mod; archivePath: string }[];
      }
    | {
        kind: "delete-download";
        record: SavedDownloadRecord;
        displayName: string;
      }
    | { kind: "switch-profile-config"; profileId: string }
    | null
  >(null);
  let reinstallDeleteOld = $state(false);
  let deleteModArchivesToo = $state(false);

  let profileDialogEl = $state<HTMLDialogElement | null>(null);

  let loadSeq = 0;
  let lastLoadKey = "";
  let titleClickTimer: ReturnType<typeof setTimeout> | undefined;
  let aboutDebounceTimer: ReturnType<typeof setTimeout> | undefined;
  let lastSmapiVersion = $state("");

  type StatusOptions = {
    progress?: boolean;
    sticky?: boolean;
    duration?: number;
    sparkle?: boolean;
  };

  function clearStatusDismissTimer() {
    if (statusDismissTimer) {
      clearTimeout(statusDismissTimer);
      statusDismissTimer = undefined;
    }
  }

  function clearStatus() {
    clearStatusDismissTimer();
    statusMessage = "";
    statusTone = "default";
    statusProgress = false;
    statusDismissible = false;
    statusSparkle = false;
  }

  function setStatus(
    message: string,
    tone: "default" | "success" | "error" = "default",
    options: StatusOptions = {},
  ) {
    clearStatusDismissTimer();
    statusMessage = message;
    statusTone = tone;
    statusProgress = options.progress ?? false;
    statusSparkle = options.sparkle ?? false;

    const sticky = options.sticky ?? tone === "error";
    statusDismissible = message.length > 0 && !statusProgress;

    if (statusProgress || sticky) return;

    const duration = options.duration ?? (tone === "success" ? 4000 : 5000);
    statusDismissTimer = setTimeout(() => {
      clearStatus();
    }, duration);
  }

  function dismissStatus() {
    clearStatus();
  }

  function setError(error: unknown) {
    setStatus(formatUserError(error), "error", { sticky: true });
  }

  function isNexusKeyRejected(error: unknown): boolean {
    const msg = formatUserError(error).toLowerCase();
    return (
      msg.includes("validate api key failed") ||
      msg.includes("http 401") ||
      msg.includes("http 403")
    );
  }

  /** Re-check stored Nexus API key against the live API (launch + after connect). */
  async function refreshNexusConnection() {
    if (USE_MOCK_DATA) return;
    if (!(await API.IsNexusConnected())) {
      nexusConnected = false;
      return;
    }
    nexusConnected = await API.ProbeNexusAPIKey();
  }

  function openProfileDialog() {
    renamingProfileName = activeProfile?.name ?? "";
    profileDialogEl?.showModal();
  }

  async function saveProfileRename() {
    if (!activeProfile || !renamingProfileName.trim() || profileBusy) return;
    profileBusy = true;
    try {
      await API.RenameProfile(activeProfile.id, renamingProfileName.trim());
      await load();
    } catch (e) {
      setError(e);
    } finally {
      profileBusy = false;
    }
  }

  async function createProfileFromDialog() {
    const name = newProfileName.trim();
    if (!name || profileBusy) return;
    profileBusy = true;
    try {
      await API.CreateProfile(name);
      newProfileName = "";
      profileDialogEl?.close();
      await load();
    } catch (e) {
      setError(e);
    } finally {
      profileBusy = false;
    }
  }

  function onTitleClick() {
    titleClickCount += 1;
    if (titleClickTimer) clearTimeout(titleClickTimer);
    titleClickTimer = setTimeout(() => {
      titleClickCount = 0;
    }, 2000);

    if (titleClickCount >= 5) {
      titleClickCount = 0;
      if (aboutDebounceTimer) {
        clearTimeout(aboutDebounceTimer);
        aboutDebounceTimer = undefined;
      }
      setStatus(hutProverb(), "success");
      return;
    }

    if (aboutDebounceTimer) clearTimeout(aboutDebounceTimer);
    aboutDebounceTimer = setTimeout(() => {
      if (titleClickCount >= 1 && titleClickCount < 5) {
        aboutOpen = true;
      }
      titleClickCount = 0;
      aboutDebounceTimer = undefined;
    }, 350);
  }

  $effect(() => {
    const version = smapiVersion;
    if (!version || version === lastSmapiVersion) return;
    lastSmapiVersion = version;
    launchReadyFlash = true;
    const timer = setTimeout(() => {
      launchReadyFlash = false;
    }, 700);
    return () => clearTimeout(timer);
  });

  const hideDisabled = $derived(settings?.hideDisabledFilter ?? "none");
  const confirmOpen = $derived(pendingConfirm !== null);
  const deleteConfirmArchiveCount = $derived.by(() => {
    if (!pendingConfirm) return 0;
    if (pendingConfirm.kind === "delete-mod") {
      return pendingConfirm.mod.savedDownloadPath?.trim() ? 1 : 0;
    }
    if (pendingConfirm.kind === "delete-mods-batch") {
      return pendingConfirm.mods.filter((m) => m.savedDownloadPath?.trim())
        .length;
    }
    return 0;
  });
  const confirmTitle = $derived.by(() => {
    if (!pendingConfirm) return "";
    switch (pendingConfirm.kind) {
      case "delete-mod":
        return deleteModConfirmTitle;
      case "delete-mods-batch":
        return deleteModsBatchConfirmTitle;
      case "delete-profile":
        return "Delete profile?";
      case "delete-download":
        return deleteDownloadConfirmTitle;
      case "reinstall-saved-batch":
        return downloadsBulkReinstallConfirmTitle;
      case "reinstall-saved":
        return reinstallSavedConfirmTitle;
      case "switch-profile-config":
        return configEditorUnsavedProfileTitle;
      default:
        return "Delete tag?";
    }
  });
  const confirmMessage = $derived.by(() => {
    if (!pendingConfirm) return "";
    switch (pendingConfirm.kind) {
      case "delete-mod":
        if (isBundleMod(pendingConfirm.mod)) {
          return deleteBundleConfirmMessage(
            displayModName(pendingConfirm.mod),
            pendingConfirm.mod.bundleChildren?.length ??
              pendingConfirm.mod.enabledTotal ??
              0,
          );
        }
        return deleteModConfirmMessage(displayModName(pendingConfirm.mod));
      case "delete-mods-batch":
        return deleteModsBatchConfirmMessage(pendingConfirm.mods.length);
      case "delete-category":
        return `”${pendingConfirm.name}” will be removed. Installed mods keep their files — only the tag and assignments are deleted.`;
      case "delete-profile":
        return `”${pendingConfirm.name}” and its mod enable/disable state will be permanently deleted.`;
      case "delete-download":
        return deleteDownloadConfirmMessage(
          pendingConfirm.displayName,
          pendingConfirm.record.fileName ?? pendingConfirm.record.archivePath,
        );
      case "reinstall-saved-batch":
        return downloadsBulkReinstallConfirmMessage(
          pendingConfirm.items.length,
        );
      case "reinstall-saved":
        return reinstallSavedConfirmMessage(
          displayModName(pendingConfirm.mod),
          pendingConfirm.archivePath,
        );
      case "switch-profile-config":
        return configEditorUnsavedProfileBody;
      default:
        return "";
    }
  });
  const confirmActionLabel = $derived.by(() => {
    if (!pendingConfirm) return "Confirm";
    switch (pendingConfirm.kind) {
      case "delete-mod":
        return deleteModConfirmLabel;
      case "delete-mods-batch":
        return deleteModsBatchConfirmLabel;
      case "delete-profile":
        return "Delete profile";
      case "delete-download":
        return "Delete archive";
      case "reinstall-saved-batch":
      case "reinstall-saved":
        return reinstallSavedConfirmLabel;
      case "switch-profile-config":
        return "Switch profile";
      default:
        return "Delete tag";
    }
  });
  const confirmVariant = $derived(
    pendingConfirm?.kind === "reinstall-saved" ||
      pendingConfirm?.kind === "reinstall-saved-batch"
      ? "default"
      : "danger",
  );
  const showDeleteArchiveCheckbox = $derived(
    pendingConfirm?.kind === "delete-mod" ||
      pendingConfirm?.kind === "delete-mods-batch",
  );
  const activeTagsFilterCount = $derived(
    categories.filter((c) => c.visible).length,
  );
  const tagsFilterNarrowed = $derived(
    categories.length > 0 &&
      activeTagsFilterCount > 0 &&
      activeTagsFilterCount < categories.length,
  );

  function loadSidebarWidth(): number {
    try {
      const n = parseInt(localStorage.getItem("sdvm-sidebar-width") ?? "", 10);
      if (n >= 200 && n <= 480) return n;
    } catch {
      /* storage unavailable */
    }
    return 280;
  }

  function setSidebarWidth(w: number) {
    sidebarWidth = w;
    try {
      localStorage.setItem("sdvm-sidebar-width", String(w));
    } catch {
      /* storage unavailable */
    }
  }

  function loadDownloadsPaneWidth(): number {
    try {
      const n = parseInt(
        localStorage.getItem("sdvm-downloads-pane-width") ?? "",
        10,
      );
      if (n >= 280 && n <= 480) return n;
    } catch {
      /* storage unavailable */
    }
    return 380;
  }

  function setDownloadsPaneWidth(w: number) {
    downloadsPaneWidth = w;
    try {
      localStorage.setItem("sdvm-downloads-pane-width", String(w));
    } catch {
      /* storage unavailable */
    }
  }

  function loadTagsSidebarVisible(): boolean {
    try {
      const raw = localStorage.getItem("sdvm-tags-sidebar-visible");
      if (raw === "0" || raw === "false") return false;
    } catch {
      /* storage unavailable */
    }
    return true;
  }

  function setTagsSidebarVisible(visible: boolean) {
    tagsSidebarVisible = visible;
    try {
      localStorage.setItem("sdvm-tags-sidebar-visible", visible ? "1" : "0");
    } catch {
      /* storage unavailable */
    }
  }

  async function load() {
    const search = searchDebounced;
    let filter = hideDisabled;
    const key = `${search}\0${filter}`;
    if (key === lastLoadKey) return;

    const seq = ++loadSeq;
    loading = true;
    loadError = "";
    try {
      let core = await refreshCore({ search, hideDisabled: filter });
      if (seq !== loadSeq) return;
      const savedFilter = core.settings?.hideDisabledFilter ?? "none";
      if (settings === null && filter !== savedFilter) {
        filter = savedFilter;
        core = await refreshCore({ search, hideDisabled: filter });
        if (seq !== loadSeq) return;
      }
      mods = core.mods;
      if (search === "" && filter === "none") {
        libraryMods = core.mods;
      } else if (libraryMods.length === 0) {
        void refreshLibraryMods();
      }
      profiles = core.profiles;
      categories = core.categories;
      settings = core.settings;
      smapiVersion = core.smapiVersion;
      applyDocumentTheme(settings?.theme ?? "stardew-dark");
      lastLoadKey = `${search}\0${filter}`;
    } catch (e) {
      if (seq !== loadSeq) return;
      loadError = formatUserError(e);
    } finally {
      if (seq === loadSeq) loading = false;
    }
    if (seq !== loadSeq) return;
    try {
      const stats = await refreshFooterStats();
      if (seq !== loadSeq) return;
      readyCount = stats.readyCount;
      dependencyIssueCount = stats.dependencyIssueCount;
      unmanagedMods = stats.unmanagedMods;
      unmanagedModCount = unmanagedMods.length;
    } catch {
      /* footer counts are non-critical */
    }
  }

  async function refreshLibraryMods() {
    try {
      libraryMods = await fetchLibraryMods();
    } catch {
      /* downloads panel can retry via refresh */
    }
  }

  onMount(() => {
    loadTranslations("en");
    void refreshNexusConnection();
    void checkSMAPIUpdate();
    Events.On("mods-changed", () => {
      if (loading) return;
      lastLoadKey = "";
      void load();
    });
    Events.On("nxm-url", (ev) => {
      if (ev.data) void handleNXMURL(ev.data);
    });
    Events.On("nexus-download-ready", (ev) => {
      const path = ev.data?.trim() ?? "";
      if (!path) return;
      pendingNexusDownloadPath = path;
    });
    Events.On("files-dropped", (ev) => {
      const payload = ev.data;
      if (!payload?.files?.length) return;
      const targetId = payload.targetId?.trim() ?? "";
      if (
        targetId !== INSTALL_MODAL_DROP_ID &&
        targetId !== MOD_GRID_DROP_ID
      ) {
        return;
      }
      queueInstallArchives(payload.files);
    });
    return () => {
      clearStatusDismissTimer();
    };
  });

  function startDownloadPolling() {
    void refreshDownloads({ background: downloadsEverLoaded });
    if (downloadPollTimer) clearInterval(downloadPollTimer);
    downloadPollTimer = setInterval(async () => {
      await refreshDownloads({ background: true });
      const active = (downloads ?? []).some((d) =>
        d.status.toLowerCase().includes("download"),
      );
      if (!active && downloadPollTimer) {
        clearInterval(downloadPollTimer);
        downloadPollTimer = undefined;
      }
    }, 500);
  }

  type DownloadEntry = NonNullable<
    Awaited<ReturnType<typeof API.ListDownloads>>
  >[number];

  function downloadEntryPath(entry: DownloadEntry): string {
    const record = entry as DownloadEntry & { FilePath?: string };
    return (record.filePath ?? record.FilePath ?? "").trim();
  }

  function latestCompletedDownloadPath(
    entries: DownloadEntry[] = downloads,
  ): string {
    for (let i = entries.length - 1; i >= 0; i--) {
      const entry = entries[i];
      if (!entry.status.toLowerCase().includes("complete")) continue;
      const filePath = downloadEntryPath(entry);
      if (filePath) return filePath;
    }
    return "";
  }

  async function fetchDownloads(): Promise<DownloadEntry[]> {
    if (USE_MOCK_DATA) return downloads;
    try {
      const entries = (await API.ListDownloads()) ?? [];
      downloads = entries;
      return entries;
    } catch {
      return downloads;
    }
  }

  async function fetchSavedDownloads(): Promise<SavedDownloadRecord[]> {
    if (USE_MOCK_DATA) {
      const { getMockSavedDownloads } = await import("$lib/mock/designData");
      savedDownloads = getMockSavedDownloads();
      return savedDownloads;
    }
    try {
      const entries = (await API.ListSavedDownloads()) ?? [];
      savedDownloads = entries;
      return entries;
    } catch {
      return savedDownloads;
    }
  }

  async function refreshDownloads(options?: { background?: boolean }) {
    const seq = ++downloadsFetchSeq;
    const background = options?.background ?? downloadsEverLoaded;

    if (background) {
      downloadsRefreshing = true;
    } else {
      downloadsLoading = true;
    }
    downloadsFetchError = "";

    try {
      let failed = false;

      if (USE_MOCK_DATA) {
        const { getMockSavedDownloads } = await import("$lib/mock/designData");
        savedDownloads = getMockSavedDownloads();
      } else {
        try {
          downloads = (await API.ListDownloads()) ?? [];
        } catch {
          failed = true;
        }
        if (seq !== downloadsFetchSeq) return;
        try {
          savedDownloads = (await API.ListSavedDownloads()) ?? [];
        } catch {
          failed = true;
        }
      }

      if (seq !== downloadsFetchSeq) return;

      if (failed) {
        downloadsFetchError = downloadsLoadError;
      } else {
        downloadsEverLoaded = true;
        downloadsFetchError = "";
      }
    } finally {
      if (seq === downloadsFetchSeq) {
        downloadsLoading = false;
        downloadsRefreshing = false;
      }
    }
  }

  async function resolveDownloadedArchivePath(
    rpcPath: string | undefined,
  ): Promise<string> {
    const trimmed = rpcPath?.trim() ?? "";
    if (trimmed) return trimmed;
    if (pendingNexusDownloadPath) return pendingNexusDownloadPath;
    const entries = await fetchDownloads();
    return latestCompletedDownloadPath(entries);
  }

  function openNexusInstallModal(
    path: string,
    modIds: number[],
    updateTarget: Mod | null = null,
  ): boolean {
    const trimmed = path.trim();
    if (!trimmed) return false;
    installUpdateTarget = updateTarget;
    installQueue = [trimmed];
    pendingNexusModIds = modIds;
    installFromNexus = true;
    installSuggestedTagIds = [];
    installModalOpen = true;
    return true;
  }

  async function loadInstallSuggestedTags(paths: string[], modIds: number[]) {
    const uniqueModIds = [...new Set(modIds.filter((id) => id > 0))];
    const uniquePaths = [
      ...new Set(paths.map((p) => p.trim()).filter(Boolean)),
    ];
    if (uniqueModIds.length === 0 && uniquePaths.length === 0) {
      installSuggestedTagIds = [];
      return;
    }
    const known = new Set(categories.map((c) => c.id));
    if (USE_MOCK_DATA) {
      installSuggestedTagIds = suggestedTagIdsForInstall(
        uniquePaths,
        uniqueModIds,
        known,
      );
      return;
    }
    try {
      const ids =
        (await API.GetInstallSuggestedTags(uniquePaths, uniqueModIds)) ?? [];
      installSuggestedTagIds = ids.filter((id) => known.has(id));
    } catch {
      installSuggestedTagIds = [];
    }
  }

  $effect(() => {
    if (!installModalOpen) return;
    const paths = installQueue;
    const modIds = pendingNexusModIds;
    void loadInstallSuggestedTags(paths, modIds);
  });

  async function handleNXMURL(url: string) {
    if (nexusDownloadInFlight) return;
    nexusDownloadInFlight = true;
    pendingNexusDownloadPath = "";
    const nexusModId = parseNxmModId(url);
    const modIds = nexusModId ? [nexusModId] : [];
    pendingNexusModIds = modIds;
    const targetMod =
      nexusModId != null && nexusModId > 0
        ? (libraryMods.find((m) =>
            m.manifest?.UpdateKeys?.some(
              (k: string) => nexusModIdFromUpdateKey(k) === nexusModId,
            ),
          ) ?? null)
        : null;
    try {
      setStatus(downloadingModFromNexus, "default", { progress: true });
      startDownloadPolling();
      const rpcPath = await API.HandleNXMURL(url);
      const path = await resolveDownloadedArchivePath(rpcPath);
      if (!openNexusInstallModal(path, modIds, targetMod)) {
        setError(downloadNoArchivePath);
        return;
      }
      setStatus(downloadCompleteReview, "success");
      await refreshDownloads();
    } catch (e) {
      const path = await resolveDownloadedArchivePath(undefined);
      if (openNexusInstallModal(path, modIds, targetMod)) {
        setStatus(downloadCompleteReview, "success");
      } else {
        setError(e);
      }
    } finally {
      nexusDownloadInFlight = false;
      pendingNexusDownloadPath = "";
    }
  }

  $effect(() => {
    const q = search;
    const timer = setTimeout(() => {
      searchDebounced = q;
    }, 250);
    return () => clearTimeout(timer);
  });

  $effect(() => {
    searchDebounced;
    hideDisabled;
    void load();
  });

  async function toggleMod(id: string, enabled: boolean) {
    await setModEnabled(id, enabled);
    await load();
  }

  async function bulkToggleMods(ids: string[], enabled: boolean) {
    const targets = enabled
      ? ids
      : ids.filter((id) => {
          const mod = findModInList(mods, id);
          return mod && !mod.isCoreMod;
        });
    if (!targets.length) return;
    try {
      await Promise.all(targets.map((id) => setModEnabled(id, enabled)));
      if (enabled && targets.length === mods.length) {
        const delight = allEnabledMessage(mods.length);
        setStatus(
          delight ?? bulkToggleStatus(enabled, targets.length),
          delight ? "success" : "default",
        );
      } else {
        setStatus(bulkToggleStatus(enabled, targets.length));
      }
      await load();
    } catch (e) {
      setError(e);
    }
  }

  async function downloadModUpdate(mod: Mod) {
    await runContextAction(mod, "downloadUpdate");
  }

  async function checkSMAPIUpdate() {
    try {
      const info = await API.CheckSMAPIUpdate();
      if (info?.updateAvailable)
        smapiUpdateAvailable = info.latestVersion ?? null;
    } catch {
      // non-critical
    }
  }

  async function switchProfile(id: string) {
    if (renamingProfileId) return;
    if (activeProfile?.id === id) return;
    try {
      if (await API.ConfigEditorIsDirty()) {
        pendingConfirm = { kind: "switch-profile-config", profileId: id };
        return;
      }
      await API.SetActiveProfile(id);
      await API.ReloadConfigEditor();
      await load();
    } catch (e) {
      setError(e);
    }
  }

  async function openConfigEditor(mod: Mod) {
    const target = configTargetMod(mod);
    if (!target?.hasJsonFiles) return;
    try {
      await API.OpenModConfigEditor(target.id);
    } catch (e) {
      setError(e);
    }
  }

  async function createProfile() {
    const name = newProfileName.trim();
    if (!name || profileBusy) return;
    profileBusy = true;
    try {
      await API.CreateProfile(name);
      newProfileName = "";
      showNewProfile = false;
      await load();
    } catch (e) {
      setError(e);
    } finally {
      profileBusy = false;
    }
  }

  async function renameProfile(id: string, name: string) {
    try {
      await API.RenameProfile(id, name);
      renamingProfileId = null;
      await load();
    } catch (e) {
      setError(e);
    }
  }

  async function setModCustomName(modId: string, name: string) {
    try {
      if (USE_MOCK_DATA) {
        const { setMockModCustomName } = await import("$lib/mock/designData");
        setMockModCustomName(modId, name);
      } else {
        await API.SetModCustomName(modId, name);
      }
      const trimmed = name.trim();
      mods = mods.map((m) => {
        if (m.id !== modId) return m;
        return { ...m, customName: trimmed || undefined };
      });
      setStatus(
        trimmed ? renamedModMessage(trimmed) : clearedModDisplayNameMessage(),
        "success",
      );
    } catch (e) {
      setError(e);
    }
  }

  function requestDeleteProfile(id: string) {
    const p = profiles.find((p) => p.id === id);
    if (!p) return;
    pendingConfirm = { kind: "delete-profile", id, name: p.name };
  }

  async function launchSMAPI() {
    if (launchBusy) return;
    launchBusy = true;
    setStatus(launchingSmapi, "default", { progress: true });
    try {
      await API.LaunchSMAPI();
      setStatus(launchSentMessage(), "success");
    } catch (e) {
      setError(e);
    } finally {
      launchBusy = false;
    }
  }

  async function checkUpdates() {
    if (updatesBusy) return;
    updatesBusy = true;
    setStatus(checkingModUpdates, "default", { progress: true });
    try {
      const before = readyCount;
      await API.CheckModUpdates();
      lastLoadKey = "";
      await load();
      setStatus(
        updatesCheckedMessage(mods.length, readyCount),
        readyCount === 0 && before === 0 ? "success" : "default",
      );
    } catch (e) {
      setError(e);
    } finally {
      updatesBusy = false;
    }
  }

  function openInstallModal(
    paths: string[] = [],
    updateTarget: Mod | null = null,
    opts?: { trusted?: boolean },
  ) {
    const trimmed = paths.map((p) => p.trim()).filter(Boolean);
    const incoming = opts?.trusted ? trimmed : normalizeArchivePaths(trimmed);
    installUpdateTarget = updateTarget;
    installQueue = incoming;
    pendingNexusModIds = [];
    installFromNexus = false;
    installSuggestedTagIds = [];
    installModalOpen = true;
  }

  function queueInstallArchives(paths: string[]) {
    const incoming = normalizeArchivePaths(paths);
    if (incoming.length === 0) return;
    if (installModalOpen) {
      installQueue = normalizeArchivePaths([...installQueue, ...incoming]);
      return;
    }
    openInstallModal(incoming);
  }

  function closeInstallModal() {
    installModalOpen = false;
    installQueue = [];
    installUpdateTarget = null;
    pendingNexusModIds = [];
    installFromNexus = false;
    installSuggestedTagIds = [];
  }

  async function reinstallFromSaved(
    mod: Mod,
    archivePath: string,
    deleteOld: boolean,
  ) {
    const trimmed = archivePath.trim();
    if (!trimmed) {
      setStatus(reinstallSavedMissingError, "error");
      return;
    }
    if (USE_MOCK_DATA) {
      setStatus(reinstallSavedSuccessLine(displayModName(mod)), "success");
      return;
    }
    await API.UpdateMod(mod.folderPath, trimmed, deleteOld);
    setStatus(reinstallSavedSuccessLine(displayModName(mod)), "success");
    await load();
  }

  async function runInstall(
    paths: string[],
    tagIds: string[],
    options: InstallOptions = { mode: "install" },
  ): Promise<InstallResult[]> {
    try {
      if (
        installUpdateTarget &&
        options.mode === "replace" &&
        paths.length === 1
      ) {
        await API.UpdateMod(
          installUpdateTarget.folderPath,
          paths[0],
          options.deleteOld ?? false,
        );
        const name = displayModName(installUpdateTarget);
        setStatus(updatedModMessage(name), "success");
        await load();
        return [
          {
            folderPath: installUpdateTarget.folderPath,
            modId: installUpdateTarget.id,
            name,
          },
        ];
      }

      const results =
        (await API.InstallMods(
          paths,
          options.useFolderDisplayNames ?? false,
          options.overwriteTargets ?? {},
        )) ?? [];
      const installed = results.filter((r) => !r.error);
      const failed = results.length - installed.length;
      const tone = failed === 0 ? "success" : "error";
      let message = installCompleteLine(installed.length, failed);
      if (failed > 0) {
        const firstError = results.find((r) => r.error)?.error;
        if (firstError) message += ` ${formatUserError(firstError)}`;
      }
      if (tagIds.length > 0) {
        const modIds = installed
          .map((r) => r.modId)
          .filter((id): id is string => !!id);
        let tagged = 0;
        for (const modId of modIds) {
          for (const tagId of tagIds) {
            try {
              await API.AssignModToCategory(tagId, modId);
              tagged++;
            } catch (e) {
              setError(e);
            }
          }
        }
        if (tagged > 0 && modIds.length > 0) {
          message += ` ${tagsAppliedMessage(modIds.length)}`;
        }
      }
      setStatus(message, tone, { sticky: tone === "error" });
      await load();
      const milestone = consumeLibraryMilestone(mods.length);
      if (milestone && failed === 0) {
        setStatus(
          `${installCompleteLine(installed.length, 0)} ${milestone}`,
          "success",
          {
            sparkle: true,
            duration: 6000,
          },
        );
      }
      return results;
    } catch (e) {
      setError(e);
      throw e;
    }
  }

  async function handleModContext(mod: Mod, action: string) {
    if (action.startsWith("install:")) {
      const paths = action.slice(8).split("|");
      openInstallModal(paths);
      return;
    }
    if (action === "menu") {
      contextMod = mod;
      return;
    }
    await runContextAction(mod, action);
  }

  function showContextMenu(mod: Mod, e: MouseEvent) {
    e.preventDefault();
    contextMod = mod;
    contextPos = { x: e.clientX, y: e.clientY };
  }

  async function runContextAction(mod: Mod, action: string) {
    contextMod = null;
    try {
      switch (action) {
        case "openFolder":
          await API.OpenModFolder(mod.folderPath);
          break;
        case "openManifest":
          await API.OpenManifest(mod.folderPath);
          break;
        case "editConfig":
          await openConfigEditor(mod);
          break;
        case "rename":
          selectedModId = mod.id;
          await tick();
          detailPane?.focusDisplayNameInput();
          break;
        case "openPage": {
          const key = mod.manifest?.UpdateKeys?.find((k: string) =>
            k.startsWith("Nexus:"),
          );
          if (key) {
            const id = key.split(":")[1];
            void openExternalUrl(
              `https://www.nexusmods.com/stardewvalley/mods/${id}`,
            );
          }
          break;
        }
        case "endorse": {
          const key = mod.manifest?.UpdateKeys?.find((k: string) =>
            k.startsWith("Nexus:"),
          );
          if (key) await API.EndorseMod(key, mod.manifest.Version);
          setStatus(modEndorsedOnNexus, "success");
          break;
        }
        case "ignoreUpdate":
          await API.SetModUpdateIgnored(bundleUpdateTarget(mods, mod).id, true);
          lastLoadKey = "";
          await load();
          setStatus(modUpdateIgnoredMessage, "success");
          break;
        case "resumeUpdate":
          await API.SetModUpdateIgnored(
            bundleUpdateTarget(mods, mod).id,
            false,
          );
          lastLoadKey = "";
          await load();
          setStatus(modUpdateResumedMessage, "success");
          break;
        case "downloadUpdate": {
          const updateMod = bundleUpdateTarget(mods, mod);
          const key = updateMod.manifest?.UpdateKeys?.find((k: string) =>
            k.startsWith("Nexus:"),
          );
          if (!key) {
            setStatus(noNexusUpdateKey, "error");
            break;
          }
          if (nexusDownloadInFlight) break;
          nexusDownloadInFlight = true;
          pendingNexusDownloadPath = "";
          const nexusModId = nexusModIdFromUpdateKey(key);
          const modIds = nexusModId ? [nexusModId] : [];
          pendingNexusModIds = modIds;
          try {
            setStatus(
              downloadingUpdateForMessage(displayModName(updateMod)),
              "default",
              { progress: true },
            );
            startDownloadPolling();
            const rpcPath = await API.DownloadModUpdate(
              key,
              displayModName(updateMod),
            );
            const path = await resolveDownloadedArchivePath(rpcPath);
            if (!openNexusInstallModal(path, modIds, updateMod)) {
              setError(downloadNoArchivePath);
              break;
            }
            setStatus(
              updateDownloadedForMessage(displayModName(updateMod)),
              "success",
            );
            await refreshDownloads();
          } catch (e) {
            const path = await resolveDownloadedArchivePath(undefined);
            if (openNexusInstallModal(path, modIds, updateMod)) {
              setStatus(
                updateDownloadedForMessage(displayModName(updateMod)),
                "success",
              );
            } else {
              setError(e);
            }
          } finally {
            nexusDownloadInFlight = false;
            pendingNexusDownloadPath = "";
          }
          break;
        }
        case "reinstallSaved": {
          const archivePath = mod.savedDownloadPath?.trim() ?? "";
          if (!archivePath) {
            setStatus(reinstallSavedMissingError, "error");
            break;
          }
          if (settings?.alwaysAskDeleteOnUpdate) {
            reinstallDeleteOld = false;
            pendingConfirm = { kind: "reinstall-saved", mod, archivePath };
          } else {
            await reinstallFromSaved(mod, archivePath, false);
          }
          break;
        }
        case "delete":
          deleteModArchivesToo = false;
          pendingConfirm = { kind: "delete-mod", mod };
          break;
      }
    } catch (e) {
      setError(e);
    }
  }

  async function toggleCategoryVisibility(id: string, visible: boolean) {
    await API.SetCategoryVisibility(id, visible);
    categories = categories.map((c) => (c.id === id ? { ...c, visible } : c));
  }

  async function showAllCategoryFilters() {
    const hidden = categories.filter((c) => !c.visible);
    if (hidden.length === 0) return;
    try {
      await Promise.all(
        hidden.map((c) => API.SetCategoryVisibility(c.id, true)),
      );
      categories = categories.map((c) => ({ ...c, visible: true }));
    } catch (e) {
      setError(e);
    }
  }

  async function createCategory(name: string, color: string) {
    const trimmed = name.trim();
    if (!trimmed) return;
    categoryCreating = true;
    try {
      await API.CreateCategory(trimmed, color);
      await load();
      setStatus(createdTagMessage(trimmed), "success");
    } catch (e) {
      setError(e);
      throw e;
    } finally {
      categoryCreating = false;
    }
  }

  function requestDeleteCategory(id: string) {
    const cat = categories.find((c) => c.id === id);
    if (!cat) return;
    pendingConfirm = { kind: "delete-category", id, name: cat.name };
  }

  function requestBulkDeleteMods(modsToDelete: Mod[]) {
    const targets = modsToDelete.filter((m) => !m.isCoreMod);
    if (!targets.length) return;
    deleteModArchivesToo = false;
    pendingConfirm = { kind: "delete-mods-batch", mods: targets };
  }

  function deleteFolderPathsForMods(modsToDelete: Mod[]): string[] {
    const paths = new Set<string>();
    for (const mod of modsToDelete) {
      for (const path of bundleDeleteFolderPaths(mod)) {
        paths.add(path);
      }
    }
    return [...paths];
  }

  async function confirmPending() {
    if (!pendingConfirm || confirmBusy) return;
    confirmBusy = true;
    try {
      if (pendingConfirm.kind === "delete-mod") {
        const folderPaths = deleteFolderPathsForMods([pendingConfirm.mod]);
        if (folderPaths.length === 1) {
          await API.DeleteMod(folderPaths[0], deleteModArchivesToo);
        } else {
          const result = await API.DeleteMods(
            folderPaths,
            deleteModArchivesToo,
          );
          if (result.errors?.length) {
            setError(new Error(result.errors[0]));
          }
        }
        if (selectedModId === pendingConfirm.mod.id) selectedModId = null;
        setStatus(
          deletedModMessage(displayModName(pendingConfirm.mod)),
          "success",
        );
        if (deleteModArchivesToo) await refreshDownloads();
      } else if (pendingConfirm.kind === "delete-mods-batch") {
        const deletedIds = new Set(pendingConfirm.mods.map((m) => m.id));
        const result = await API.DeleteMods(
          deleteFolderPathsForMods(pendingConfirm.mods),
          deleteModArchivesToo,
        );
        if (selectedModId && deletedIds.has(selectedModId))
          selectedModId = null;
        if (result.deletedCount > 0) {
          setStatus(
            deletedModsMessage(
              result.deletedCount,
              result.archivesDeletedCount ?? 0,
            ),
            "success",
          );
        }
        if (result.errors?.length) {
          setError(new Error(result.errors[0]));
        }
        if (deleteModArchivesToo) await refreshDownloads();
      } else if (pendingConfirm.kind === "delete-category") {
        await API.DeleteCategory(pendingConfirm.id);
        setStatus(deletedTagMessage(pendingConfirm.name), "success");
      } else if (pendingConfirm.kind === "delete-profile") {
        await API.DeleteProfile(pendingConfirm.id);
        setStatus(deletedProfileMessage(pendingConfirm.name), "success");
      } else if (pendingConfirm.kind === "reinstall-saved") {
        await reinstallFromSaved(
          pendingConfirm.mod,
          pendingConfirm.archivePath,
          reinstallDeleteOld,
        );
      } else if (pendingConfirm.kind === "reinstall-saved-batch") {
        for (const item of pendingConfirm.items) {
          await reinstallFromSaved(
            item.mod,
            item.archivePath,
            reinstallDeleteOld,
          );
        }
      } else if (pendingConfirm.kind === "delete-download") {
        await API.DeleteSavedDownload(pendingConfirm.record.archivePath);
        await refreshDownloads();
        setStatus(
          deletedDownloadMessage(pendingConfirm.displayName),
          "success",
        );
      } else if (pendingConfirm.kind === "switch-profile-config") {
        await API.SetConfigEditorDirty(false);
        await API.SetActiveProfile(pendingConfirm.profileId);
        await API.ReloadConfigEditor();
      }
      pendingConfirm = null;
      reinstallDeleteOld = false;
      deleteModArchivesToo = false;
      await load();
    } catch (e) {
      setError(e);
    } finally {
      confirmBusy = false;
    }
  }

  function cancelConfirm() {
    if (confirmBusy) return;
    pendingConfirm = null;
    reinstallDeleteOld = false;
    deleteModArchivesToo = false;
  }

  async function toggleModTag(
    modId: string,
    categoryId: string,
    assign: boolean,
  ) {
    try {
      if (assign) {
        await API.AssignModToCategory(categoryId, modId);
        setStatus(tagAddedToMod, "success");
      } else {
        await API.UnassignModFromCategory(categoryId, modId);
        setStatus(tagRemovedFromMod, "success");
      }
      await load();
    } catch (e) {
      setError(e);
      throw e;
    }
  }

  async function saveSettings(s: Settings) {
    try {
      await API.SaveSettings(s);
      settings = s;
      if (s.theme) {
        applyDocumentTheme(s.theme);
      }
      if (!settingsOpen) {
        await load();
      }
    } catch (e) {
      setError(e);
      throw e;
    }
  }

  async function updateVisibleColumns(cols: string[]) {
    if (!settings) return;
    const next = { ...settings, visibleColumns: cols };
    settings = next;
    try {
      await API.SaveSettings(next);
    } catch (e) {
      setError(e);
      await load();
    }
  }

  async function connectNexus(key: string) {
    const trimmed = key.trim();
    if (!trimmed) {
      setStatus(nexusKeyRequired, "error");
      return;
    }
    try {
      await API.SetNexusAPIKey(trimmed);
      nexusConnected = await API.ValidateNexusAPIKey();
      setStatus(
        nexusConnected ? nexusConnectedMessage : nexusKeyRejected,
        nexusConnected ? "success" : "error",
      );
    } catch (e) {
      nexusConnected = false;
      if (isNexusKeyRejected(e)) {
        setStatus(nexusKeyRejected, "error");
      } else {
        setError(e);
      }
    }
  }

  async function openDownloads() {
    if (downloadsOpen) {
      downloadsOpen = false;
      return;
    }
    downloadsOpen = true;
    void Promise.all([
      refreshDownloads({ background: downloadsEverLoaded }),
      refreshLibraryMods(),
    ]);
  }

  function viewModFromDownloads(modId: string) {
    selectedModId = modId;
    downloadsOpen = false;
  }

  function installFromArchive(archivePath: string) {
    installFromArchives([archivePath]);
  }

  function installFromArchives(paths: string[]) {
    const incoming = paths.map((p) => p.trim()).filter(Boolean);
    if (!incoming.length) return;
    installUpdateTarget = null;
    installQueue = incoming;
    installFromNexus = false;
    pendingNexusModIds = [];
    installModalOpen = true;
  }

  function bulkReinstallFromArchives(
    items: { mod: Mod; archivePath: string }[],
  ) {
    if (!items.length) return;
    if (settings?.alwaysAskDeleteOnUpdate) {
      reinstallDeleteOld = false;
      pendingConfirm = { kind: "reinstall-saved-batch", items };
      return;
    }
    void runBulkReinstallFromArchives(items, false);
  }

  async function runBulkReinstallFromArchives(
    items: { mod: Mod; archivePath: string }[],
    deleteOld: boolean,
  ) {
    for (const item of items) {
      await reinstallFromSaved(item.mod, item.archivePath, deleteOld);
    }
  }

  function requestReinstallFromArchive(mod: Mod, archivePath: string) {
    if (settings?.alwaysAskDeleteOnUpdate) {
      reinstallDeleteOld = false;
      pendingConfirm = { kind: "reinstall-saved", mod, archivePath };
      return;
    }
    void reinstallFromSaved(mod, archivePath, false);
  }

  async function showArchiveInFolder(archivePath: string) {
    try {
      await API.RevealArchiveInFileManager(archivePath);
    } catch (e) {
      setError(e);
    }
  }

  function requestDeleteArchive(
    record: SavedDownloadRecord,
    displayName: string,
  ) {
    pendingConfirm = { kind: "delete-download", record, displayName };
  }

  function openArchiveOnNexus(nexusModId: number) {
    if (nexusModId <= 0) return;
    void openExternalUrl(nexusModPageUrl(String(nexusModId)));
  }

  async function registerNXMProtocol() {
    try {
      await API.RegisterNXMProtocol();
      setStatus(registerNxmSuccess, "success");
    } catch (e) {
      setError(e);
    }
  }

  const activeProfile = $derived(
    profiles.find((p) => p.isActive) ?? profiles[0],
  );
  const enabledCount = $derived(mods.filter((m) => m.enabled).length);
  const modsRoot = $derived(settings?.modsRoot ?? "");
  const activeDownloadCount = $derived(
    (downloads ?? []).filter(
      (d) => !d.status.toLowerCase().includes("complete"),
    ).length,
  );

  function reportError(message: string) {
    setStatus(message, "error");
  }

  function showDependencyIssuesHint() {
    gridStatusFilter =
      gridStatusFilter === "dependencies" ? "none" : "dependencies";
    setStatus(
      gridStatusFilter === "dependencies"
        ? dependencyIssuesFooterMessage(dependencyIssueCount)
        : filterCleared,
      "default",
    );
  }

  function showUpdatesFilter() {
    gridStatusFilter = gridStatusFilter === "updates" ? "none" : "updates";
    setStatus(
      gridStatusFilter === "updates"
        ? updatesFilterFooterMessage(readyCount)
        : filterCleared,
      "default",
    );
  }

  function clearGridStatusFilter() {
    gridStatusFilter = "none";
  }
</script>

{#snippet appStatusFooter()}
  <footer
    class="app-footer border-t app-border type-ui"
    class:app-footer--error={!loadError && statusTone === "error"}
    class:app-footer--success={!loadError && statusTone === "success"}
  >
    {#if loadError}
      <div class="footer-alert footer-alert--error" role="alert">
        <span class="footer-alert-icon" aria-hidden="true">!</span>
        <p class="footer-alert-text">{loadError}</p>
        <button
          type="button"
          class="btn btn-sm preset-tonal shrink-0"
          onclick={() => load()}>Try again</button
        >
      </div>
    {:else}
      <div class="footer-badges">
        {#if smapiUpdateAvailable}
          <button
            type="button"
            class="update-badge"
            onclick={() => {
              settingsOpen = true;
              smapiUpdateAvailable = null;
            }}
            title="SMAPI {smapiUpdateAvailable} is available — open Settings to install"
          >
            <span class="update-badge-count">SMAPI</span>
            <span>{smapiUpdateAvailable} available</span>
          </button>
        {/if}
        {#if readyCount > 0}
          <button
            type="button"
            class="update-badge"
            class:update-badge--active={gridStatusFilter === "updates"}
            aria-pressed={gridStatusFilter === "updates"}
            onclick={showUpdatesFilter}
            title={gridStatusFilter === "updates"
              ? "Clear update filter"
              : "Show mods with updates available"}
          >
            <span class="update-badge-count">{readyCount}</span>
            <span
              >{readyCount === 1
                ? "update available"
                : "updates available"}</span
            >
          </button>
        {/if}
        {#if dependencyIssueCount > 0}
          <button
            type="button"
            class="update-badge update-badge--deps"
            class:update-badge--active={gridStatusFilter === "dependencies"}
            aria-pressed={gridStatusFilter === "dependencies"}
            onclick={showDependencyIssuesHint}
            title={gridStatusFilter === "dependencies"
              ? "Clear dependency filter"
              : "Show mods with dependency issues"}
          >
            <span class="update-badge-count">{dependencyIssueCount}</span>
            <span>{dependencyIssueCountLabel(dependencyIssueCount)}</span>
          </button>
        {/if}
        {#if unmanagedModCount > 0}
          <button
            type="button"
            class="update-badge update-badge--unmanaged"
            onclick={() => (unmanagedModsOpen = true)}
            title="Show mods installed directly in the game Mods folder"
          >
            <span class="update-badge-count">{unmanagedModCount}</span>
            <span>{unmanagedModCountLabel(unmanagedModCount)}</span>
          </button>
        {/if}
      </div>
      {#if statusMessage}
        <div
          class="footer-status"
          class:footer-status--error={statusTone === "error"}
          class:footer-status--success={statusTone === "success"}
          class:footer-status--progress={statusProgress}
          class:footer-status--sparkle={statusSparkle}
          role={statusTone === "error" ? "alert" : "status"}
          aria-live={statusTone === "error" ? "assertive" : "polite"}
          aria-busy={statusProgress}
        >
          {#if statusSparkle && statusTone === "success"}
            <span class="milestone-sparkle" aria-hidden="true">
              <span></span><span></span><span></span>
            </span>
          {/if}
          {#if statusProgress}
            <RefreshCw
              size={14}
              class="spin-icon footer-status-spinner"
              aria-hidden="true"
            />
          {:else if statusTone === "error"}
            <span class="footer-status-icon" aria-hidden="true">!</span>
          {/if}
          {#key statusMessage}
            <span
              class="footer-status-text motion-status-in"
              title={statusMessage}>{statusMessage}</span
            >
          {/key}
          {#if statusDismissible}
            <button
              type="button"
              class="footer-status-dismiss"
              aria-label={statusDismissLabel}
              onclick={dismissStatus}
            >
              <X size={12} aria-hidden="true" />
            </button>
          {/if}
        </div>
      {/if}
    {/if}
  </footer>
{/snippet}

{#if settings && !settings.setupComplete}
  <div class="app-shell h-full flex flex-col text-surface-50">
    <SetupWizard {settings} oncomplete={load} onerror={reportError} />
    {@render appStatusFooter()}
  </div>
{:else}
  <div class="app-shell h-full flex flex-col text-surface-50">
    <AppShellHeader {onTitleClick}>
      {#snippet toolbar()}
        <!-- Left zone: mod count + profile -->
        <div class="toolbar-zone toolbar-zone-left">
          <span class="type-caption type-data tabular-nums toolbar-mod-count">
            <span class="state-success font-semibold">{enabledCount}</span><span
              class="type-meta"
            >
              / {mods.length}</span
            >
          </span>
          {#if USE_MOCK_DATA}
            <span
              class="state-badge state-badge--update type-caption font-medium"
              >Mock</span
            >
          {/if}

          <div
            class="toolbar-tags-slot"
            class:toolbar-tags-slot--open={!tagsSidebarVisible}
            aria-hidden={tagsSidebarVisible}
          >
            <button
              type="button"
              class="btn btn-sm preset-tonal toolbar-icon-btn toolbar-tags-btn"
              class:toolbar-tags-btn--filtered={tagsFilterNarrowed}
              onclick={() => setTagsSidebarVisible(true)}
              title={tagsSidebarShowTitle(
                activeTagsFilterCount,
                tagsFilterNarrowed,
              )}
              aria-label={tagsSidebarShowAria(
                activeTagsFilterCount,
                tagsFilterNarrowed,
              )}
              aria-expanded={false}
              aria-controls="tags-sidebar"
              tabindex={tagsSidebarVisible ? -1 : 0}
            >
              <Tags size={14} aria-hidden="true" />
              {#if tagsFilterNarrowed}
                <span
                  class="toolbar-badge toolbar-tags-filter-badge type-caption tabular-nums"
                  aria-hidden="true"
                >
                  {activeTagsFilterCount}
                </span>
              {/if}
            </button>
          </div>

          <div class="toolbar-group toolbar-profile">
            <DropdownList
              size="sm"
              class="font-medium"
              ariaLabel={toolbarProfileAria}
              options={profiles.map((p) => ({ value: p.id, label: p.name }))}
              value={activeProfile?.id ?? ""}
              onchange={(id) => switchProfile(id)}
            />
            <button
              class="btn btn-sm preset-tonal toolbar-icon-btn"
              onclick={openProfileDialog}
              title="Manage profiles"
              aria-label="Manage profiles"
            >
              <MoreHorizontal size={14} />
            </button>
          </div>
        </div>

        <!-- Center zone: search -->
        <div class="toolbar-zone toolbar-zone-center">
          <input
            class="input input-sm toolbar-search-input"
            placeholder={searchModsPlaceholder}
            aria-label={searchModsAria}
            bind:value={search}
          />
        </div>

        <!-- Right zone: actions + icon buttons + launch -->
        <div class="toolbar-zone toolbar-zone-right">
          <button
            class="btn btn-sm preset-tonal font-medium"
            onclick={() => openInstallModal()}
          >
            Install Mod
          </button>

          <div class="toolbar-icon-group">
            <button
              class="btn btn-sm preset-tonal toolbar-icon-btn toolbar-downloads-btn"
              class:preset-filled={downloadsOpen}
              onclick={openDownloads}
              title="Downloads"
              aria-label={activeDownloadCount > 0
                ? `Downloads (${activeDownloadCount} active)`
                : "Downloads"}
              aria-expanded={downloadsOpen}
              aria-controls="downloads-pane"
            >
              <Download size={14} />
              {#if activeDownloadCount > 0}
                <span class="toolbar-badge" aria-hidden="true"
                  >{activeDownloadCount}</span
                >
              {/if}
            </button>

            <button
              class="btn btn-sm preset-tonal toolbar-icon-btn"
              onclick={checkUpdates}
              disabled={updatesBusy}
              aria-busy={updatesBusy}
              title={updatesBusy
                ? "Checking for updates…"
                : "Check for updates"}
              aria-label="Check for updates"
            >
              <span
                class:spin-icon={updatesBusy}
                style="display: flex; align-items: center;"
              >
                <RefreshCw size={14} />
              </span>
            </button>

            <button
              class="btn btn-sm preset-tonal toolbar-icon-btn"
              onclick={() => (settingsOpen = true)}
              title="Settings"
              aria-label="Settings"
            >
              <SettingsIcon size={14} />
            </button>
          </div>

          <button
            class="btn btn-sm launch-cta preset-filled-primary-500 type-ui font-semibold"
            class:launch-cta--ready-flash={launchReadyFlash}
            onclick={launchSMAPI}
            disabled={launchBusy || !smapiVersion}
            title={smapiVersion
              ? `Launch SMAPI ${smapiVersion}`
              : "SMAPI not found — check Settings"}
            aria-busy={launchBusy}
          >
            {launchBusy ? "Launching…" : "Launch SMAPI"}
          </button>
        </div>
      {/snippet}
    </AppShellHeader>

    <dialog
      bind:this={profileDialogEl}
      class="profile-dialog motion-dialog-enter"
      onclick={(e) => {
        if (e.target === profileDialogEl) profileDialogEl?.close();
      }}
      onclose={() => {
        newProfileName = "";
      }}
    >
      <div class="profile-dialog-panel app-panel">
        <div class="profile-dialog-header">
          <h2 class="type-subhead">Manage Profiles</h2>
          <button
            class="btn btn-sm preset-tonal toolbar-icon-btn"
            onclick={() => profileDialogEl?.close()}
            aria-label="Close"
          >
            <X size={14} />
          </button>
        </div>

        <div class="profile-dialog-section">
          <p class="profile-dialog-label">Rename active profile</p>
          <div class="profile-dialog-row field-action-row">
            <input
              class="input input-sm"
              bind:value={renamingProfileName}
              placeholder="Profile name"
              aria-label="Rename profile"
              maxlength="80"
              disabled={profileBusy}
              onkeydown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  void saveProfileRename();
                }
                if (e.key === "Escape") {
                  e.preventDefault();
                  profileDialogEl?.close();
                }
              }}
            />
            <button
              class="btn btn-sm preset-filled"
              onclick={saveProfileRename}
              disabled={profileBusy || !renamingProfileName.trim()}
              aria-busy={profileBusy}
            >
              {profileBusy ? "Saving…" : "Save"}
            </button>
          </div>
        </div>

        <div class="profile-dialog-section">
          <p class="profile-dialog-label">New profile</p>
          <div class="profile-dialog-row field-action-row">
            <input
              class="input input-sm"
              bind:value={newProfileName}
              placeholder="Profile name"
              aria-label="New profile name"
              maxlength="80"
              disabled={profileBusy}
              onkeydown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  void createProfileFromDialog();
                }
              }}
            />
            <button
              class="btn btn-sm preset-filled"
              onclick={createProfileFromDialog}
              disabled={profileBusy || !newProfileName.trim()}
              aria-busy={profileBusy}
            >
              {profileBusy ? "Creating…" : "Create"}
            </button>
          </div>
        </div>

        {#if profiles.length > 1}
          <div class="profile-dialog-section profile-dialog-section--danger">
            <button
              class="btn btn-sm preset-filled-error-500"
              onclick={() => {
                if (activeProfile) {
                  requestDeleteProfile(activeProfile.id);
                  profileDialogEl?.close();
                }
              }}
            >
              Delete "{activeProfile?.name}"
            </button>
          </div>
        {/if}
      </div>
    </dialog>

    <div
      class="layout-workspace"
      class:layout-workspace--tags-hidden={!tagsSidebarVisible}
      style="--sidebar-width: {sidebarWidth}px"
    >
      <div
        class="layout-workspace-sidebar"
        class:layout-workspace-sidebar--hidden={!tagsSidebarVisible}
        aria-hidden={!tagsSidebarVisible}
        inert={!tagsSidebarVisible ? true : undefined}
      >
        <CategorySidebar
          {categories}
          creating={categoryCreating}
          ontoggle={toggleCategoryVisibility}
          onshowall={showAllCategoryFilters}
          oncreate={createCategory}
          onrename={async (id, name) => {
            try {
              await API.UpdateCategory(
                id,
                name,
                categories.find((c) => c.id === id)?.color ?? "",
                categories.find((c) => c.id === id)?.visible ?? true,
                categories.find((c) => c.id === id)?.sortOrder ?? 0,
              );
              await load();
              setStatus(renamedTagMessage(name), "success");
            } catch (e) {
              setError(e);
              throw e;
            }
          }}
          ondelete={requestDeleteCategory}
          onwidthchange={setSidebarWidth}
          onhide={() => setTagsSidebarVisible(false)}
        />
      </div>
      <div
        class="layout-main-col"
        class:layout-main-col--downloads-open={downloadsOpen}
        style={downloadsOpen
          ? `--downloads-pane-width: ${downloadsPaneWidth}px`
          : undefined}
      >
        <div class="layout-main-grid-col">
          <ModGrid
            {mods}
            {categories}
            {gridStatusFilter}
            onClearGridStatusFilter={clearGridStatusFilter}
            {selectedModId}
            searchQuery={search}
            refreshing={loading}
            onselect={(id) => (selectedModId = id)}
            ontoggle={toggleMod}
            onbulktoggle={bulkToggleMods}
            onbulkdelete={requestBulkDeleteMods}
            ondownloadupdate={downloadModUpdate}
            oncontext={(mod, action, event) => {
              if (action === "menu" && event) showContextMenu(mod, event);
              else handleModContext(mod, action);
            }}
            onqueueinstall={openInstallModal}
            ontoggletag={toggleModTag}
            visibleColumns={settings?.visibleColumns}
            lastUpdateCheck={settings?.lastUpdateCheck ?? 0}
            oncolumnschange={updateVisibleColumns}
          />
          {#if !downloadsOpen}
            <ModDetailPane
              bind:this={detailPane}
              {selectedModId}
              {mods}
              {libraryMods}
              {categories}
              lastUpdateCheck={settings?.lastUpdateCheck ?? 0}
              onclose={() => (selectedModId = null)}
              ondownloadupdate={downloadModUpdate}
              onignoreupdate={(m) => runContextAction(m, "ignoreUpdate")}
              onresumeupdate={(m) => runContextAction(m, "resumeUpdate")}
              onenabledependency={(modId) => toggleMod(modId, true)}
              onselectmod={(id) => (selectedModId = id)}
              onsetcustomname={setModCustomName}
              oneditconfig={openConfigEditor}
            />
          {/if}
        </div>
        {#if downloadsOpen}
          <DownloadsPanel
            activeDownloads={downloads ?? []}
            {savedDownloads}
            mods={libraryMods}
            loading={downloadsLoading}
            refreshing={downloadsRefreshing}
            fetchError={downloadsFetchError}
            onretry={() => void refreshDownloads()}
            paneWidth={downloadsPaneWidth}
            onwidthchange={setDownloadsPaneWidth}
            onclose={() => (downloadsOpen = false)}
            oninstall={installFromArchive}
            onreinstall={requestReinstallFromArchive}
            onbulkinstall={installFromArchives}
            onbulkreinstall={bulkReinstallFromArchives}
            onshowfolder={(path) => void showArchiveInFolder(path)}
            ondelete={requestDeleteArchive}
            onopennexus={openArchiveOnNexus}
            onviewmod={viewModFromDownloads}
          />
        {/if}
      </div>
    </div>

    {@render appStatusFooter()}
  </div>
{/if}

<InstallModal
  open={installModalOpen}
  bind:paths={installQueue}
  updateTarget={installUpdateTarget}
  manualQueue={!installFromNexus}
  alwaysAskDeleteOnUpdate={settings?.alwaysAskDeleteOnUpdate ?? false}
  showInstallSummary={settings?.showInstallSummary ?? true}
  {categories}
  {modsRoot}
  suggestedTagIds={installSuggestedTagIds}
  onclose={closeInstallModal}
  oninstall={runInstall}
  onpreview={previewInstallDependencies}
/>

<SettingsDrawer
  bind:drawerOpen={settingsOpen}
  settings={settings ?? ({} as Settings)}
  {nexusConnected}
  onclose={() => (settingsOpen = false)}
  onsave={saveSettings}
  onnexus={connectNexus}
  onerror={reportError}
  oninstallsmapi={async () => {
    try {
      await API.InstallSMAPI();
      setStatus(smapiInstallerOpened, "success");
    } catch (e) {
      setError(e);
    }
  }}
  onregisternxm={registerNXMProtocol}
/>

<AboutDialog open={aboutOpen} onclose={() => (aboutOpen = false)} />

<ConfirmDialog
  open={unmanagedModsOpen}
  title={unmanagedModsDialogTitle()}
  message={unmanagedModsDialogMessage()}
  confirmLabel={unmanagedModsOpenFolderLabel()}
  cancelLabel={unmanagedModsDismissLabel()}
  onconfirm={async () => {
    try {
      await API.OpenActiveModsFolder();
    } catch (e) {
      setError(e);
    } finally {
      unmanagedModsOpen = false;
    }
  }}
  oncancel={() => (unmanagedModsOpen = false)}
>
  <ul class="unmanaged-mod-list layout-stack-xs" role="list">
    {#each unmanagedMods as entry (entry.folderName)}
      <li class="unmanaged-mod-item">
        <span class="type-ui text-surface-100">{entry.name}</span>
        <span class="type-caption type-meta type-mono">{entry.folderName}</span>
        {#if entry.uniqueID}
          <span class="type-caption type-meta">{entry.uniqueID}</span>
        {/if}
      </li>
    {/each}
  </ul>
</ConfirmDialog>

<ConfirmDialog
  open={confirmOpen}
  title={confirmTitle}
  message={confirmMessage}
  confirmLabel={confirmActionLabel}
  cancelLabel="Keep"
  variant={confirmVariant}
  busy={confirmBusy}
  onconfirm={confirmPending}
  oncancel={cancelConfirm}
>
  {#if pendingConfirm?.kind === "reinstall-saved" || pendingConfirm?.kind === "reinstall-saved-batch"}
    <label class="flex items-center gap-2">
      <input
        type="checkbox"
        class="checkbox"
        bind:checked={reinstallDeleteOld}
        disabled={confirmBusy}
      />
      <span class="type-caption type-meta">{reinstallSavedDeleteOldLabel}</span>
    </label>
  {:else if showDeleteArchiveCheckbox}
    <label class="flex items-center gap-2">
      <input
        type="checkbox"
        class="checkbox"
        bind:checked={deleteModArchivesToo}
        disabled={confirmBusy || deleteConfirmArchiveCount === 0}
      />
      <span class="type-caption type-meta">{deleteModDeleteArchiveLabel}</span>
    </label>
    <span class="type-caption type-meta">
      {deleteConfirmArchiveCount > 0
        ? deleteModDeleteArchiveHint(deleteConfirmArchiveCount)
        : deleteModDeleteArchiveNoneHint}
    </span>
  {/if}
</ConfirmDialog>

<ModContextMenu
  mod={contextMod}
  x={contextPos.x}
  y={contextPos.y}
  suppressUpdateActions={contextSuppressUpdateActions}
  onaction={(action) => contextMod && runContextAction(contextMod, action)}
  onclose={() => (contextMod = null)}
/>

<style>
  :global(html, body, #app) {
    height: 100%;
  }

  .launch-cta {
    padding-inline: var(--space-5);
    white-space: nowrap;
  }

  .update-badge {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-1) var(--space-3) var(--space-1) var(--space-2);
    font-size: var(--type-meta);
    font-weight: var(--weight-semibold);
    color: var(--sdvm-warning-fg);
    background-color: var(--sdvm-warning-bg);
    border: 1px solid var(--sdvm-warning-border);
    border-radius: var(--radius-base, 0.25rem);
    transition: background-color var(--motion-fast) var(--ease-out-quart);
  }

  .update-badge:hover {
    background-color: color-mix(
      in oklab,
      var(--color-warning-500) 22%,
      var(--color-surface-900)
    );
  }

  .update-badge:focus-visible {
    outline: 2px solid
      color-mix(in oklab, var(--color-warning-500) 50%, transparent);
    outline-offset: 2px;
  }

  .update-badge-count {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 1.375rem;
    height: 1.375rem;
    padding-inline: 0.25rem;
    font-size: var(--type-caption);
    font-weight: var(--weight-bold);
    color: var(--color-surface-950);
    background-color: var(--color-warning-400);
    border-radius: var(--radius-base, 0.25rem);
  }

  .update-badge--deps {
    color: var(--sdvm-error-fg);
    background-color: var(--sdvm-error-bg);
    border-color: var(--sdvm-error-border);
  }

  .update-badge--deps:hover {
    background-color: color-mix(
      in oklab,
      var(--color-error-500) 22%,
      var(--color-surface-900)
    );
  }

  .update-badge--deps:focus-visible {
    outline: 2px solid
      color-mix(in oklab, var(--color-error-500) 50%, transparent);
    outline-offset: 2px;
  }

  .update-badge--deps .update-badge-count {
    color: var(--color-surface-50);
    background-color: var(--color-error-500);
  }

  .update-badge--unmanaged {
    color: var(--sdvm-warning-fg);
    background-color: var(--sdvm-warning-bg);
    border-color: var(--sdvm-warning-border);
  }

  .update-badge--unmanaged:hover {
    background-color: color-mix(
      in oklab,
      var(--color-warning-500) 22%,
      var(--color-surface-900)
    );
  }

  .update-badge--unmanaged:focus-visible {
    outline: 2px solid
      color-mix(in oklab, var(--color-warning-500) 50%, transparent);
    outline-offset: 2px;
  }

  .update-badge--unmanaged .update-badge-count {
    color: var(--color-surface-950);
    background-color: var(--color-warning-400);
  }

  .unmanaged-mod-list {
    max-height: 16rem;
    overflow: auto;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .unmanaged-mod-item {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--color-surface-700);
    border-radius: var(--radius-base, 0.25rem);
    background-color: color-mix(
      in oklab,
      var(--color-surface-800) 80%,
      transparent
    );
  }

  .update-badge--active {
    box-shadow: inset 0 0 0 1px
      color-mix(in oklab, currentColor 35%, transparent);
  }

  .update-badge--deps.update-badge--active {
    background-color: color-mix(
      in oklab,
      var(--color-error-500) 18%,
      var(--color-surface-900)
    );
  }

  @media (prefers-reduced-motion: reduce) {
    .update-badge {
      transition: none;
    }
  }
</style>
