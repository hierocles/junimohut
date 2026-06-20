import { Window as WailsWindow } from '@wailsio/runtime';

type WailsGlobal = Window & {
  _wails?: { invoke?: unknown };
};

/** True when running inside a Wails webview (not Vite-only mock dev). */
export function isWailsHost(): boolean {
  return typeof window !== 'undefined' && typeof (window as WailsGlobal)._wails?.invoke === 'function';
}

export async function minimiseWindow(): Promise<void> {
  if (!isWailsHost()) return;
  await WailsWindow.Minimise();
}

export async function toggleWindowMaximise(): Promise<void> {
  if (!isWailsHost()) return;
  await WailsWindow.ToggleMaximise();
}

export async function closeWindow(): Promise<void> {
  if (!isWailsHost()) return;
  await WailsWindow.Close();
}

export async function queryWindowMaximised(): Promise<boolean> {
  if (!isWailsHost()) return false;
  return WailsWindow.IsMaximised();
}

export async function queryWindowFocused(): Promise<boolean> {
  if (!isWailsHost()) return true;
  return WailsWindow.IsFocused();
}

export function onDragRegionDoubleClick(event: MouseEvent): void {
  if (event.button !== 0) return;
  void toggleWindowMaximise();
}
