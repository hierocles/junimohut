import * as m from "$lib/paraglide/messages.js";
import { pathBasename } from "$lib/paths";
import { ParseErrorCode } from "$lib/mods/jsoncEnums";

export function modCount(n: number): string {
  return m.mod_count({ count: n });
}

export function installSummary(installed: number, failed: number): string {
  if (failed > 0) {
    return m.install_summary_with_fail({
      installed: modCount(installed),
      failed: modCount(failed),
    });
  }
  return m.install_summary_ok({ installed: modCount(installed) });
}

export function installCompleteLine(ok: number, fail: number): string {
  if (fail > 0 && ok > 0) {
    return m.install_complete_partial({ ok: modCount(ok), fail: String(fail) });
  }
  if (fail > 0) {
    return m.install_complete_none({
      archives: m.archive_count({ count: fail }),
    });
  }
  if (ok === 1) return m.install_complete_one_mod();
  return m.install_complete_added({ count: modCount(ok) });
}

const EMPTY_LIBRARY_TIP_KEYS = [
  m.empty_library_tip_0,
  m.empty_library_tip_1,
  m.empty_library_tip_2,
  m.empty_library_tip_3,
] as const;

export function emptyLibraryTip(): string {
  const day = Math.floor(Date.now() / 86_400_000);
  return EMPTY_LIBRARY_TIP_KEYS[day % EMPTY_LIBRARY_TIP_KEYS.length]();
}

export function emptyLibraryState(searchQuery?: string) {
  const q = searchQuery?.trim();
  if (q) {
    return {
      title: m.no_matching_mods(),
      hint: m.no_matching_mods_hint({ query: q }),
      tip: null as string | null,
    };
  }
  return {
    title: m.empty_library_title(),
    hint: m.empty_library_hint(),
    tip: emptyLibraryTip(),
  };
}

const LIBRARY_MILESTONE_MESSAGES: Record<number, () => string> = {
  1: m.library_milestone_1,
  10: m.library_milestone_10,
  25: m.library_milestone_25,
  50: m.library_milestone_50,
  100: m.library_milestone_100,
};

const MILESTONE_STORAGE_KEY = "jh-milestones-seen";
const MILESTONE_STORAGE_KEY_LEGACY = "sdvm-milestones-seen";

function readMilestonesSeen(): Set<number> {
  try {
    let raw = localStorage.getItem(MILESTONE_STORAGE_KEY);
    if (!raw) {
      raw = localStorage.getItem(MILESTONE_STORAGE_KEY_LEGACY);
      if (raw) {
        localStorage.setItem(MILESTONE_STORAGE_KEY, raw);
        localStorage.removeItem(MILESTONE_STORAGE_KEY_LEGACY);
      }
    }
    return new Set(raw ? (JSON.parse(raw) as number[]) : []);
  } catch {
    return new Set();
  }
}

export function consumeLibraryMilestone(count: number): string | null {
  const messageFn = LIBRARY_MILESTONE_MESSAGES[count];
  if (!messageFn) return null;
  const message = messageFn();
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

export function allEnabledMessage(count: number): string | null {
  if (count >= 5) return m.all_enabled_message({ count: modCount(count) });
  return null;
}

export function launchSentMessage(): string {
  return m.launch_sent_message();
}

export function updatesCheckedMessage(
  count: number,
  updatesFound: number,
): string {
  if (updatesFound === 0) {
    return m.updates_checked_none({ count: modCount(count) });
  }
  return m.updates_checked_found();
}

const HUT_PROVERB_KEYS = [
  m.hut_proverb_0,
  m.hut_proverb_1,
  m.hut_proverb_2,
  m.hut_proverb_3,
] as const;

export function hutProverb(): string {
  const index = Math.floor(Math.random() * HUT_PROVERB_KEYS.length);
  return HUT_PROVERB_KEYS[index]();
}

export function appVersionLabel(): string {
  return "v0.1.0";
}

export function reinstallSavedConfirmMessage(
  modName: string,
  archivePath: string,
): string {
  return m.reinstall_saved_confirm_message({
    modName,
    archiveName: pathBasename(archivePath),
  });
}

export function reinstallSavedSuccessLine(modName: string): string {
  return m.reinstall_saved_success_line({ modName });
}

export function downloadsProgressAria(modName: string): string {
  return m.downloads_progress_aria({ modName });
}

export function downloadsSelectionCount(count: number): string {
  return m.selection_count({ count });
}

export function downloadsBulkInstallLabel(count: number): string {
  return count === 1
    ? m.downloads_bulk_install_one()
    : m.downloads_bulk_install_other({ count });
}

export function downloadsBulkReinstallLabel(count: number): string {
  return count === 1
    ? m.downloads_bulk_reinstall_one()
    : m.downloads_bulk_reinstall_other({ count });
}

export function downloadsBulkReinstallConfirmMessage(count: number): string {
  return m.downloads_bulk_reinstall_confirm_message({ count: modCount(count) });
}

export function downloadsUniqueIdLine(uniqueId: string): string {
  return m.downloads_unique_id_line({ uniqueId });
}

export function downloadsNexusIdLine(nexusModId: number): string {
  return m.downloads_nexus_id_line({ nexusModId });
}

export function downloadsModLibraryLine(mod: {
  manifest?: { Author?: string; Version?: string; Name?: string } | null;
}): string | null {
  const author = mod.manifest?.Author?.trim();
  const version = mod.manifest?.Version?.trim();
  if (author && version) return `${author} · v${version}`;
  if (author) return author;
  if (version) return `v${version}`;
  return null;
}

export function downloadsRowMoreAriaFor(displayName: string): string {
  return m.downloads_row_more_aria({ displayName });
}

export function downloadsSearchEmpty(query: string): string {
  return m.downloads_search_empty({ query });
}

export function deleteDownloadConfirmMessage(
  displayName: string,
  fileName: string,
): string {
  const forMod =
    displayName !== m.download_unknown_mod_label()
      ? m.delete_download_confirm_for_mod({ displayName })
      : "";
  return m.delete_download_confirm_message({ fileName, forMod });
}

export function deletedDownloadMessage(displayName: string): string {
  return m.deleted_download_message({ displayName });
}

export function bulkToggleStatus(enabled: boolean, count: number): string {
  return m.bulk_toggle_status({
    action: enabled ? m.bulk_toggle_enabled() : m.bulk_toggle_disabled(),
    count: modCount(count),
  });
}

export function updatedModMessage(name: string): string {
  return m.updated_mod_message({ name });
}

export function tagsAppliedMessage(count: number): string {
  return m.tags_applied_message({ count: modCount(count) });
}

export function downloadingUpdateForMessage(name: string): string {
  return m.downloading_update_for_message({ name });
}

export function updateDownloadedForMessage(name: string): string {
  return m.update_downloaded_for_message({ name });
}

export function createdTagMessage(name: string): string {
  return m.created_tag_message({ name });
}

export function deletedModMessage(name: string): string {
  return m.deleted_mod_message({ name });
}

export function deleteModConfirmMessage(name: string): string {
  return m.delete_mod_confirm_message({ name });
}

export function deleteModsBatchConfirmMessage(count: number): string {
  return m.delete_mods_batch_confirm_message({ count: modCount(count) });
}

export function gridBulkDeleteLabel(count: number): string {
  return count === 1
    ? m.grid_bulk_delete_one()
    : m.grid_bulk_delete_other({ count });
}

export function gridBundlePartsLabel(count: number): string {
  return m.parts_count({ count });
}

export function deleteBundleConfirmMessage(
  name: string,
  count: number,
): string {
  return m.delete_bundle_confirm_message({ name, count });
}

export function deleteModDeleteArchiveHint(count: number): string {
  return count === 1
    ? m.delete_mod_delete_archive_hint_one()
    : m.delete_mod_delete_archive_hint_other({ count });
}

export function deletedModsMessage(
  deleted: number,
  archivesDeleted: number,
): string {
  if (archivesDeleted > 0) {
    return m.deleted_mods_with_archives({
      deleted: modCount(deleted),
      archives: String(archivesDeleted),
      archiveSuffix: archivesDeleted === 1 ? "" : "s",
    });
  }
  return m.deleted_mods_only({ deleted: modCount(deleted) });
}

export function deletedTagMessage(name: string): string {
  return m.deleted_tag_message({ name });
}

export function deletedProfileMessage(name: string): string {
  return m.deleted_profile_message({ name });
}

export function renamedTagMessage(name: string): string {
  return m.renamed_tag_message({ name });
}

export function renamedModMessage(name: string): string {
  return m.renamed_mod_message({ name });
}

export function clearedModDisplayNameMessage(): string {
  return m.cleared_mod_display_name_message();
}

export function modClearDisplayNameAria(name: string): string {
  return m.mod_clear_display_name_aria({ name });
}

export function missingDependencyBadge(
  count: number,
  issues: Array<{ state: string }>,
): string {
  const hasVersionIssue = issues.some((i) => i.state === "version_too_low");
  const hasDisabled = issues.some((i) => i.state === "disabled");
  if (count === 1) {
    if (hasVersionIssue) return m.missing_dependency_badge_one_version();
    if (hasDisabled) return m.missing_dependency_badge_one_disabled();
    return m.missing_dependency_badge_one();
  }
  if (hasVersionIssue || hasDisabled) {
    return m.missing_dependency_badge_many_issues({ count });
  }
  return m.missing_dependency_badge_many({ count });
}

export function dependencyIssuesTooltip(
  issues: Array<{ uniqueID: string; state: string }>,
): string {
  return issues
    .map((i) => {
      if (i.state === "version_too_low") {
        return m.dependency_issue_version({ uniqueId: i.uniqueID });
      }
      if (i.state === "disabled") {
        return m.dependency_issue_disabled({ uniqueId: i.uniqueID });
      }
      return i.uniqueID;
    })
    .join(", ");
}

export function dependencyIssueCountLabel(count: number): string {
  return count === 1
    ? m.dependency_issue_count_one()
    : m.dependency_issue_count_other({ count });
}

export function unmanagedModCountLabel(count: number): string {
  return count === 1
    ? m.unmanaged_mod_count_one()
    : m.unmanaged_mod_count_other({ count });
}

export function unmanagedModsDialogTitle(): string {
  return m.unmanaged_mods_dialog_title();
}

export function unmanagedModsDialogMessage(): string {
  return m.unmanaged_mods_dialog_message();
}

export function unmanagedModsOpenFolderLabel(): string {
  return m.unmanaged_mods_open_folder_label();
}

export function unmanagedModsDismissLabel(): string {
  return m.unmanaged_mods_dismiss_label();
}

export function duplicateModCountLabel(count: number): string {
  return count === 1
    ? m.duplicate_mod_count_one()
    : m.duplicate_mod_count_other({ count });
}

export function duplicateModsDialogTitle(): string {
  return m.duplicate_mods_dialog_title();
}

export function duplicateModsDialogMessage(): string {
  return m.duplicate_mods_dialog_message();
}

export function duplicateModsCleanupLabel(): string {
  return m.duplicate_mods_cleanup_label();
}

export function duplicateModsDismissLabel(): string {
  return m.duplicate_mods_dismiss_label();
}

export function duplicateModsKeepLabel(): string {
  return m.duplicate_mods_keep_label();
}

export function duplicateModsCleanupSuccess(count: number): string {
  return count === 1
    ? m.duplicate_mods_cleanup_success_one()
    : m.duplicate_mods_cleanup_success_other({ count });
}

export function installOverwriteExistingModHint(): string {
  return m.install_overwrite_existing_mod_hint();
}

export function installOverwriteExistingModMergeIntro(count: number): string {
  return count === 1
    ? m.install_overwrite_existing_mod_merge_intro_one()
    : m.install_overwrite_existing_mod_merge_intro_other();
}

export function installOverwriteSelectRequiredHint(): string {
  return m.install_overwrite_select_required_hint();
}

export function dependencyNotInstalled(): string {
  return m.dependency_not_installed();
}

export function dependencyVersionTooLow(): string {
  return m.dependency_version_too_low();
}

export function dependencyInstalled(): string {
  return m.dependency_installed();
}

export function dependencyDisabled(): string {
  return m.dependency_disabled();
}

export function dependencyLoadOrderLabel(): string {
  return m.dependency_load_order_label();
}

export function dependencyOptionalAbsent(): string {
  return m.dependency_optional_absent();
}

export function dependencySearchNexus(): string {
  return m.dependency_search_nexus();
}

export function dependencyOpenNexus(): string {
  return m.dependency_open_nexus();
}

export function dependencyEnableMod(): string {
  return m.dependency_enable_mod();
}

export function dependentViewMod(): string {
  return m.dependent_view_mod();
}

export function dependentsSummary(count: number): string {
  return count === 1
    ? m.dependents_summary_one()
    : m.dependents_summary_other({ count });
}

export function installDependencyWarningTitle(): string {
  return m.install_dependency_warning_title();
}

export function installDependencyWarningBody(count: number): string {
  return count === 1
    ? m.install_dependency_warning_body_one()
    : m.install_dependency_warning_body_other();
}

export function installAnywayLabel(): string {
  return m.install_anyway_label();
}

export function installOverwriteWarningTitle(): string {
  return m.install_overwrite_warning_title();
}

export function installOverwriteWarningBody(fileCount: number): string {
  return fileCount === 1
    ? m.install_overwrite_warning_body_one()
    : m.install_overwrite_warning_body_other();
}

export function installOverwriteConfirmLabel(): string {
  return m.install_overwrite_confirm_label();
}

export function installOverwriteTargetLegend(): string {
  return m.install_overwrite_target_legend();
}

export function installOverwriteTargetHint(): string {
  return m.install_overwrite_target_hint();
}

export function installOverwriteMultiTargetHint(): string {
  return m.install_overwrite_multi_target_hint();
}

export function installOverwriteMatchSummary(
  matched: number,
  total: number,
): string {
  return m.install_overwrite_match_summary({ matched, total });
}

export function installOverwriteSamplePathsLabel(): string {
  return m.install_overwrite_sample_paths_label();
}

export function modContainsOverwritesLabel(): string {
  return m.mod_contains_overwrites_label();
}

export function modContainsOverwritesTooltip(): string {
  return m.mod_contains_overwrites_tooltip();
}

export function installSuggestedTagsHint(count: number): string {
  return count === 1
    ? m.install_suggested_tags_hint_one()
    : m.install_suggested_tags_hint_other({ count });
}

export function dependencyIssuesFooterMessage(count: number): string {
  return count === 1
    ? m.dependency_issues_footer_one()
    : m.dependency_issues_footer_other({ count });
}

export function updatesFilterFooterMessage(count: number): string {
  return count === 1
    ? m.updates_filter_footer_one()
    : m.updates_filter_footer_other({ count });
}

export function gridUpdatesFilterLabel(count: number): string {
  return count === 1
    ? m.grid_updates_filter_one()
    : m.grid_updates_filter_other({ count });
}

export function gridDependencyFilterLabel(count: number): string {
  return count === 1
    ? m.grid_dependency_filter_one()
    : m.grid_dependency_filter_other({ count });
}

export function gridIncompatibleFilterLabel(count: number): string {
  return count === 1
    ? m.grid_incompatible_filter_one()
    : m.grid_incompatible_filter_other({ count });
}

export function gridIncompatibleFilterEmptyTitle(): string {
  return m.grid_incompatible_filter_empty_title();
}

export function gridIncompatibleFilterEmptyHint(): string {
  return m.grid_incompatible_filter_empty_hint();
}

export function incompatibleFilterFooterMessage(count: number): string {
  return count === 1
    ? m.incompatible_filter_footer_one()
    : m.incompatible_filter_footer_other({ count });
}

export function incompatibleIssueCountLabel(count: number): string {
  return count === 1
    ? m.incompatible_issue_count_one()
    : m.incompatible_issue_count_other({ count });
}

export function gridUpdatesFilterEmptyTitle(): string {
  return m.grid_updates_filter_empty_title();
}

export function gridUpdatesFilterEmptyHint(): string {
  return m.grid_updates_filter_empty_hint();
}

export function gridDependencyFilterEmptyTitle(): string {
  return m.grid_dependency_filter_empty_title();
}

export function gridDependencyFilterEmptyHint(): string {
  return m.grid_dependency_filter_empty_hint();
}

export function gridTagsFilteringBadge(count: number): string {
  return count === 1
    ? m.grid_tags_filtering_badge_one()
    : m.grid_tags_filtering_badge_other({ count });
}

export function gridBulkSelectedLabel(count: number): string {
  return m.selection_count({ count });
}

export function gridBulkDeleteOpeningAnnounce(count: number): string {
  return count === 1
    ? m.grid_bulk_delete_opening_one()
    : m.grid_bulk_delete_opening_other({ count });
}

export function dependenciesMissingSummary(count: number): string {
  return count === 1
    ? m.dependencies_missing_summary_one()
    : m.dependencies_missing_summary_other({ count });
}

export function installNamingDisclosure(count: number): string {
  return count === 1
    ? m.install_naming_disclosure_one()
    : m.install_naming_disclosure_other({ count });
}

export function workspaceOnboardingText(): string {
  return m.workspace_onboarding_text();
}

export function tagsSidebarShowTitle(
  activeFilters: number,
  narrowed: boolean,
): string {
  if (narrowed) {
    return activeFilters === 1
      ? m.tags_sidebar_show_title_one_filter()
      : m.tags_sidebar_show_title_many_filters({ count: activeFilters });
  }
  return m.tags_sidebar_show_title();
}

export function tagsSidebarShowAria(
  activeFilters: number,
  narrowed: boolean,
): string {
  if (narrowed) {
    return activeFilters === 1
      ? m.tags_sidebar_show_aria_one_filter()
      : m.tags_sidebar_show_aria_many_filters({ count: activeFilters });
  }
  return m.tags_sidebar_show_aria();
}

export function tagsSidebarFilterMeta(count: number): string {
  return count === 1
    ? m.tags_sidebar_filter_meta_one()
    : m.tags_sidebar_filter_meta_other({ count });
}

export function tagsFilterToggleTitle(name: string, active: boolean): string {
  return active
    ? m.tags_filter_toggle_title_active({ name })
    : m.tags_filter_toggle_title_inactive({ name });
}

export function tagsFilterToggleAria(name: string, active: boolean): string {
  return active
    ? m.tags_filter_toggle_aria_active({ name })
    : m.tags_filter_toggle_aria_inactive({ name });
}

export function tagsRenameAria(name: string): string {
  return m.tags_rename_aria({ name });
}

export function tagsDeleteAria(name: string): string {
  return m.tags_delete_aria({ name });
}

export function tagColorAria(label: string): string {
  return m.tag_color_aria({ label });
}

export const TAG_COLOR_PRESETS = [
  { hex: "#4a7c59", label: m.tag_color_forest() },
  { hex: "#6b8e4e", label: m.tag_color_grass() },
  { hex: "#c9a227", label: m.tag_color_gold() },
  { hex: "#8b6914", label: m.tag_color_wheat() },
  { hex: "#5b8a8a", label: m.tag_color_water() },
  { hex: "#a0522d", label: m.tag_color_wood() },
  { hex: "#c45c5c", label: m.tag_color_berry() },
  { hex: "#64748b", label: m.tag_color_stone() },
] as const;

export function tagsCellEditLabel(modName: string, tagNames: string[]): string {
  if (tagNames.length === 0) {
    return m.tags_cell_edit_empty({ modName });
  }
  return m.tags_cell_edit_with_tags({
    modName,
    tagNames: tagNames.join(", "),
  });
}

export function tagsOverflowLabel(count: number, names: string): string {
  return count === 1
    ? m.tags_overflow_one({ names })
    : m.tags_overflow_other({ count, names });
}

export function dropdownTypeaheadJumpHint(query: string): string {
  return m.dropdown_typeahead_jump_hint({ query });
}

export const settingsHideDisabledOptions = [
  { value: "none", label: m.settings_hide_disabled_none() },
  { value: "enabled", label: m.settings_hide_disabled_enabled() },
  { value: "disabled", label: m.settings_hide_disabled_disabled() },
] as const;

export function themeDropdownOptions(): { value: string; label: string }[] {
  return [
    { value: "stardew-dark", label: m.theme_stardew_dark_label() },
    { value: "stardew-light", label: m.theme_stardew_light_label() },
    { value: "cerberus", label: m.theme_cerberus_label() },
    { value: "mona", label: m.theme_mona_label() },
    { value: "vox", label: m.theme_vox_label() },
  ];
}

export function configEditorWindowTitle(
  modName: string,
  fileName?: string,
): string {
  return m.config_editor_window_title({
    modName: modName.trim() || "Mod",
    fileName: fileName?.trim() || "config.json",
  });
}

export function jsoncParseErrorMessage(code: number): string {
  switch (code) {
    case ParseErrorCode.InvalidSymbol:
      return m.jsonc_error_invalid_symbol();
    case ParseErrorCode.InvalidNumberFormat:
      return m.jsonc_error_invalid_number_format();
    case ParseErrorCode.PropertyNameExpected:
      return m.jsonc_error_property_name_expected();
    case ParseErrorCode.ValueExpected:
      return m.jsonc_error_value_expected();
    case ParseErrorCode.ColonExpected:
      return m.jsonc_error_colon_expected();
    case ParseErrorCode.CommaExpected:
      return m.jsonc_error_comma_expected();
    case ParseErrorCode.CloseBraceExpected:
      return m.jsonc_error_close_brace_expected();
    case ParseErrorCode.CloseBracketExpected:
      return m.jsonc_error_close_bracket_expected();
    case ParseErrorCode.EndOfFileExpected:
      return m.jsonc_error_end_of_file_expected();
    case ParseErrorCode.InvalidCommentToken:
      return m.jsonc_error_invalid_comment_token();
    case ParseErrorCode.UnexpectedEndOfComment:
      return m.jsonc_error_unexpected_end_of_comment();
    case ParseErrorCode.UnexpectedEndOfString:
      return m.jsonc_error_unexpected_end_of_string();
    case ParseErrorCode.UnexpectedEndOfNumber:
      return m.jsonc_error_unexpected_end_of_number();
    case ParseErrorCode.InvalidUnicode:
      return m.jsonc_error_invalid_unicode();
    case ParseErrorCode.InvalidEscapeCharacter:
      return m.jsonc_error_invalid_escape_character();
    case ParseErrorCode.InvalidCharacter:
      return m.jsonc_error_invalid_character();
    default:
      return m.jsonc_error_default();
  }
}

export function configEditorLoadFailedFor(path: string): string {
  const file = path.trim();
  return file
    ? m.config_editor_load_failed_for({ path: file })
    : m.config_editor_load_failed();
}

export function configEditorParseError(
  line: number,
  column: number,
  message: string,
): string {
  if (line > 0 && column > 0) {
    return m.config_editor_parse_error({ line, column, message });
  }
  return message;
}

export function configEditorProfileBanner(profileName: string): string {
  return m.config_editor_profile_banner({
    profileName: profileName.trim() || "Active profile",
  });
}

export function configEditorSaveFailed(reason: string): string {
  return reason.trim()
    ? m.config_editor_save_failed({ reason })
    : m.config_editor_save_failed_generic();
}

export { normalizeArchivePaths } from "$lib/paths";
