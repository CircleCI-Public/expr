/* Grammar:
 * expression -> logic_or
 * logic_or -> logic_and ( "or" logic_and )*;
 * logic_and -> equality ( "and" equality )*;
 * equality -> comparison ( ( "==" | "!=" | BUILTIN ) comparison )*;
 *   (when lists are enabled, the right operand of `in` is a list)
 * list -> "[" literal ( "," literal )* "]";
 * literal -> NUMBER | STRING | "true" | "false";
 * comparison -> unary ( ( ">=" | ">" | "<=" | "<" ) unary)*;
 * unary -> "not" unary | primary;
 * primary -> "true" | "false" | NUMBER | STRING | IDENTIFIER | PATTERN | "(" expression ")"
 */

import type { Expr, ListExpr, LiteralExpr } from './expr.ts';
import { Builtin, builtinForLexemeWithLists } from './builtin.ts';
import { ExprError } from './errors.ts';
import type { Options } from './options.ts';
import { type Token, TokenType } from './token.ts';

export const ParseErrorType = {
  UNEXPECTED_ADDITIONAL_INPUT: 'UNEXPECTED_ADDITIONAL_INPUT',
  EXPECTED_EXPRESSION: 'EXPECTED_EXPRESSION',
  EXPECTED_RIGHT_PAREN: 'EXPECTED_RIGHT_PAREN',
  UNKNOWN_BUILTIN_FUNCTION: 'UNKNOWN_BUILTIN_FUNCTION',
  EXPECTED_LIST: 'EXPECTED_LIST',
  EMPTY_LIST: 'EMPTY_LIST',
  EXPECTED_LIST_ELEMENT: 'EXPECTED_LIST_ELEMENT',
  EXPECTED_RIGHT_BRACKET: 'EXPECTED_RIGHT_BRACKET',
  MIXED_LIST_TYPES: 'MIXED_LIST_TYPES',
  UNEXPECTED_LIST: 'UNEXPECTED_LIST',
} as const;

export type ParseErrorType =
  (typeof ParseErrorType)[keyof typeof ParseErrorType];

export class ParseError extends ExprError {
  readonly type: ParseErrorType;
  readonly token: Token;

  constructor(type: ParseErrorType, token: Token) {
    super(
      `error parsing: ${type} at <token ${token.type} ${token.lexeme}>[${token.charPos}]`,
    );
    this.name = 'ParseError';
    this.type = type;
    this.token = token;
  }

  get from(): number {
    return this.token.charPos;
  }

  get to(): number {
    // EXPECTED_RIGHT_PAREN marks a single character, even at EOF.
    const length =
      this.type === ParseErrorType.EXPECTED_RIGHT_PAREN
        ? 1
        : this.token.lexeme.length;
    return this.token.charPos + length;
  }

  describe(): string {
    const lexeme = this.token.lexeme;
    switch (this.type) {
      case ParseErrorType.UNEXPECTED_ADDITIONAL_INPUT:
        return `Unexpected additional input, found "${lexeme}", expected EOF:`;
      case ParseErrorType.EXPECTED_EXPRESSION:
        return `Expected expression, found "${lexeme}":`;
      case ParseErrorType.EXPECTED_RIGHT_PAREN:
        return "Expected ')' after expression:";
      case ParseErrorType.UNKNOWN_BUILTIN_FUNCTION:
        return 'Unknown infix function:';
      case ParseErrorType.EXPECTED_LIST:
        return `Expected '[' after "in", found "${lexeme}":`;
      case ParseErrorType.EMPTY_LIST:
        return 'List must not be empty:';
      case ParseErrorType.EXPECTED_LIST_ELEMENT:
        return `Expected a string, number or boolean list element, found "${lexeme}":`;
      case ParseErrorType.EXPECTED_RIGHT_BRACKET:
        return `Expected ',' or ']' after list element, found "${lexeme}":`;
      case ParseErrorType.MIXED_LIST_TYPES:
        return 'List elements must all have the same type:';
      case ParseErrorType.UNEXPECTED_LIST:
        return 'Unexpected list, a list can only follow "in":';
    }
  }
}

/**
 * Parse a list of tokens, ending in EOF, into an abstract syntax tree.
 *
 * Throws ParseError if the tokens don't form a valid expression, or a
 * RangeError if they're too deeply nested, as parsing recurses for each level
 * of nesting.
 */
export function parse(tokens: readonly Token[], options?: Options): Expr {
  return new Parser(tokens, options?.lists === true).parse();
}

class Parser {
  private readonly tokens: readonly Token[];
  private readonly lists: boolean;
  private current = 0;

  constructor(tokens: readonly Token[], lists: boolean) {
    if (tokens.at(-1)?.type !== TokenType.EOF) {
      throw new TypeError('the token list must end with an EOF token');
    }
    this.tokens = tokens;
    this.lists = lists;
  }

  parse(): Expr {
    const expr = this.expression();

    if (!this.eof()) {
      this.checkUnrecognisedFunction();
      throw new ParseError(
        ParseErrorType.UNEXPECTED_ADDITIONAL_INPUT,
        this.peek(),
      );
    }

    return expr;
  }

  /**
   * Throw a ParseError if the remaining tokens look like a call of an unknown
   * builtin function.
   *
   * If an invalid name is used for a builtin function the stream of tokens
   * will look like:
   *
   *     [...tokens that parse as `comparison`..., IDENT, ...more tokens...]
   *
   * The tokens to the left of IDENT can terminate an expression. IDENT and the
   * following tokens (if any) are extra input. If the tokens following IDENT
   * also parse as a `comparison` then it's very likely that IDENT is a
   * misspelled builtin function name.
   */
  private checkUnrecognisedFunction(): void {
    const pos = this.current;

    if (this.match(TokenType.IDENTIFIER)) {
      const potentialBuiltin = this.previous();
      let operand = true;
      try {
        this.comparison();
      } catch (error) {
        if (!(error instanceof ParseError)) {
          throw error;
        }
        operand = false;
      }
      if (operand) {
        throw new ParseError(
          ParseErrorType.UNKNOWN_BUILTIN_FUNCTION,
          potentialBuiltin,
        );
      }
    }

    // The rest of the tokens don't parse as an operand, so we can't assume
    // anything; undo any token consumption.
    this.current = pos;
  }

  private expression(): Expr {
    return this.or();
  }

  private or(): Expr {
    let expr = this.and();
    while (this.match(TokenType.OR)) {
      const operator = this.previous();
      const right = this.and();
      expr = { kind: 'logical', left: expr, operator, right };
    }
    return expr;
  }

  private and(): Expr {
    let expr = this.equality();
    while (this.match(TokenType.AND)) {
      const operator = this.previous();
      const right = this.equality();
      expr = { kind: 'logical', left: expr, operator, right };
    }
    return expr;
  }

  private equality(): Expr {
    let expr = this.comparison();
    while (
      this.match(TokenType.EQUAL, TokenType.NOT_EQUAL, TokenType.BUILTIN)
    ) {
      const operator = this.previous();
      const builtin = builtinForLexemeWithLists(operator.lexeme);
      const right = builtin === Builtin.IN ? this.list() : this.comparison();
      expr =
        operator.type === TokenType.BUILTIN && builtin !== undefined
          ? { kind: 'infix', left: expr, operator, builtin, right }
          : { kind: 'binary', left: expr, operator, right };
    }
    return expr;
  }

  private list(): ListExpr {
    if (!this.match(TokenType.LEFT_BRACKET)) {
      throw new ParseError(ParseErrorType.EXPECTED_LIST, this.peek());
    }
    const open = this.previous();

    if (this.peek().type === TokenType.RIGHT_BRACKET) {
      throw new ParseError(ParseErrorType.EMPTY_LIST, this.peek());
    }

    const elements: LiteralExpr[] = [];
    do {
      const token = this.peek();
      const element = this.listElement();
      if (
        elements.length > 0 &&
        typeof element.value !== typeof elements[0]!.value
      ) {
        throw new ParseError(ParseErrorType.MIXED_LIST_TYPES, token);
      }
      elements.push(element);
    } while (this.match(TokenType.COMMA));

    if (!this.match(TokenType.RIGHT_BRACKET)) {
      throw new ParseError(ParseErrorType.EXPECTED_RIGHT_BRACKET, this.peek());
    }
    return { kind: 'list', open, elements };
  }

  private listElement(): LiteralExpr {
    if (this.match(TokenType.FALSE)) {
      return { kind: 'literal', value: false };
    }
    if (this.match(TokenType.TRUE)) {
      return { kind: 'literal', value: true };
    }
    if (this.match(TokenType.NUMBER, TokenType.STRING)) {
      const value = this.previous().literal;
      if (value === undefined) {
        throw new TypeError('literal token has no value');
      }
      return { kind: 'literal', value };
    }
    throw new ParseError(ParseErrorType.EXPECTED_LIST_ELEMENT, this.peek());
  }

  private comparison(): Expr {
    let expr = this.unary();
    while (
      this.match(
        TokenType.GREATER,
        TokenType.GREATER_EQUAL,
        TokenType.LESS,
        TokenType.LESS_EQUAL,
      )
    ) {
      const operator = this.previous();
      const right = this.unary();
      expr = { kind: 'binary', left: expr, operator, right };
    }
    return expr;
  }

  private unary(): Expr {
    if (this.match(TokenType.NOT)) {
      const operator = this.previous();
      const right = this.unary();
      return { kind: 'unary', operator, right };
    }
    return this.primary();
  }

  private primary(): Expr {
    if (this.match(TokenType.FALSE)) {
      return { kind: 'literal', value: false };
    }
    if (this.match(TokenType.TRUE)) {
      return { kind: 'literal', value: true };
    }

    if (this.match(TokenType.NUMBER, TokenType.STRING, TokenType.PATTERN)) {
      const value = this.previous().literal;
      if (value === undefined) {
        throw new TypeError('literal token has no value');
      }
      return { kind: 'literal', value };
    }

    if (this.match(TokenType.IDENTIFIER)) {
      return { kind: 'identifier', name: this.previous() };
    }

    if (this.match(TokenType.LEFT_PAREN)) {
      const expression = this.expression();
      if (!this.match(TokenType.RIGHT_PAREN)) {
        throw new ParseError(ParseErrorType.EXPECTED_RIGHT_PAREN, this.peek());
      }
      return { kind: 'grouping', expression };
    }

    if (this.lists && this.peek().type === TokenType.LEFT_BRACKET) {
      throw new ParseError(ParseErrorType.UNEXPECTED_LIST, this.peek());
    }

    throw new ParseError(ParseErrorType.EXPECTED_EXPRESSION, this.peek());
  }

  /** Consume the next token if it's one of `types`. */
  private match(...types: TokenType[]): boolean {
    if (!this.eof() && types.includes(this.peek().type)) {
      this.current++;
      return true;
    }
    return false;
  }

  private previous(): Token {
    return this.tokens[this.current - 1]!;
  }

  private eof(): boolean {
    return this.peek().type === TokenType.EOF;
  }

  private peek(): Token {
    return this.tokens[this.current]!;
  }
}
