import { HighlightStyle } from "@codemirror/language";
import { tags } from "@lezer/highlight";

/** Dark-surface JSON / JSONC syntax colors — values are CSS vars in app.css. */
export const sdvmJsonHighlightStyle = HighlightStyle.define([
  { tag: tags.propertyName, color: "var(--sdvm-editor-key)" },
  { tag: tags.string, color: "var(--sdvm-editor-string)" },
  { tag: tags.number, color: "var(--sdvm-editor-number)" },
  { tag: tags.bool, color: "var(--sdvm-editor-bool)" },
  {
    tag: tags.null,
    color: "var(--sdvm-editor-null)",
    fontStyle: "italic",
  },
  { tag: tags.separator, color: "var(--sdvm-editor-punct)" },
  { tag: tags.brace, color: "var(--sdvm-editor-bracket)" },
  { tag: tags.squareBracket, color: "var(--sdvm-editor-bracket)" },
]);
