/** User-facing strings and small copy helpers (i18n fallbacks until translations load). */
export function formatUserError(error) {
  if (error && typeof error === "object") {
    const record = error;
    if (typeof record.message === "string" && record.message.length > 0) {
      return sanitizeNetworkError(record.message);
    }
  }
  const message = error instanceof Error ? error.message : String(error);
  if (!message || message === "[object Object]") {
    return "Something went wrong. Check your paths in Settings and try again.";
  }
  return sanitizeNetworkError(message);
}
function sanitizeNetworkError(message) {
  if (
    /could not resolve Nexus server|could not connect to Nexus/i.test(message)
  ) {
    return message;
  }
  if (/lookup|getaddrinfo|no such host/i.test(message)) {
    return "Could not reach the server (network or DNS error). Check your internet connection and try again.";
  }
  if (/dial tcp|connectex|connection refused|i\/o timeout/i.test(message)) {
    return "Could not connect to Nexus. If the website works in your browser, allow Junimo Hut through your firewall or configure your system proxy for desktop apps.";
  }
  return message;
}
export function modCount(n) {
  return n === 1 ? "1 mod" : `${n} mods`;
}
export function installSummary(installed, failed) {
  if (failed > 0) {
    return `Installed ${modCount(installed)}. ${modCount(failed)} failed — check the footer for details.`;
  }
  return `Installed ${modCount(installed)}.`;
}
/** Warmer install-complete line for the modal and footer. */
export function installCompleteLine(ok, fail) {
  if (fail > 0 && ok > 0) {
    return `Added ${modCount(ok)} to your library. ${fail} need a second look.`;
  }
  if (fail > 0) {
    return `Nothing installed — ${modCount(fail)} archive${fail === 1 ? "" : "s"} could not be unpacked.`;
  }
  if (ok === 1) return "One more mod for the collection.";
  return `${modCount(ok)} added to your library.`;
}
const EMPTY_LIBRARY_TIPS = [
  "Profiles remember which mods are on for each save — handy when you swap playstyles.",
  "Click a Tags cell to label a mod without opening its details.",
  "Use Check Updates to scan Nexus-linked mods before you launch.",
  "Drag an archive onto the list to queue it in the install dialog.",
];
export function emptyLibraryTip() {
  const day = Math.floor(Date.now() / 86_400_000);
  return EMPTY_LIBRARY_TIPS[day % EMPTY_LIBRARY_TIPS.length];
}
export function emptyLibraryState(searchQuery) {
  const q = searchQuery?.trim();
  if (q) {
    return {
      title: "No matching mods",
      hint: `Nothing matches "${q}". Try another search or change filters in Settings.`,
      tip: null,
    };
  }
  return {
    title: "No mods in the hut yet",
    hint: "Drop a .zip, .7z, or .rar here, or choose Install Mod above.",
    tip: emptyLibraryTip(),
  };
}
const LIBRARY_MILESTONES = {
  1: "First mod in the library. The hut's open for business.",
  10: "Ten mods — your load order is taking shape.",
  25: "Twenty-five mods. Profiles are starting to matter.",
  50: "Fifty mods under management. You know what you're doing.",
  100: "A hundred mods. The junimos are busy today.",
};
const MILESTONE_STORAGE_KEY = "jh-milestones-seen";
const MILESTONE_STORAGE_KEY_LEGACY = "sdvm-milestones-seen";
function readMilestonesSeen() {
  try {
    let raw = localStorage.getItem(MILESTONE_STORAGE_KEY);
    if (!raw) {
      raw = localStorage.getItem(MILESTONE_STORAGE_KEY_LEGACY);
      if (raw) {
        localStorage.setItem(MILESTONE_STORAGE_KEY, raw);
        localStorage.removeItem(MILESTONE_STORAGE_KEY_LEGACY);
      }
    }
    return new Set(raw ? JSON.parse(raw) : []);
  } catch {
    return new Set();
  }
}
export function consumeLibraryMilestone(count) {
  const message = LIBRARY_MILESTONES[count];
  if (!message) return null;
  try {
    const seen = readMilestonesSeen();
    if (seen.has(count)) return null;
    seen.add(count);
    localStorage.setItem(MILESTONE_STORAGE_KEY, JSON.stringify([...seen]));
    return message;
  } catch {
    return message;
  }
}
export function allEnabledMessage(count) {
  if (count >= 5)
    return `Every mod enabled — ${modCount(count)} ready for Pelican Town.`;
  return null;
}
export function launchSentMessage() {
  return "SMAPI's starting — see you in Pelican Town.";
}
export function updatesCheckedMessage(count, updatesFound) {
  if (updatesFound === 0) {
    return `Checked ${modCount(count)}. Everything matches the latest version.`;
  }
  return "Update check finished. Review the Status column for changes.";
}
const HUT_PROVERBS = [
  "Every mod folder tells a story. Yours is organized.",
  "Load order is peace of mind.",
  "The best save file is the one you actually play.",
  "A tidy Mods folder beats a perfect load order spreadsheet.",
];
export function hutProverb() {
  const index = Math.floor(Math.random() * HUT_PROVERBS.length);
  return HUT_PROVERBS[index];
}
export const aboutDialogTitle = "Junimo Hut";
export const aboutDialogTagline = "Mod manager for Stardew Valley.";
export const aboutDialogDisclaimer =
  "Not affiliated with ConcernedApe or Chucklefish.";
export const aboutCloseLabel = "Close";
const ARCHIVE_PATTERN = /\.(zip|7z|rar)$/i;
export function isArchivePath(path) {
  return ARCHIVE_PATTERN.test(path.trim());
}
export function pathBasename(path) {
  const normalized = path.replace(/\\/g, "/");
  const base = normalized.split("/").pop();
  return base && base.length > 0 ? base : path;
}
export const contextMenuReinstallSavedLabel = "Reinstall from saved download";
export const reinstallSavedConfirmTitle = "Reinstall from saved download?";
export const reinstallSavedConfirmLabel = "Reinstall";
export const reinstallSavedDeleteOldLabel =
  "Remove old mod files first (keeps config.json and manifest.json)";
export const reinstallSavedMissingError =
  "No saved download found for this mod.";
export function reinstallSavedConfirmMessage(modName, archivePath) {
  return `Reinstall “${modName}” from ${pathBasename(archivePath)}? Your mod folder will be replaced with the saved archive.`;
}
export function reinstallSavedSuccessLine(modName) {
  return `Reinstalled ${modName} from saved download.`;
}
export const windowMinimizeLabel = "Minimize window";
export const windowMaximizeLabel = "Maximize window";
export const windowRestoreLabel = "Restore window";
export const windowCloseLabel = "Close window";
export const statusDismissLabel = "Dismiss status message";
/** Shown in the frameless chrome row (matches build/config.yml info.version). */
export function appVersionLabel() {
  return "v0.1.0";
}
export const brandWordmark = "Junimo Hut";
export const brandWordmarkTitle = "Junimo Hut — Mod manager for Stardew Valley";
export const setupWelcomeTitle = "Welcome to Junimo Hut";
export const setupWelcomeBody =
  "Tell Junimo Hut where Stardew Valley lives and where to store mod files. Enabled mods are symlinked into your game's Mods folder.";
export const settingsNexusHint =
  "Paste an API key from your Nexus Mods account to download updates in Junimo Hut.";
export const searchModsPlaceholder = "Search mods…";
export const searchModsAria = "Search mods";
export const downloadingModFromNexus = "Downloading mod from Nexus…";
export const downloadNoArchivePath =
  "Download finished but no archive path was returned.";
export const downloadCompleteReview =
  "Download complete. Review the install dialog.";
export const downloadsPaneTitle = "Downloads";
export const downloadsPaneCloseAria = "Close downloads panel";
export const downloadsActiveSectionLabel = "In progress";
export const downloadsHistorySectionLabel = "Saved archives";
export const downloadsSearchPlaceholder = "Search by mod or file name…";
export const downloadsEmptyHistory =
  "No saved archives yet. Finished Nexus downloads appear here for install or reinstall.";
export const downloadsActiveEmpty =
  "Nothing downloading. Nexus downloads show progress here.";
export const downloadsLoadingAria = "Loading download history";
export const downloadsRefreshingAria = "Refreshing downloads";
export function downloadsProgressAria(modName) {
  return `Download progress for ${modName}`;
}
export const downloadsLoadError =
  "Couldn't load saved archives. Check Settings paths or your connection, then try again.";
export const downloadsLoadRetry = "Try again";
export const downloadUnknownModLabel = "Unknown mod";
export const downloadsUnlinkedBadge = "Unlinked";
export const downloadsUnlinkedTitle = "Not linked to a mod in your library";
export const downloadsUnlinkedHint =
  "Install once so future archives match by manifest UniqueID or Nexus mod ID.";
export const downloadsViewInLibrary = "View in library";
export const downloadsRowDetailsShow = "Show archive details";
export const downloadsRowDetailsHide = "Hide archive details";
export const downloadsBulkLearnHint =
  "Ctrl+click to select · Shift+click for a range · Enter on a focused row installs · Esc to clear";
export const downloadsBulkClearSelection = "Clear selection";
export const downloadsBulkRangeHint =
  "Shift+click another row to add to the selection";
export const learnHintDismissLabel = "Dismiss";
export function downloadsSelectionCount(count) {
  return count === 1 ? "1 selected" : `${count} selected`;
}
export function downloadsBulkInstallLabel(count) {
  return count === 1 ? "Install selected" : `Install ${count} selected`;
}
export function downloadsBulkReinstallLabel(count) {
  return count === 1 ? "Reinstall selected" : `Reinstall ${count} selected`;
}
export const downloadsBulkReinstallConfirmTitle =
  "Reinstall selected archives?";
export function downloadsBulkReinstallConfirmMessage(count) {
  return `Reinstall ${modCount(count)} from saved archives? Each mod folder will be replaced with its saved archive.`;
}
export function downloadsUniqueIdLine(uniqueId) {
  return `Unique ID: ${uniqueId}`;
}
export function downloadsNexusIdLine(nexusModId) {
  return `Nexus mod ${nexusModId}`;
}
export function downloadsModLibraryLine(mod) {
  const author = mod.manifest?.Author?.trim();
  const version = mod.manifest?.Version?.trim();
  if (author && version) return `${author} · v${version}`;
  if (author) return author;
  if (version) return `v${version}`;
  return null;
}
export const downloadsActionInstall = "Install";
export const downloadsActionReinstall = "Reinstall";
export const downloadsActionShowFolder = "Show in folder";
export const downloadsActionDelete = "Delete archive";
export const downloadsActionOpenNexus = "Open on Nexus";
export const downloadsResizeAria = "Resize downloads panel";
export function downloadsRowMoreAriaFor(displayName) {
  return `More actions for ${displayName}`;
}
export function downloadsSearchEmpty(query) {
  return `No matches for “${query}”.`;
}
export const deleteDownloadConfirmTitle = "Delete saved archive?";
export function deleteDownloadConfirmMessage(displayName, fileName) {
  return `“${fileName}”${displayName !== downloadUnknownModLabel ? ` for ${displayName}` : ""} will be permanently deleted from your downloads folder.`;
}
export function deletedDownloadMessage(displayName) {
  return `Deleted archive for ${displayName}.`;
}
export const launchingSmapi = "Launching SMAPI…";
export const checkingModUpdates = "Checking for mod updates…";
export const modEndorsedOnNexus = "Mod endorsed on Nexus Mods.";
export const noNexusUpdateKey = "No Nexus update key for this mod.";
export const tagAddedToMod = "Tag added to mod.";
export const tagRemovedFromMod = "Tag removed from mod.";
export const nexusKeyRequired = "Paste a Nexus Mods API key before connecting.";
export const nexusConnectedMessage = "Connected to Nexus Mods.";
export const nexusKeyRejected =
  "Nexus API key was rejected. Copy a valid key from your Nexus account settings.";
export const registerNxmSuccess =
  "Registered nxm:// links to open in Junimo Hut.";
export const smapiInstallerOpened =
  "Opened the SMAPI installer. Run it to install or update SMAPI.";
export const filterCleared = "Filter cleared.";
export function bulkToggleStatus(enabled, count) {
  return `${enabled ? "Enabled" : "Disabled"} ${modCount(count)}`;
}
export function updatedModMessage(name) {
  return `Updated ${name}.`;
}
export function tagsAppliedMessage(count) {
  return `Tags applied to ${modCount(count)}.`;
}
export function downloadingUpdateForMessage(name) {
  return `Downloading update for ${name}…`;
}
export function updateDownloadedForMessage(name) {
  return `Update downloaded for ${name}. Review the install dialog.`;
}
export function createdTagMessage(name) {
  return `Created tag “${name}”.`;
}
export function deletedModMessage(name) {
  return `Deleted ${name}.`;
}
export const deleteModConfirmTitle = "Delete mod?";
export function deleteModConfirmMessage(name) {
  return `“${name}” will be removed from your Mods folder. This cannot be undone.`;
}
export const deleteModConfirmLabel = "Delete mod";
export const deleteModsBatchConfirmTitle = "Delete selected mods?";
export function deleteModsBatchConfirmMessage(count) {
  return `${modCount(count)} will be removed from your Mods folder. This cannot be undone.`;
}
export const deleteModsBatchConfirmLabel = "Delete mods";
export function gridBulkDeleteLabel(count) {
  return count === 1 ? "Delete selected" : `Delete ${count} selected`;
}
export const deleteModDeleteArchiveLabel = "Also delete saved archive";
export function deleteModDeleteArchiveHint(count) {
  return count === 1
    ? "Applies to 1 mod with a saved archive"
    : `Applies to ${count} mods with saved archives`;
}
export const deleteModDeleteArchiveNoneHint =
  "No selected mods have a saved archive";
export function deletedModsMessage(deleted, archivesDeleted) {
  if (archivesDeleted > 0) {
    return `Deleted ${modCount(deleted)} and ${archivesDeleted} saved archive${archivesDeleted === 1 ? "" : "s"}.`;
  }
  return `Deleted ${modCount(deleted)}.`;
}
export function deletedTagMessage(name) {
  return `Deleted tag “${name}”.`;
}
export function deletedProfileMessage(name) {
  return `Deleted profile “${name}”.`;
}
export function renamedTagMessage(name) {
  return `Renamed tag to "${name}".`;
}
export function renamedModMessage(name) {
  return `Display name set to "${name}".`;
}
export function clearedModDisplayNameMessage() {
  return "Display name cleared.";
}
export const modOfficialNameLabel = "Official name";
export const modDisplayNameLabel = "Display name";
export const modDisplayNamePlaceholder = "Same as official name";
export const modRenameLabel = "Rename display name…";
export const modClearDisplayName = "Reset display name";
export const modDisplayNameAria = "Custom display name for mod list";
export function modClearDisplayNameAria(name) {
  return `Clear display name for ${name}`;
}
export const settingsSectionPaths = "Paths";
export function missingDependencyBadge(count, issues) {
  const hasVersionIssue = issues.some((i) => i.state === "version_too_low");
  const hasDisabled = issues.some((i) => i.state === "disabled");
  if (count === 1) {
    if (hasVersionIssue) return "Dependency version too low";
    if (hasDisabled) return "Dependency disabled";
    return "Missing dependency";
  }
  if (hasVersionIssue || hasDisabled) return `${count} dependency issues`;
  return `${count} missing deps`;
}
export function dependencyIssuesTooltip(issues) {
  return issues
    .map((i) => {
      if (i.state === "version_too_low")
        return `${i.uniqueID} (version too low)`;
      if (i.state === "disabled") return `${i.uniqueID} (disabled)`;
      return i.uniqueID;
    })
    .join(", ");
}
export function dependencyIssueCountLabel(count) {
  return count === 1 ? "1 dependency issue" : `${count} dependency issues`;
}
export function unmanagedModCountLabel(count) {
  return count === 1 ? "1 unmanaged mod" : `${count} unmanaged mods`;
}
export function unmanagedModsDialogTitle() {
  return "Unmanaged mods in game folder";
}
export function unmanagedModsDialogMessage() {
  return "These folders live in your Stardew Valley Mods directory but are not managed by Junimo Hut. They can cause duplicate installs or SMAPI launch errors if they overlap with your library.";
}
export function unmanagedModsOpenFolderLabel() {
  return "Open Mods folder";
}
export function unmanagedModsDismissLabel() {
  return "Got it";
}
export function dependencyNotInstalled() {
  return "Not installed";
}
export function dependencyVersionTooLow() {
  return "Version too low";
}
export function dependencyInstalled() {
  return "Installed";
}
export function dependencyDisabled() {
  return "Disabled";
}
export function dependencyLoadOrderLabel() {
  return "load order";
}
export function dependencyOptionalAbsent() {
  return "Not installed";
}
export function dependencySearchNexus() {
  return "Search Nexus";
}
export function dependencyOpenNexus() {
  return "Open on Nexus";
}
export function dependencyEnableMod() {
  return "Enable dependency";
}
export function dependentViewMod() {
  return "View mod";
}
export function dependentsSummary(count) {
  return count === 1 ? "1 dependent mod" : `${count} dependent mods`;
}
export function installDependencyWarningTitle() {
  return "Missing dependencies detected";
}
export function installDependencyWarningBody(count) {
  return count === 1
    ? "This mod declares dependencies that are not satisfied in your library. You can still install it, but it may not work until you add them."
    : "Some mods in this queue declare dependencies that are not satisfied in your library. You can still install, but they may not work until you add them.";
}
export function installAnywayLabel() {
  return "Install anyway";
}
export function installOverwriteWarningTitle() {
  return "No manifest — looks like a file patch";
}
export function installOverwriteWarningBody(fileCount) {
  return fileCount === 1
    ? "This archive has no manifest.json. Its files match paths inside an installed mod and are usually meant to overwrite files there, not install as a new mod."
    : "Some archives have no manifest.json. Their files match paths inside installed mods and are usually meant to overwrite files there, not install as new mods.";
}
export function installOverwriteConfirmLabel() {
  return "Merge into mod";
}
export function installOverwriteTargetLegend() {
  return "Merge target";
}
export function installOverwriteTargetHint() {
  return "Choose which installed mod folder should receive these files.";
}
export function installOverwriteMatchSummary(matched, total) {
  return `${matched} of ${total} files match this mod`;
}
export function installOverwriteSamplePathsLabel() {
  return "Sample paths";
}
export function modContainsOverwritesLabel() {
  return "Contains overwrites";
}
export function modContainsOverwritesTooltip() {
  return "File patches were merged into this mod through Junimo Hut. Its files may differ from a clean install.";
}
export function installSuggestedTagsHint(count) {
  return count === 1
    ? "One tag was suggested from mod metadata. Change or clear before installing."
    : `${count} tags were suggested from mod metadata. Change or clear before installing.`;
}
export function dependencyIssuesFooterMessage(count) {
  return count === 1
    ? "1 mod has dependency issues — review the Status column."
    : `${count} mods have dependency issues — review the Status column.`;
}
export function updatesFilterFooterMessage(count) {
  return count === 1
    ? "Showing 1 mod with an update available."
    : `Showing ${count} mods with updates available.`;
}
export function gridUpdatesFilterLabel(count) {
  return count === 1 ? "1 update available" : `${count} updates available`;
}
export function gridDependencyFilterLabel(count) {
  return count === 1 ? "1 dependency issue" : `${count} dependency issues`;
}
export function gridUpdatesFilterEmptyTitle() {
  return "No mods with updates available";
}
export function gridUpdatesFilterEmptyHint() {
  return "Clear the filter or run Check Updates from the toolbar.";
}
export function gridDependencyFilterEmptyTitle() {
  return "No mods with dependency issues";
}
export function gridDependencyFilterEmptyHint() {
  return "Clear the filter to see your full library.";
}
export const gridTagsFilterEmptyTitle = "No mods match these tags";
export const gridTagsFilterEmptyHint =
  "Turn tags back on in the sidebar, or assign tags to mods.";
export const gridClearFilter = "Clear filter";
export const gridQuickStartLabel = "Quick start";
export const gridUpdatesFilterMeta =
  "Showing mods with an update in the Status column";
export const gridDependencyFilterMeta =
  "Showing mods with missing or unsatisfied dependencies";
export const gridTagsFilteringMeta =
  "Hides rows only — enabled mods and SMAPI are unchanged";
export function gridTagsFilteringBadge(count) {
  return count === 1 ? "1 tag filtering list" : `${count} tags filtering list`;
}
export function gridBulkSelectedLabel(count) {
  return count === 1 ? "1 selected" : `${count} selected`;
}
export const gridBulkEnableSelected = "Enable selected";
export const gridBulkDisableSelected = "Disable selected";
export const gridBulkClearSelection = "Clear selection";
export const gridBulkShiftHint =
  "Shift+click another row to add to the selection";
export const gridBulkKeyboardHint =
  "Ctrl+click to select multiple mods · Shift+click for a range · Esc to clear";
export const gridSortClearedAnnounce = "Sort cleared";
export const gridSelectionClearedAnnounce = "Selection cleared";
export function gridBulkDeleteOpeningAnnounce(count) {
  return count === 1
    ? "Opening delete for 1 mod"
    : `Opening delete for ${count} mods`;
}
export const contextMenuOpenFolder = "Open mod folder";
export const contextMenuOpenManifest = "Open manifest.json";
export const contextMenuEditConfig = "Edit configs";
export const contextMenuViewNexus = "View on Nexus Mods";
export const contextMenuEndorse = "Endorse on Nexus Mods";
export const contextMenuDownloadUpdate = "Download update";
export const contextMenuDeleteMod = "Delete mod…";
export const dialogCancelLabel = "Cancel";
export function dependenciesMissingSummary(count) {
  return count === 1 ? "1 dependency missing" : `${count} dependencies missing`;
}
export const settingsSectionLibrary = "Library";
export const settingsSectionAppearance = "Appearance";
export const installDisplayNameLegend = "Display names in mod list";
export const installDisplayNameHint =
  "Some archives include multiple CP/AT variants that share the same manifest name. Choose how they should appear after install.";
export const installDisplayNameOfficial = "Official manifest names";
export const installDisplayNameFolder = "Folder names (Junimo Hut split)";
export const installDisplayNamePreviewLabel = "Will install as";
export const themeStardewDarkLabel = "Stardew Dark";
export const themeStardewLightLabel = "Stardew Light";
export const themeCerberusLabel = "Cerberus";
export const themeMonaLabel = "Mona";
export const themeVoxLabel = "Vox";
export const settingsSectionNexus = "Nexus Mods";
export const modGridColumnsTitle = "Visible columns";
export const modGridColumnsRequiredHint = "Name is always shown.";
export const settingsBrowseLabel = "Browse…";
export const settingsPathsHint =
  "Enabled mods are symlinked from your mod library into the game Mods folder.";
export const settingsSavePaths = "Save paths";
export const settingsDone = "Done";
export const settingsUnsavedPaths = "Unsaved path changes";
export const settingsInstallSmapi = "Install or update SMAPI";
export const settingsOpeningSmapi = "Opening installer…";
/** First-run workspace guidance (dismissible banner). */
export function workspaceOnboardingText() {
  return "Profiles switch which mods are enabled per playstyle. Click a Tags cell to assign labels. Drop archives on the list or use Install Mod.";
}
export const gridTagsLearnEmphasis = "Click any Tags cell";
export const gridTagsLearnHint =
  "to assign labels · Filter the list with tag toggles in the sidebar";
export const tagsCellAddLabel = "Add tags";
/** Left tag pane — sidebar filter / create / manage */
export const tagsSidebarAria = "Mod tags";
export const tagsSidebarTitle = "Tags";
export const tagsSidebarNew = "New";
export const tagsSidebarCancel = "Cancel";
export const tagsSidebarTagNameLabel = "Tag name";
export const tagsSidebarTagNamePlaceholder = "e.g. Quality of life";
export const tagsSidebarColorLegend = "Color";
export const tagsSidebarColorGroupLabel = "Tag color";
export const tagsSidebarCreateTag = "Create tag";
export const tagsSidebarCreating = "Creating…";
export const tagsSidebarEmptyTitle = "No tags yet";
export const tagsSidebarEmptyHint =
  "Tags label mods and filter the list. Click a Tags cell in the grid to assign labels.";
export const tagsSidebarCreateFirst = "Create your first tag";
export const tagsSidebarFooterHint =
  "Toggle filters to narrow the mod list. Click a Tags cell to assign labels.";
export const tagsSidebarAllShown = "All mods shown";
export const tagsSidebarShowAll = "Show all";
export const tagsRenameLabel = "Rename tag";
export const tagsDeleteLabel = "Delete tag";
export const tagsResizeAria = "Resize tags sidebar";
export const tagsSidebarHideTitle = "Hide Tags";
export const tagsSidebarHideAria = "Hide Tags sidebar";
export function tagsSidebarShowTitle(activeFilters, narrowed) {
  if (narrowed) {
    return activeFilters === 1
      ? "Show Tags — 1 filter active"
      : `Show Tags — ${activeFilters} filters active`;
  }
  return "Show Tags";
}
export function tagsSidebarShowAria(activeFilters, narrowed) {
  if (narrowed) {
    return activeFilters === 1
      ? "Show Tags sidebar, 1 filter active"
      : `Show Tags sidebar, ${activeFilters} filters active`;
  }
  return "Show Tags sidebar";
}
export function tagsSidebarFilterMeta(count) {
  return count === 1 ? "1 filter active" : `${count} filters active`;
}
export function tagsFilterToggleTitle(name, active) {
  return active
    ? `Stop filtering by ${name}`
    : `Filter list to mods tagged ${name}`;
}
export function tagsFilterToggleAria(name, active) {
  return active ? `Remove ${name} from list filter` : `Filter list by ${name}`;
}
export function tagsRenameAria(name) {
  return `Rename tag ${name}`;
}
export function tagsDeleteAria(name) {
  return `Delete tag ${name}`;
}
export function tagColorAria(label) {
  return `${label} tag color`;
}
/** Stardew-adjacent tag chip presets (hex + accessible name). */
export const TAG_COLOR_PRESETS = [
  { hex: "#4a7c59", label: "Forest" },
  { hex: "#6b8e4e", label: "Grass" },
  { hex: "#c9a227", label: "Gold" },
  { hex: "#8b6914", label: "Wheat" },
  { hex: "#5b8a8a", label: "Water" },
  { hex: "#a0522d", label: "Wood" },
  { hex: "#c45c5c", label: "Berry" },
  { hex: "#64748b", label: "Stone" },
];
export function tagsCellEditLabel(modName, tagNames) {
  if (tagNames.length === 0) return `Add tags to ${modName}`;
  return `Edit tags for ${modName}: ${tagNames.join(", ")}`;
}
export function tagsOverflowLabel(count, names) {
  return count === 1 ? `1 more tag: ${names}` : `${count} more tags: ${names}`;
}
/** Custom dropdown list */
export const dropdownEmptyOptions = "No options";
export const dropdownSelectPlaceholder = "Select…";
export const dropdownTypeaheadIdleHint = "Type a letter to jump to an option";
export function dropdownTypeaheadJumpHint(query) {
  return `Jump to… ${query}`;
}
export const toolbarProfileAria = "Active profile";
export const settingsHideDisabledOptions = [
  { value: "none", label: "Show all mods" },
  { value: "enabled", label: "Hide disabled mods" },
  { value: "disabled", label: "Show only disabled mods" },
];
export function themeDropdownOptions() {
  return [
    { value: "stardew-dark", label: themeStardewDarkLabel },
    { value: "stardew-light", label: themeStardewLightLabel },
    { value: "cerberus", label: themeCerberusLabel },
    { value: "mona", label: themeMonaLabel },
    { value: "vox", label: themeVoxLabel },
  ];
}
export function normalizeArchivePaths(paths) {
  const seen = new Set();
  const out = [];
  for (const raw of paths) {
    const path = raw.trim();
    if (!path || !isArchivePath(path) || seen.has(path)) continue;
    seen.add(path);
    out.push(path);
  }
  return out;
}

export function configEditorWindowTitle(modName, fileName) {
  const name = modName.trim() || "Mod";
  const file = fileName?.trim() || "config.json";
  return `${name} — ${file}`;
}

export const configEditorOpen = "Edit configs";
export const configEditorEditConfig = "Edit configs";
export const configEditorSidebarModsHeading = "Mods";
export const configEditorSidebarFilesHeading = "Files";
export const configEditorSearchModsPlaceholder = "Search mods…";
export const configEditorNoModsWithJson = "No mods with JSON files in your library.";
export const configEditorEmptyLibraryHint =
  "Install mods that include config.json or other .json files, then choose Edit configs from a mod in Junimo Hut.";
export const configEditorSelectModHint =
  "Select a mod above to browse its JSON files.";
export const configEditorNoJsonInMod = "No JSON files in this mod folder.";
export const configEditorSaveAndSwitch = "Save and switch";
export const configEditorUnsavedFileSwitchTitle = "Switch file without saving?";
export const configEditorUnsavedFileSwitchBody =
  "Your edits have not been saved. Discard them and open the other file?";
export const configEditorLoadingFile = "Loading file…";
export const configEditorSave = "Save";
export const configEditorDiscard = "Discard";
export const configEditorSaving = "Saving…";
export const configEditorValidJson = "Valid JSONC";
export const configEditorInvalidJson = "Invalid JSONC";
export const configEditorUnsaved = "Unsaved changes";
export const configEditorSaved = "Config saved";
export const configEditorOpenExternal = "Open in external editor";
export const configEditorNoConfig = "This mod has no config.json yet.";
export const configEditorModMissing = "This mod is no longer in your library.";
export const configEditorLoadFailed = "Could not load this file.";

export function configEditorLoadFailedFor(path) {
  const file = path.trim();
  return file ? `Could not load ${file}.` : configEditorLoadFailed;
}

export const configEditorLoadingMods = "Loading mods…";
export const configEditorTitleFallback = "Config editor";
export const configEditorLoadingFileAria = "Loading config file";
export const configEditorJsoncHint =
  "Comments and trailing commas are allowed.";

export function jsoncParseErrorMessage(code) {
  switch (code) {
    case 1:
      return "Invalid character here.";
    case 2:
      return "This number is not formatted correctly.";
    case 3:
      return "Expected a property name in quotes.";
    case 4:
      return "Expected a value here.";
    case 5:
      return "Expected a colon after the property name.";
    case 6:
      return "Expected a comma between items.";
    case 7:
      return "Expected a closing brace `}`.";
    case 8:
      return "Expected a closing bracket `]`.";
    case 9:
      return "Unexpected end of file.";
    case 10:
      return "This comment is not valid here.";
    case 11:
      return "Comment was not closed.";
    case 12:
      return "String was not closed.";
    case 13:
      return "Number was not finished.";
    case 14:
      return "Invalid Unicode escape in string.";
    case 15:
      return "Invalid escape sequence in string.";
    case 16:
      return "Invalid character in JSON.";
    default:
      return "Invalid JSONC syntax.";
  }
}
export const configEditorMissingModId = "No mod selected for editing.";

export function configEditorParseError(line, column, message) {
  if (line > 0 && column > 0) {
    return `Line ${line}, column ${column}: ${message}`;
  }
  return message;
}

export function configEditorProfileBanner(profileName) {
  const name = profileName.trim() || "Active profile";
  return `Profile-specific configs enabled · ${name}`;
}

export function configEditorSaveFailed(reason) {
  return reason.trim()
    ? `Could not save config: ${reason}`
    : "Could not save config.";
}

export const configEditorUnsavedCloseTitle = "Discard unsaved changes?";
export const configEditorUnsavedCloseBody =
  "Your edits have not been saved. Discard them and close the editor?";

export const configEditorUnsavedSwitchTitle = "Switch mod without saving?";
export const configEditorUnsavedSwitchBody =
  "Your edits have not been saved. Discard them and open the other mod's config?";

export const configEditorUnsavedProfileTitle = "Switch profile without saving?";
export const configEditorUnsavedProfileBody =
  "The config editor has unsaved changes. Discard them and switch profiles?";

export const configEditorDiscardConfirmTitle = "Discard changes?";
export const configEditorDiscardConfirmBody =
  "Reload the file from disk and lose your unsaved edits?";
//# sourceMappingURL=copy.js.map
