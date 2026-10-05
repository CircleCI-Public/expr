/**
 * Base class of the errors produced while scanning, parsing or interpreting
 * an expression.
 *
 * `from` and `to` are the UTF-16 offsets of the region of the expression the
 * error refers to, suitable for an editor diagnostic.
 */
export abstract class ExprError extends Error {
  abstract readonly from: number;
  abstract readonly to: number;

  /** A one-line description of the error, ending with a colon. */
  abstract describe(): string;

  /**
   * Return a multiline error message describing this error, of the form:
   *
   *     explanation
   *     line of expression containing the error
   *     marker line pointing at the error token
   *
   * E.g.
   *
   *     Unexpected character "&":
   *     2 > 5 && false
   *           ^
   */
  asErrorMessage(expression: string): string {
    return errorMessage(
      this.describe(),
      expression,
      this.from,
      this.to - this.from,
    );
  }
}

export function errorMessage(
  preamble: string,
  expression: string,
  errorPos: number,
  errorLength: number,
): string {
  // lastIndexOf returns -1 when there is no preceding newline, so either way
  // the line starts at the next index.
  const lineStart = expression.lastIndexOf('\n', errorPos - 1) + 1;
  const lineEndOffset = expression.slice(errorPos).indexOf('\n');

  // Matches the Go implementation: an error positioned on a newline shows the
  // rest of the expression.
  const errorEnd =
    lineEndOffset > 0 ? errorPos + lineEndOffset : expression.length;

  const errorLine = expression.slice(lineStart, errorEnd);
  const markerLine = ' '.repeat(errorPos - lineStart) + '^'.repeat(errorLength);

  return [preamble, errorLine, markerLine].join('\n');
}

const runeEscapes: Readonly<Record<string, string>> = {
  "'": "\\'",
  '\\': '\\\\',
  '\x07': '\\a',
  '\b': '\\b',
  '\f': '\\f',
  '\n': '\\n',
  '\r': '\\r',
  '\t': '\\t',
  '\v': '\\v',
};

// Go's unicode.IsPrint: letters, marks, numbers, punctuation, symbols and the
// ASCII space.
const printable = /^[\p{L}\p{M}\p{N}\p{P}\p{S} ]$/u;

/**
 * Formats a character as a single-quoted literal, as Go's %q does: characters
 * that aren't printable, e.g. a non-breaking or zero-width space, are escaped.
 */
export function quoteRune(c: string): string {
  const escaped = runeEscapes[c];
  if (escaped !== undefined) {
    return `'${escaped}'`;
  }
  if (printable.test(c)) {
    return `'${c}'`;
  }
  const code = c.codePointAt(0) ?? 0;
  const hex = code.toString(16);
  if (code < 0x20 || code === 0x7f) {
    return `'\\x${hex.padStart(2, '0')}'`;
  }
  if (code < 0x10000) {
    return `'\\u${hex.padStart(4, '0')}'`;
  }
  return `'\\U${hex.padStart(8, '0')}'`;
}
