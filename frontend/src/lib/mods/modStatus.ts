import type { Mod } from "$lib/api/client";
import * as m from "$lib/paraglide/messages.js";
import {
  dependencyIssuesTooltip,
  missingDependencyBadge,
} from "$lib/i18n/helpers";

export type ModStatusInfo = {
  text: string;
  badge: string;
  title?: string;
};

export function modStatusInfo(mod: Mod, lastUpdateCheck = 0): ModStatusInfo {
  const state = mod.updateStatus?.state;
  if (state === "update" || state === "update_available") {
    const latest = mod.updateStatus?.latestVersion;
    const text = latest
      ? m.status_update_to_version({ version: latest })
      : m.status_update_on_nexus();
    return { text, badge: "state-badge state-badge--update" };
  }
  if (state === "update_ignored") {
    const latest = mod.updateStatus?.latestVersion;
    const text = latest
      ? m.status_update_ignored_version({ version: latest })
      : m.status_update_ignored();
    return { text, badge: "state-badge state-badge--muted" };
  }
  if (state === "incompatible") {
    const compat = mod.updateStatus?.compatibilityStatus?.trim();
    const summary = mod.updateStatus?.compatibilitySummary?.trim();
    const msg = mod.updateStatus?.message?.trim();
    const text = compat || msg || m.status_incompatible();
    const title = summary || (msg && msg.length > 40 ? msg : undefined);
    return {
      text,
      badge: "state-badge state-badge--error",
      title,
    };
  }

  const issueCount =
    mod.missingDependencyCount ?? mod.dependencyIssues?.length ?? 0;
  if (issueCount > 0) {
    const issues = mod.dependencyIssues ?? [];
    const text = missingDependencyBadge(issueCount, issues);
    const title = dependencyIssuesTooltip(issues);
    return { text, badge: "state-badge state-badge--error", title };
  }

  if (state === "unofficial") {
    const latest = mod.updateStatus?.latestVersion?.trim();
    const text = latest
      ? m.status_unofficial_version({ version: latest })
      : m.status_unofficial();
    return { text, badge: "state-badge state-badge--muted" };
  }
  if (mod.isCoreMod) {
    return { text: m.status_core(), badge: "state-badge state-badge--info" };
  }
  if (mod.enabled) {
    if (lastUpdateCheck === 0) {
      return {
        text: m.mod_not_checked_label(),
        badge: "state-badge state-badge--muted",
      };
    }
    return {
      text: m.status_up_to_date(),
      badge: "state-badge state-badge--success",
    };
  }
  return { text: m.status_disabled(), badge: "state-badge state-badge--muted" };
}

export function modStatusSortKey(mod: Mod): number {
  const state = mod.updateStatus?.state;
  if (state === "update" || state === "update_available") return 0;
  if (state === "incompatible") return 1;
  if ((mod.missingDependencyCount ?? mod.dependencyIssues?.length ?? 0) > 0)
    return 2;
  if (state === "unofficial") return 3;
  if (mod.isCoreMod) return 4;
  if (mod.enabled) return 5;
  return 6;
}
