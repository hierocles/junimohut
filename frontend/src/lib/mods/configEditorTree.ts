import type { ModJsonFileNode } from "../../../bindings/junimohut/internal/mods/models";

export type { ModJsonFileNode };

export type FlatTreeRow =
  | {
      type: "dir";
      dirKey: string;
      name: string;
      depth: number;
      expanded: boolean;
      focusKey: string;
    }
  | {
      type: "file";
      relPath: string;
      name: string;
      depth: number;
      focusKey: string;
    };

export function flattenFileTree(
  nodes: ModJsonFileNode[],
  expandedDirs: ReadonlySet<string>,
  dirKeyPrefix = "",
  depth = 0,
): FlatTreeRow[] {
  const out: FlatTreeRow[] = [];
  for (const node of nodes) {
    if (node.isDir) {
      const dirKey = dirKeyPrefix + node.name + "/";
      const expanded = expandedDirs.has(dirKey);
      out.push({
        type: "dir",
        dirKey,
        name: node.name,
        depth,
        expanded,
        focusKey: `dir:${dirKey}`,
      });
      if (expanded && node.children?.length) {
        out.push(
          ...flattenFileTree(node.children, expandedDirs, dirKey, depth + 1),
        );
      }
    } else if (node.relPath) {
      out.push({
        type: "file",
        relPath: node.relPath,
        name: node.name,
        depth,
        focusKey: `file:${node.relPath}`,
      });
    }
  }
  return out;
}

export function treeFocusKeyForPath(relPath: string): string {
  return `file:${relPath}`;
}
