import { downloadUnknownModLabel, pathBasename } from "$lib/copy";
import { nexusModIdFromUpdateKey } from "$lib/mods/nexusTags";
function archiveFileLabel(record) {
    const name = record.fileName?.trim() || pathBasename(record.archivePath ?? "");
    return name.replace(/\.(zip|7z|rar)$/i, "").trim();
}
export function resolveArchiveMod(record, mods) {
    const uniqueId = record.uniqueId?.trim() ?? "";
    if (uniqueId) {
        const mod = mods.find((m) => (m.manifest?.UniqueID ?? "").toLowerCase() === uniqueId.toLowerCase());
        if (mod)
            return { displayName: mod.manifest.Name, mod };
    }
    const nexusModId = record.nexusModId ?? 0;
    if (nexusModId > 0) {
        const mod = mods.find((m) => m.manifest?.UpdateKeys?.some((key) => {
            const id = nexusModIdFromUpdateKey(key);
            return id === nexusModId;
        }));
        if (mod)
            return { displayName: mod.manifest.Name, mod };
    }
    const modName = record.modName?.trim() ?? "";
    if (modName)
        return { displayName: modName, mod: null };
    const fileLabel = archiveFileLabel(record);
    if (fileLabel)
        return { displayName: fileLabel, mod: null };
    return { displayName: downloadUnknownModLabel, mod: null };
}
export function archiveSearchText(record, displayName) {
    return [
        displayName,
        record.modName ?? "",
        record.fileName ?? "",
        record.uniqueId ?? "",
        record.archivePath ?? "",
    ]
        .join(" ")
        .toLowerCase();
}
export function formatDownloadTimestamp(ts) {
    if (!ts)
        return { label: "—", title: "" };
    const date = new Date(ts * 1000);
    const title = date.toLocaleString();
    const diffSec = Math.round((date.getTime() - Date.now()) / 1000);
    const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });
    const absSec = Math.abs(diffSec);
    if (absSec < 60)
        return { label: rtf.format(diffSec, "second"), title };
    const diffMin = Math.round(diffSec / 60);
    if (Math.abs(diffMin) < 60)
        return { label: rtf.format(diffMin, "minute"), title };
    const diffHour = Math.round(diffMin / 60);
    if (Math.abs(diffHour) < 48)
        return { label: rtf.format(diffHour, "hour"), title };
    const diffDay = Math.round(diffHour / 24);
    if (Math.abs(diffDay) < 14)
        return { label: rtf.format(diffDay, "day"), title };
    return {
        label: date.toLocaleDateString(undefined, {
            month: "short",
            day: "numeric",
            year: "numeric",
        }),
        title,
    };
}
//# sourceMappingURL=savedDownloads.js.map