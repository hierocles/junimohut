import { normalizeArchivePaths } from "$lib/copy";
import { isWailsHost } from "$lib/wails/windowApi";

/** Drop zone ids — must match `data-file-drop-target` element ids in the DOM. */
export const INSTALL_MODAL_DROP_ID = "install-modal-drop";
export const MOD_GRID_DROP_ID = "mod-grid-drop";

/** Wails native OS file drop; HTML5 drag/drop is for dev/mock only. */
export const useNativeArchiveFileDrop = isWailsHost();

export type FilesDroppedPayload = {
  files: string[];
  targetId: string;
};

export function pathsFromFileList(files: FileList | File[]): string[] {
  return [...files]
    .map((f) => (f as File & { path?: string }).path ?? f.name)
    .filter((p) => p.length > 0);
}

export function pathsFromDataTransfer(
  dataTransfer: DataTransfer | null,
): string[] {
  if (!dataTransfer?.files?.length) return [];
  return normalizeArchivePaths(pathsFromFileList(dataTransfer.files));
}

export function isOsFileDrag(dataTransfer: DataTransfer | null): boolean {
  return !!dataTransfer?.types.includes("Files");
}
