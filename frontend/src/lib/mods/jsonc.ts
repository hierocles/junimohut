import { parse, type ParseError } from "jsonc-parser";
import { jsoncParseErrorMessage } from "$lib/i18n/helpers";

export type JsoncParseState = {
  valid: boolean;
  line: number;
  column: number;
  message: string;
};

function offsetToLineColumn(
  text: string,
  offset: number,
): {
  line: number;
  column: number;
} {
  const before = text.slice(0, offset);
  const line = before.split("\n").length;
  const lastBreak = before.lastIndexOf("\n");
  return { line, column: offset - lastBreak };
}

export function parseJsoncState(text: string): JsoncParseState {
  const errors: ParseError[] = [];
  parse(text, errors, { allowTrailingComma: true });
  if (errors.length === 0) {
    return { valid: true, line: 0, column: 0, message: "" };
  }
  const err = errors[0];
  const { line, column } = offsetToLineColumn(text, err.offset);
  return {
    valid: false,
    line,
    column,
    message: jsoncParseErrorMessage(err.error),
  };
}

export function jsoncLintDiagnostics(text: string) {
  const errors: ParseError[] = [];
  parse(text, errors, { allowTrailingComma: true });
  return errors.map((err) => ({
    from: err.offset,
    to: Math.min(err.offset + 1, text.length),
    message: jsoncParseErrorMessage(err.error),
  }));
}

export { ParseErrorCode } from "$lib/mods/jsoncEnums";
