import * as API from "./index";
import { dedupeMods } from "$lib/mods/dedupe";
import { USE_MOCK_DATA, getMockRefreshData, getMockInstallDependencyPreview, setMockModEnabled, } from "$lib/mock/designData";
export { USE_MOCK_DATA };
export async function refreshAll(state) {
    if (USE_MOCK_DATA) {
        return getMockRefreshData(state.search, state.hideDisabled);
    }
    const [mods, profiles, categories, settings, smapiVersion, readyCount, dependencyIssueCount,] = await Promise.all([
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
export async function previewInstallDependencies(paths) {
    if (USE_MOCK_DATA) {
        return getMockInstallDependencyPreview(paths);
    }
    return API.PreviewInstallDependencies(paths) ?? [];
}
export async function setModEnabled(modId, enabled) {
    if (USE_MOCK_DATA) {
        setMockModEnabled(modId, enabled);
        return;
    }
    return API.SetModEnabled(modId, enabled);
}
//# sourceMappingURL=client.js.map