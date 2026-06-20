export type Mod = Awaited<
  ReturnType<typeof import("./index").ListMods>
>[number];
export type Profile = Awaited<
  ReturnType<typeof import("./index").ListProfiles>
>[number];
export type Category = Awaited<
  ReturnType<typeof import("./index").ListCategories>
>[number];
export type Settings = Awaited<
  ReturnType<typeof import("./index").GetSettings>
>;
export type InstallResult = NonNullable<
  Awaited<ReturnType<typeof import("./index").InstallMods>>
>[number];

export type InstallOptions = {
  mode: "install" | "replace";
  deleteOld?: boolean;
  useFolderDisplayNames?: boolean;
};

import * as API from "./index";
import { dedupeMods } from "$lib/mods/dedupe";
import {
  USE_MOCK_DATA,
  getMockRefreshData,
  getMockInstallDependencyPreview,
  setMockModEnabled,
} from "$lib/mock/designData";

export { USE_MOCK_DATA };

export async function refreshAll(state: {
  search: string;
  hideDisabled: string;
}) {
  if (USE_MOCK_DATA) {
    return getMockRefreshData(state.search, state.hideDisabled);
  }

  const [
    mods,
    profiles,
    categories,
    settings,
    smapiVersion,
    readyCount,
    dependencyIssueCount,
  ] = await Promise.all([
    API.ListMods(state.search, state.hideDisabled),
    API.ListProfiles(),
    API.ListCategories(),
    API.GetSettings(),
    API.GetSMAPIVersion(),
    API.ModsReadyToUpdate(),
    API.ModsWithDependencyIssues(),
  ]);
  return {
    mods: dedupeMods(mods ?? []),
    profiles,
    categories,
    settings,
    smapiVersion,
    readyCount,
    dependencyIssueCount,
  };
}

export async function previewInstallDependencies(paths: string[]) {
  if (USE_MOCK_DATA) {
    return getMockInstallDependencyPreview(paths);
  }
  return API.PreviewInstallDependencies(paths) ?? [];
}

export async function setModEnabled(modId: string, enabled: boolean) {
  if (USE_MOCK_DATA) {
    setMockModEnabled(modId, enabled);
    return;
  }
  return API.SetModEnabled(modId, enabled);
}
