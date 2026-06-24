const ARCHIVE_PATTERN = /\.(zip|7z|rar)$/i;

export function isArchivePath(path: string): boolean {
  return ARCHIVE_PATTERN.test(path.trim());
}

export function pathBasename(path: string): string {
  const normalized = path.replace(/\\/g, "/");
  const base = normalized.split("/").pop();
  return base && base.length > 0 ? base : path;
}

export function normalizeArchivePaths(paths: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const raw of paths) {
    const path = raw.trim();
    if (!path || !isArchivePath(path) || seen.has(path)) continue;
    seen.add(path);
    out.push(path);
  }
  return out;
}
