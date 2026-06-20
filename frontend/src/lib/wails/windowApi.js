import { Window as WailsWindow } from '@wailsio/runtime';
/** True when running inside a Wails webview (not Vite-only mock dev). */
export function isWailsHost() {
    return typeof window !== 'undefined' && typeof window._wails?.invoke === 'function';
}
export async function minimiseWindow() {
    if (!isWailsHost())
        return;
    await WailsWindow.Minimise();
}
export async function toggleWindowMaximise() {
    if (!isWailsHost())
        return;
    await WailsWindow.ToggleMaximise();
}
export async function closeWindow() {
    if (!isWailsHost())
        return;
    await WailsWindow.Close();
}
export async function queryWindowMaximised() {
    if (!isWailsHost())
        return false;
    return WailsWindow.IsMaximised();
}
export async function queryWindowFocused() {
    if (!isWailsHost())
        return true;
    return WailsWindow.IsFocused();
}
export function onDragRegionDoubleClick(event) {
    if (event.button !== 0)
        return;
    void toggleWindowMaximise();
}
//# sourceMappingURL=windowApi.js.map