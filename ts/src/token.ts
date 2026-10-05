import type { Pattern } from './val.ts';

/**
 * The token types produced by the scanner. The names match the Go and Java
 * implementations, and the token types in the shared test corpus.
 */
export const TokenType = {
  LEFT_PAREN: 'LEFT_PAREN',
  RIGHT_PAREN: 'RIGHT_PAREN',

  NOT_EQUAL: 'NOT_EQUAL',
  EQUAL: 'EQUAL',
  GREATER: 'GREATER',
  GREATER_EQUAL: 'GREATER_EQUAL',
  LESS: 'LESS',
  LESS_EQUAL: 'LESS_EQUAL',

  IDENTIFIER: 'IDENTIFIER',
  BUILTIN: 'BUILTIN',
  STRING: 'STRING',
  NUMBER: 'NUMBER',
  PATTERN: 'PATTERN',

  AND: 'AND',
  OR: 'OR',
  NOT: 'NOT',
  TRUE: 'TRUE',
  FALSE: 'FALSE',

  EOF: 'EOF',
} as const;

export type TokenType = (typeof TokenType)[keyof typeof TokenType];

/** The scanned value of a literal token. */
export type Literal = string | bigint | Pattern;

export interface Token {
  readonly type: TokenType;
  /** The source text of the token. */
  readonly lexeme: string;
  /**
   * The value of NUMBER, STRING and PATTERN tokens. IDENTIFIER, BUILTIN and
   * keyword tokens carry their lexeme.
   */
  readonly literal?: Literal;
  /** The UTF-16 offset of the start of the token in the source. */
  readonly charPos: number;
}
