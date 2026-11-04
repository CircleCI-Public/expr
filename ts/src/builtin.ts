export const Builtin = {
  CONTAINS: 'CONTAINS',
  MATCHES: 'MATCHES',
  STARTS_WITH: 'STARTS_WITH',
  IN: 'IN',
} as const;

export type Builtin = (typeof Builtin)[keyof typeof Builtin];

const lexemes: ReadonlyMap<string, Builtin> = new Map([
  ['contains', Builtin.CONTAINS],
  ['CONTAINS', Builtin.CONTAINS],
  ['matches', Builtin.MATCHES],
  ['MATCHES', Builtin.MATCHES],
  ['starts-with', Builtin.STARTS_WITH],
  ['STARTS-WITH', Builtin.STARTS_WITH],
]);

/** Returns the builtin named by lexeme, or undefined if there isn't one. */
export function builtinForLexeme(lexeme: string): Builtin | undefined {
  return lexemes.get(lexeme);
}

/**
 * Like builtinForLexeme, but also maps `in` and `IN` to Builtin.IN, which only
 * exist as builtins when lists are enabled.
 */
export function builtinForLexemeWithLists(lexeme: string): Builtin | undefined {
  return lexeme === 'in' || lexeme === 'IN'
    ? Builtin.IN
    : builtinForLexeme(lexeme);
}
