// jsonc-parser uses `const enum` which cannot be accessed under isolatedModules.
// These objects mirror the library's stable numeric values so callers can import
// concrete values without disabling any TypeScript flags.

export const ParseErrorCode = {
  InvalidSymbol: 1,
  InvalidNumberFormat: 2,
  PropertyNameExpected: 3,
  ValueExpected: 4,
  ColonExpected: 5,
  CommaExpected: 6,
  CloseBraceExpected: 7,
  CloseBracketExpected: 8,
  EndOfFileExpected: 9,
  InvalidCommentToken: 10,
  UnexpectedEndOfComment: 11,
  UnexpectedEndOfString: 12,
  UnexpectedEndOfNumber: 13,
  InvalidUnicode: 14,
  InvalidEscapeCharacter: 15,
  InvalidCharacter: 16,
} as const;

export const SyntaxKind = {
  Unknown: 0,
  OpenBraceToken: 1,
  CloseBraceToken: 2,
  OpenBracketToken: 3,
  CloseBracketToken: 4,
  CommaToken: 5,
  ColonToken: 6,
  NullKeyword: 7,
  TrueKeyword: 8,
  FalseKeyword: 9,
  StringLiteral: 10,
  NumericLiteral: 11,
  LineCommentTrivia: 12,
  BlockCommentTrivia: 13,
  LineBreakTrivia: 14,
  Trivia: 15,
  EOF: 17,
} as const;
