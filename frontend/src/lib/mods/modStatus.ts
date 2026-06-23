import type { Mod } from "$lib/api/client";
import {
  dependencyIssuesTooltip,
  missingDependencyBadge,
  modNotCheckedLabel,
} from "$lib/copy";

export type ModStatusInfo = {
  text: string;
  badge: string;
  title?: string;
};

export function modStatusInfo(mod: Mod, lastUpdateCheck = 0): ModStatusInfo {
  const state = mod.updateStatus?.state;
  if (state === "update" || state === "update_available") {
    const latest = mod.updateStatus?.latestVersion;
    const text = latest ? `Update to v${latest}` : "Update on Nexus";
    return { text, badge: "state-badge state-badge--update" };
  }
  if (state === "update_ignored") {
    const latest = mod.updateStatus?.latestVersion;
    const text = latest ? `v${latest} ignored` : "Update ignored";
    return { text, badge: "state-badge state-badge--muted" };
  }
  if (state === "incompatible") {
    const msg = mod.updateStatus?.message?.trim();
    const text = msg || "Incompatible";
    return {
      text,
      badge: "state-badge state-badge--error",
      title: msg && msg.length > 40 ? msg : undefined,
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
    return { text: "Unofficial", badge: "state-badge state-badge--muted" };
  }
  if (mod.isCoreMod) {
    return { text: "Core", badge: "state-badge state-badge--info" };
  }
  if (mod.enabled) {
    if (lastUpdateCheck === 0) {
      return {
        text: modNotCheckedLabel,
        badge: "state-badge state-badge--muted",
      };
    }
    return { text: "Up to date", badge: "state-badge state-badge--success" };
  }
  return { text: "Disabled", badge: "state-badge state-badge--muted" };
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
