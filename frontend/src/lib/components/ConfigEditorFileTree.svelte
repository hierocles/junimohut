<script lang="ts">
  import { ChevronRight, FileJson, Folder } from "@lucide/svelte";
  import ConfigEditorFileTree from "$lib/components/ConfigEditorFileTree.svelte";

  export type ModJsonFileNode = {
    name: string;
    relPath?: string;
    isDir: boolean;
    children?: ModJsonFileNode[];
  };

  interface Props {
    nodes: ModJsonFileNode[];
    activeRelPath: string;
    expandedDirs: ReadonlySet<string>;
    focusedKey?: string | null;
    onselectfile: (relPath: string) => void;
    ontoggledir: (dirKey: string) => void;
    dirKeyPrefix?: string;
    depth?: number;
    setSize?: number;
    posInSet?: number;
  }

  let {
    nodes,
    activeRelPath,
    expandedDirs,
    focusedKey = null,
    onselectfile,
    ontoggledir,
    dirKeyPrefix = "",
    depth = 0,
    setSize = 1,
    posInSet = 1,
  }: Props = $props();
</script>

<ul class="config-file-tree" role="tree" aria-label="JSON files">
  {#each nodes as node, index (dirKeyPrefix + node.name + (node.relPath ?? ""))}
    {@const pos = posInSet + index}
    <li role="none">
      {#if node.isDir}
        {@const dirKey = dirKeyPrefix + node.name + "/"}
        {@const expanded = expandedDirs.has(dirKey)}
        {@const focusKey = `dir:${dirKey}`}
        <button
          type="button"
          class="config-tree-row config-tree-row--dir"
          class:expanded
          class:keyboard-focused={focusedKey === focusKey}
          style:--tree-depth={depth}
          role="treeitem"
          aria-selected={false}
          aria-expanded={expanded}
          aria-level={depth + 1}
          aria-setsize={setSize}
          aria-posinset={pos}
          tabindex={focusedKey === focusKey ? 0 : -1}
          data-focus-key={focusKey}
          onclick={() => ontoggledir(dirKey)}
        >
          <span class="config-tree-chevron" class:rotated={expanded}>
            <ChevronRight size={14} aria-hidden="true" />
          </span>
          <Folder size={14} aria-hidden="true" />
          <span class="truncate">{node.name}</span>
        </button>
        {#if expanded && node.children?.length}
          <ConfigEditorFileTree
            nodes={node.children}
            {activeRelPath}
            {expandedDirs}
            {focusedKey}
            {onselectfile}
            {ontoggledir}
            dirKeyPrefix={dirKeyPrefix + node.name + "/"}
            depth={depth + 1}
            setSize={node.children.length}
            posInSet={1}
          />
        {/if}
      {:else if node.relPath}
        {@const focusKey = `file:${node.relPath}`}
        <button
          type="button"
          class="config-tree-row config-tree-row--file"
          class:active={activeRelPath === node.relPath}
          class:keyboard-focused={focusedKey === focusKey}
          style:--tree-depth={depth}
          role="treeitem"
          aria-selected={activeRelPath === node.relPath}
          aria-level={depth + 1}
          aria-setsize={setSize}
          aria-posinset={pos}
          tabindex={focusedKey === focusKey ? 0 : -1}
          data-focus-key={focusKey}
          onclick={() => onselectfile(node.relPath!)}
        >
          <FileJson size={14} aria-hidden="true" />
          <span class="truncate">{node.name}</span>
        </button>
      {/if}
    </li>
  {/each}
</ul>

<style>
  .config-file-tree {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .config-tree-row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    width: 100%;
    min-height: 2rem;
    padding: var(--space-1) var(--space-2);
    padding-left: calc(var(--space-2) + var(--tree-depth, 0) * var(--space-3));
    border: none;
    background: transparent;
    color: var(--color-surface-300);
    font-size: var(--type-ui);
    text-align: left;
    border-radius: var(--radius-base);
    transition: background-color var(--motion-fast) var(--ease-out-quart);
  }

  .config-tree-row:hover {
    background: var(--sdvm-raised);
    color: var(--color-surface-50);
  }

  .config-tree-row.active {
    background: color-mix(
      in oklch,
      var(--color-primary-500) 16%,
      var(--sdvm-raised)
    );
    color: var(--color-surface-50);
  }

  .config-tree-row.keyboard-focused,
  .config-tree-row:focus-visible {
    outline: none;
    box-shadow: inset 0 0 0 2px
      color-mix(in oklch, var(--color-primary-500) 55%, transparent);
  }

  .config-tree-chevron {
    display: inline-flex;
    flex-shrink: 0;
    transition: transform var(--motion-fast) var(--ease-out-quart);
  }

  .config-tree-chevron.rotated {
    transform: rotate(90deg);
  }

  @media (prefers-reduced-motion: reduce) {
    .config-tree-row,
    .config-tree-chevron {
      transition: none;
    }
  }
</style>
