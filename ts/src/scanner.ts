import { RE2JSSyntaxException } from 're2js';

import { builtinForLexeme } from './builtin.ts';
import { ExprError, errorMessage, quoteRune } from './errors.ts';
import { type Literal, type Token, TokenType } from './token.ts';
import { Pattern } from './val.ts';

export const ScanErrorType = {
  UNEXPECTED_CHARACTER: 'UNEXPECTED_CHARACTER',
  INCOMPLETE_EQUALS: 'INCOMPLETE_EQUALS',
  INVALID_NUMERIC_LITERAL: 'INVALID_NUMERIC_LITERAL',
  UNTERMINATED_STRING: 'UNTERMINATED_STRING',
  UNTERMINATED_PATTERN: 'UNTERMINATED_PATTERN',
  INVALID_PATTERN_CHARACTER: 'INVALID_PATTERN_CHARACTER',
  INVALID_PATTERN: 'INVALID_PATTERN',
  PATTERN_TOO_LONG: 'PATTERN_TOO_LONG',
} as const;

export type ScanErrorType = (typeof ScanErrorType)[keyof typeof ScanErrorType];

export const MAX_PATTERN_LENGTH = 256;

const MAX_INT64 = 2n ** 63n - 1n;

export class ScanError extends ExprError {
  readonly type: ScanErrorType;
  /** The character the error was detected at. */
  readonly char: string;
  readonly pos: number;

  constructor(type: ScanErrorType, char: string, pos: number, cause?: Error) {
    super(`error scanning expression: ${type} scanning '${char}' at ${pos}`, {
      cause,
    });
    this.name = 'ScanError';
    this.type = type;
    this.char = char;
    this.pos = pos;
  }

  get from(): number {
    return this.pos;
  }

  get to(): number {
    return this.pos + this.char.length;
  }

  describe(): string {
    switch (this.type) {
      case ScanErrorType.UNEXPECTED_CHARACTER:
        return `Unexpected character ${quoteRune(this.char)}:`;
      case ScanErrorType.INCOMPLETE_EQUALS:
        return `Incomplete token, expected "==", found ${quoteRune(this.char)}:`;
      case ScanErrorType.INVALID_NUMERIC_LITERAL:
        return 'Invalid numeric literal, numbers can range from 0 to 2^63 - 1:';
      case ScanErrorType.UNTERMINATED_STRING:
        return 'Unterminated string starting here:';
      case ScanErrorType.UNTERMINATED_PATTERN:
        return 'Unterminated pattern starting here:';
      case ScanErrorType.INVALID_PATTERN_CHARACTER:
        return 'Invalid pattern character, only ASCII and Latin-1 are allowed in patterns:';
      case ScanErrorType.INVALID_PATTERN:
        return 'Syntax error in pattern:';
      case ScanErrorType.PATTERN_TOO_LONG:
        return `Pattern length exceeded, limit is ${MAX_PATTERN_LENGTH} characters:`;
    }
  }

  override asErrorMessage(expression: string): string {
    // The marker is always one character wide, as in the Go implementation.
    return errorMessage(this.describe(), expression, this.pos, 1);
  }
}

export interface ScanResult {
  /** The tokens scanned, ending with EOF if scanning succeeded. */
  readonly tokens: Token[];
  readonly error?: ScanError;
}

const keywords: ReadonlyMap<string, TokenType> = new Map([
  ['and', TokenType.AND],
  ['AND', TokenType.AND],
  ['or', TokenType.OR],
  ['OR', TokenType.OR],
  ['not', TokenType.NOT],
  ['NOT', TokenType.NOT],
  ['true', TokenType.TRUE],
  ['TRUE', TokenType.TRUE],
  ['false', TokenType.FALSE],
  ['FALSE', TokenType.FALSE],
]);

/**
 * Scan an expression into tokens.
 *
 * Never throws. If an error is detected the result has the error, and the
 * tokens scanned before it, which an editor can use to complete a
 * half-written expression.
 */
export function scan(source: string): ScanResult {
  return new Scanner(source).scan();
}

class Scanner {
  private readonly source: string;
  private start = 0;
  private current = 0;
  private readonly tokens: Token[] = [];

  constructor(source: string) {
    this.source = source;
  }

  scan(): ScanResult {
    while (!this.eof()) {
      this.start = this.current;
      try {
        this.scanToken();
      } catch (error) {
        if (error instanceof ScanError) {
          return { tokens: [...this.tokens], error };
        }
        throw error;
      }
    }

    this.tokens.push({
      type: TokenType.EOF,
      lexeme: '',
      charPos: this.current,
    });
    return { tokens: [...this.tokens] };
  }

  /** Scan a single token, adding it to `tokens`. */
  private scanToken(): void {
    const c = this.advance();
    switch (c) {
      case '(':
        return this.addToken(TokenType.LEFT_PAREN);
      case ')':
        return this.addToken(TokenType.RIGHT_PAREN);
      case '!':
        return this.addToken(
          this.match('=') ? TokenType.NOT_EQUAL : TokenType.NOT,
        );
      case '=':
        return this.equal();
      case '>':
        return this.addToken(
          this.match('=') ? TokenType.GREATER_EQUAL : TokenType.GREATER,
        );
      case '<':
        return this.addToken(
          this.match('=') ? TokenType.LESS_EQUAL : TokenType.LESS,
        );

      // Ignore whitespace
      case ' ':
      case '\t':
      case '\r':
      case '\n':
        return;

      case '"':
        return this.string();
      case '/':
        return this.pattern();
    }

    if (isDigit(c)) {
      return this.number();
    }
    if (isAlpha(c)) {
      return this.identifier();
    }
    throw new ScanError(ScanErrorType.UNEXPECTED_CHARACTER, c, this.start);
  }

  /** Match an "==" token. */
  private equal(): void {
    const c = this.peek();
    if (c !== '=') {
      throw new ScanError(ScanErrorType.INCOMPLETE_EQUALS, c, this.current);
    }
    this.advance();
    this.addToken(TokenType.EQUAL);
  }

  /**
   * Match a string token, from the starting " to the terminating ". The
   * literal value excludes the double-quotes.
   *
   * Double-quote and backslash characters can be embedded by escaping with
   * `\`: e.g. `\"`, `\\`
   */
  private string(): void {
    while (this.peek() !== '"' && !this.eof()) {
      if (this.peek() === '\\' && isEscapableChar(this.peekNext())) {
        this.advance();
      }
      this.advance();
    }

    if (this.eof()) {
      throw new ScanError(ScanErrorType.UNTERMINATED_STRING, '"', this.start);
    }

    this.advance();
    const v = this.source
      .slice(this.start + 1, this.current - 1)
      .replaceAll('\\"', '"')
      .replaceAll('\\\\', '\\');
    this.addToken(TokenType.STRING, v);
  }

  /**
   * Match a regular expression pattern token, from the starting / to the
   * terminating /. The literal value is the compiled pattern.
   *
   * Slash characters can be embedded by escaping with `\`: e.g. `\/`
   */
  private pattern(): void {
    while (this.peek() !== '/' && !this.eof()) {
      const c = this.peek();
      const next = this.peekNext();
      if (c === '\\' && next === '/') {
        this.advance();
      } else if (
        (c === '\\' && (next === 'u' || next === 'x')) ||
        (c.codePointAt(0) ?? 0) > 0xff
      ) {
        throw new ScanError(
          ScanErrorType.INVALID_PATTERN_CHARACTER,
          c,
          this.current,
        );
      }
      this.advance();
    }

    if (this.eof()) {
      throw new ScanError(ScanErrorType.UNTERMINATED_PATTERN, '/', this.start);
    }

    this.advance();
    // Escaped backslashes are left for the regexp engine, which also
    // interprets them. The only escape handled here is the delimiter.
    const v = this.source
      .slice(this.start + 1, this.current - 1)
      .replaceAll('\\/', '/');

    // The limit is in characters. Patterns are Latin-1, so that's their
    // length in UTF-16 code units.
    if (v.length > MAX_PATTERN_LENGTH) {
      throw new ScanError(ScanErrorType.PATTERN_TOO_LONG, '/', this.start);
    }

    let pattern: Pattern;
    try {
      pattern = new Pattern(v);
    } catch (error) {
      if (error instanceof RE2JSSyntaxException) {
        throw new ScanError(
          ScanErrorType.INVALID_PATTERN,
          '/',
          this.start,
          error,
        );
      }
      throw error;
    }
    this.addToken(TokenType.PATTERN, pattern);
  }

  private number(): void {
    while (isDigit(this.peek())) {
      this.advance();
    }

    const v = BigInt(this.lexeme());
    if (v > MAX_INT64) {
      throw new ScanError(
        ScanErrorType.INVALID_NUMERIC_LITERAL,
        this.source[this.start] ?? '',
        this.start,
      );
    }
    this.addToken(TokenType.NUMBER, v);
  }

  private identifier(): void {
    const continues = (): boolean =>
      isIdentifierTail(this.peek()) ||
      (this.peek() === '.' && isIdentifierTail(this.peekNext()));
    while (continues()) {
      this.advance();
    }

    const identifier = this.lexeme();
    if (builtinForLexeme(identifier) !== undefined) {
      this.addToken(TokenType.BUILTIN, identifier);
    } else {
      this.addToken(
        keywords.get(identifier) ?? TokenType.IDENTIFIER,
        identifier,
      );
    }
  }

  private addToken(type: TokenType, literal?: Literal): void {
    this.tokens.push({
      type,
      lexeme: this.lexeme(),
      ...(literal === undefined ? {} : { literal }),
      charPos: this.start,
    });
  }

  /**
   * Returns true, consuming it, if the next character in the source is
   * `expected`.
   */
  private match(expected: string): boolean {
    if (this.peek() !== expected) {
      return false;
    }
    this.advance();
    return true;
  }

  /** The next character, or '\0' at the end of the source. */
  private peek(): string {
    return charAt(this.source, this.current);
  }

  /** The character after next, or '\0' if there isn't one. */
  private peekNext(): string {
    return charAt(this.source, this.current + this.peek().length);
  }

  /** Consume and return the next character. */
  private advance(): string {
    const c = this.peek();
    this.current += c.length;
    return c;
  }

  private eof(): boolean {
    return this.current >= this.source.length;
  }

  private lexeme(): string {
    return this.source.slice(this.start, this.current);
  }
}

/** The code point at a UTF-16 offset, as a string, or '\0' past the end. */
function charAt(s: string, i: number): string {
  const cp = s.codePointAt(i);
  return cp === undefined ? '\0' : String.fromCodePoint(cp);
}

function isEscapableChar(c: string): boolean {
  return c === '"' || c === '\\';
}

export function isIdentifierTail(c: string): boolean {
  return isAlpha(c) || isDigit(c) || c === '-' || c === '_' || c === '?';
}

export function isAlpha(c: string): boolean {
  return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z');
}

function isDigit(c: string): boolean {
  return c >= '0' && c <= '9';
}
