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
  import { syntaxHighlighting } from "@codemirror/language";
  import { jsoncLintDiagnostics } from "$lib/mods/jsonc";
  import { jsoncCommentHighlight } from "$lib/mods/jsoncComments";
  import { sdvmJsonHighlightStyle } from "$lib/mods/jsonEditorHighlight";

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
      fontFamily: "var(--font-mono)",
      letterSpacing: "0.01em",
      lineHeight: "var(--leading-snug)",
      backgroundColor: "var(--sdvm-shell)",
      color: "var(--color-surface-200)",
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
      padding: "var(--space-3) var(--space-4)",
    },
    ".cm-line": {
      padding: "0 var(--space-1)",
    },
    ".cm-gutters": {
      backgroundColor: "var(--sdvm-panel)",
      color: "var(--color-surface-500)",
      border: "none",
      borderRight: "1px solid var(--sdvm-border)",
    },
    ".cm-gutterElement": {
      padding: "0 var(--space-2) 0 var(--space-3)",
      minWidth: "2.75rem",
    },
    ".cm-activeLineGutter": {
      color: "var(--color-surface-300)",
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
      backgroundColor: "var(--sdvm-selection-combined) !important",
    },
    ".cm-lintGutter": {
      width: "1.125rem",
    },
    ".cm-gutter-lint .cm-gutterElement": {
      padding: "0 var(--space-1)",
    },
    ".cm-lint-marker-error": {
      width: "0.4375rem",
      height: "0.4375rem",
      margin: "0.45rem auto 0",
      borderRadius: "var(--radius-pill)",
      backgroundColor: "var(--sdvm-error-fg)",
    },
    ".cm-lintRange-error": {
      backgroundImage: "none",
      textDecoration: "underline wavy var(--sdvm-error-fg)",
      textDecorationSkipInk: "none",
    },
    ".cm-jsonc-comment": {
      color: "var(--sdvm-editor-comment)",
      fontStyle: "italic",
    },
    ".cm-jsonc-comment.cm-invalid": {
      color: "var(--sdvm-editor-comment) !important",
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
      syntaxHighlighting(sdvmJsonHighlightStyle, { fallback: false }),
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
