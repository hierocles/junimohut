<script lang="ts">
  import { onMount } from "svelte";
  import { EditorState, type Extension } from "@codemirror/state";
  import {
    EditorView,
    keymap,
    lineNumbers,
    highlightActiveLine,
    highlightActiveLineGutter,
  } from "@codemirror/view";
  import { json } from "@codemirror/lang-json";
  import { linter, lintGutter, type Diagnostic } from "@codemirror/lint";
  import { defaultKeymap, indentWithTab } from "@codemirror/commands";
  import {
    syntaxHighlighting,
    defaultHighlightStyle,
  } from "@codemirror/language";
  import { jsoncLintDiagnostics } from "$lib/mods/jsonc";
  import { jsoncCommentHighlight } from "$lib/mods/jsoncComments";

  interface Props {
    value: string;
    revision?: number;
    onchange?: (value: string) => void;
  }

  let { value, revision = 0, onchange }: Props = $props();

  let host = $state<HTMLDivElement | undefined>();
  let view = $state<EditorView | undefined>();
  let appliedRevision = $state(-1);

  const editorTheme = EditorView.theme({
    "&": {
      height: "100%",
      fontSize: "0.8125rem",
      fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
    },
    ".cm-scroller": {
      overflow: "auto",
      fontFamily: "inherit",
    },
    "&.cm-focused": {
      outline: "none",
    },
    ".cm-content": {
      caretColor: "var(--color-surface-50)",
      padding: "var(--space-3) 0",
    },
    ".cm-gutters": {
      backgroundColor: "var(--sdvm-panel)",
      color: "var(--color-surface-500)",
      border: "none",
      borderRight: "1px solid var(--sdvm-border)",
    },
    ".cm-activeLineGutter": {
      backgroundColor:
        "color-mix(in oklch, var(--color-primary-500) 8%, var(--sdvm-panel))",
    },
    ".cm-activeLine": {
      backgroundColor:
        "color-mix(in oklch, var(--color-primary-500) 6%, transparent)",
    },
    ".cm-cursor, .cm-dropCursor": {
      borderLeftColor: "var(--color-surface-50)",
    },
    "&.cm-focused .cm-selectionBackground, .cm-selectionBackground": {
      backgroundColor:
        "color-mix(in oklch, var(--color-primary-500) 28%, transparent) !important",
    },
    ".cm-lintRange-error": {
      backgroundImage: "none",
      textDecoration: "underline wavy var(--color-error-500)",
    },
    ".cm-jsonc-comment": {
      color: "var(--color-surface-500)",
      fontStyle: "italic",
    },
    ".cm-jsonc-comment.cm-invalid": {
      color: "var(--color-surface-500) !important",
      textDecoration: "none",
    },
  });

  function jsoncParseLinter() {
    return linter((view) => {
      const text = view.state.doc.toString();
      return jsoncLintDiagnostics(text).map(
        (diag): Diagnostic => ({
          from: diag.from,
          to: diag.to,
          severity: "error",
          message: diag.message,
        }),
      );
    });
  }

  function buildExtensions(): Extension[] {
    return [
      lineNumbers(),
      highlightActiveLine(),
      highlightActiveLineGutter(),
      lintGutter(),
      json(),
      jsoncCommentHighlight(),
      jsoncParseLinter(),
      syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
      editorTheme,
      EditorView.lineWrapping,
      keymap.of([...defaultKeymap, indentWithTab]),
      EditorView.updateListener.of((update) => {
        if (update.docChanged) {
          onchange?.(update.state.doc.toString());
        }
      }),
    ];
  }

  onMount(() => {
    if (!host) return;
    appliedRevision = revision;
    const created = new EditorView({
      state: EditorState.create({
        doc: value,
        extensions: buildExtensions(),
      }),
      parent: host,
    });
    view = created;
    return () => {
      created.destroy();
      view = undefined;
    };
  });

  $effect(() => {
    const current = view;
    const nextRevision = revision;
    if (!current || nextRevision === appliedRevision) return;
    appliedRevision = nextRevision;
    current.dispatch({
      changes: { from: 0, to: current.state.doc.length, insert: value },
    });
  });

  export function focusEditor() {
    view?.focus();
  }
</script>

<div
  class="json-editor-host"
  bind:this={host}
  role="textbox"
  aria-multiline="true"
></div>

<style>
  .json-editor-host {
    height: 100%;
    min-height: 0;
    background: var(--sdvm-shell);
  }

  .json-editor-host :global(.cm-editor) {
    height: 100%;
  }
</style>
