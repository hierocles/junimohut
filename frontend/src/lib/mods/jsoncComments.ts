import { RangeSetBuilder } from "@codemirror/state";
import {
  Decoration,
  EditorView,
  ViewPlugin,
  type ViewUpdate,
} from "@codemirror/view";
import { createScanner } from "jsonc-parser";
import { SyntaxKind } from "$lib/mods/jsoncEnums";

const commentMark = Decoration.mark({ class: "cm-jsonc-comment" });

function buildCommentDecorations(text: string) {
  const builder = new RangeSetBuilder<Decoration>();
  const scanner = createScanner(text, false);
  let kind = scanner.scan();
  while (kind !== SyntaxKind.EOF) {
    if (
      kind === SyntaxKind.LineCommentTrivia ||
      kind === SyntaxKind.BlockCommentTrivia
    ) {
      builder.add(
        scanner.getTokenOffset(),
        scanner.getTokenOffset() + scanner.getTokenLength(),
        commentMark,
      );
    }
    kind = scanner.scan();
  }
  return builder.finish();
}

export function jsoncCommentHighlight() {
  return ViewPlugin.fromClass(
    class {
      decorations = Decoration.none;

      constructor(view: EditorView) {
        this.decorations = buildCommentDecorations(view.state.doc.toString());
      }

      update(update: ViewUpdate) {
        if (update.docChanged) {
          this.decorations = buildCommentDecorations(
            update.view.state.doc.toString(),
          );
        }
      }
    },
    { decorations: (plugin) => plugin.decorations },
  );
}
