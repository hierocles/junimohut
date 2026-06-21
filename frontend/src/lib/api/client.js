import * as API from "./index";
import { dedupeMods } from "$lib/mods/dedupe";

export const USE_MOCK_DATA = import.meta.env.VITE_USE_MOCK_DATA === "true";

export async function refreshCore(state) {
    if (USE_MOCK_DATA) {
        const { getMockRefreshData } = await import("$lib/mock/designData");
        const data = getMockRefreshData(state.search, state.hideDisabled);
        return {
            mods: data.mods,
            profiles: data.profiles,
            categories: data.categories,
            settings: data.settings,
            smapiVersion: data.smapiVersion,
        };
    }
    const [mods, profiles, categories, settings, smapiVersion] = await Promise.all([
        API.ListMods(state.search, state.hideDisabled),
        API.ListProfiles(),
        API.ListCategories(),
        API.GetSettings(),
        API.GetSMAPIVersion(),
    ]);
    return {
        mods: dedupeMods(mods ?? []),
        profiles,
        categories,
        settings,
        smapiVersion,
    };
}
export async function refreshFooterStats() {
    if (USE_MOCK_DATA) {
        const { getMockRefreshData } = await import("$lib/mock/designData");
        const data = getMockRefreshData("", "none");
        return {
            readyCount: data.readyCount,
            dependencyIssueCount: data.dependencyIssueCount,
            unmanagedMods: data.unmanagedMods,
        };
    }
    const [readyCount, dependencyIssueCount, unmanagedMods] = await Promise.all([
        API.ModsReadyToUpdate(),
        API.ModsWithDependencyIssues(),
        API.ListUnmanagedMods(),
    ]);
    return {
        readyCount: readyCount ?? 0,
        dependencyIssueCount: dependencyIssueCount ?? 0,
        unmanagedMods: unmanagedMods ?? [],
    };
}
export async function refreshAll(state) {
    const [core, stats] = await Promise.all([
        refreshCore(state),
        refreshFooterStats(),
    ]);
    return { ...core, ...stats };
}
export async function fetchLibraryMods() {
    if (USE_MOCK_DATA) {
        const { getMockRefreshData } = await import("$lib/mock/designData");
        return getMockRefreshData("", "none").mods;
    }
    return dedupeMods((await API.ListMods("", "none")) ?? []);
}
export async function previewInstallDependencies(paths) {
    if (USE_MOCK_DATA) {
        const { getMockInstallDependencyPreview } = await import("$lib/mock/designData");
        return getMockInstallDependencyPreview(paths);
    }
    return API.PreviewInstallDependencies(paths) ?? [];
}
export async function setModEnabled(modId, enabled) {
    if (USE_MOCK_DATA) {
        const { setMockModEnabled } = await import("$lib/mock/designData");
        setMockModEnabled(modId, enabled);
        return;
    }
    return API.SetModEnabled(modId, enabled);
}
