/*
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to
deal in the Software without restriction, including without limitation the
rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
sell copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
IN THE SOFTWARE.
*/

package com.circleci.expr;

/* Grammar:
 * expression -> logic_or
 * logic_or -> logic_and ( "or" logic_and )*;
 * logic_and -> equality ( "and" equality )*;
 * equality -> comparison ( ( "==" | "!=" | "starts-with" | "matches" ) comparison )*;
 * comparison -> unary ( ( ">=" | ">" | "<=" | "<" ) unary)*;
 * unary -> "not" unary | primary;
 * primary -> "true" | "false" | NUMBER | STRING | IDENTIFIER | PATTERN | "(" expression ")"
 */

import java.util.List;

import static com.circleci.expr.Errors.ErrorMessage;
import static com.circleci.expr.TokenType.*;

/**
 * Parser performs a recursive descent parse of the input tokens.
 */
public class Parser {
  public static class ParseError extends RuntimeException implements ErrorMessage {
    static final long serialVersionUID = 1;

    public static enum Type {
      UNEXPECTED_ADDITIONAL_INPUT("Unexpected additional input."),
      EXPECTED_EXPRESSION("Expected expression."),
      EXPECTED_RIGHT_PAREN("Expected ')' after expression.");

      public final String message;
      private Type(String message) {
        this.message = message;
      }
    }

    public final Token token;
    public final Type type;

    public ParseError(Token token, Type type) {
      super(type.message);
      this.type = type;
      this.token = token;
    }

    /**
     * Return a multiline error message describing the error represented by
     * this exception.
     *
     * Returns a string of the form:
     *
     * explanation
     * line of expression containing the error
     * marker line pointing at the error token
     *
     * E.g.
     * "Expected expression, found \")\":\n" +
     * "foo and ) bar\n" +
     * "        ^"
     */
    @Override
    public String asErrorMessage(String expression) {
      var errorString = this.token.lexeme;

      return switch (this.type) {
        case UNEXPECTED_ADDITIONAL_INPUT ->
          Errors.errorMessage(String.format("Unexpected additional input, found \"%s\", expected EOF:", errorString),
              expression,
              this.token.charPos,
              errorString.length());

        case EXPECTED_EXPRESSION ->
          Errors.errorMessage(String.format("Expected expression, found \"%s\":", errorString),
              expression,
              this.token.charPos,
              errorString.length());

        case EXPECTED_RIGHT_PAREN ->
          Errors.errorMessage("Expected ')' after expression:",
              expression,
              this.token.charPos,
              1);
      };
    }
  }

  // List of tokens to assemble into an AST
  private final List<Token> tokens;
  // Index of the current token in the `tokens` list
  private int current = 0;

  /**
   * Create a parser to parse the given list of Tokens.
   */
  public Parser(List<Token> tokens) {
    this.tokens = tokens;
  }

  /**
   * Parse the `tokens` list into an abstract syntax tree.
   *
   * Returns the root node of the AST.
   *
   * @throws ParseError If an error occurs during parsing.
   */
  public Expr parse() {
    Expr expr =  expression();
    if (!eof()) throw new ParseError(peek(), ParseError.Type.UNEXPECTED_ADDITIONAL_INPUT);

    return expr;
  }

  /**
   * Match an 'expression' production.
   */
  private Expr expression() {
    return or();
  }

  /**
   * Match an 'or' production.
   */
  private Expr or() {
    Expr expr = and();

    while (match(OR)) {
      Token operator = previous();
      Expr right = and();
      expr = new Expr.Logical(expr, operator, right);
    }

    return expr;
  }

  /**
   * Match an 'and' production.
   */
  private Expr and() {
    Expr expr = equality();

    while (match(AND)) {
      Token operator = previous();
      Expr right = equality();
      expr = new Expr.Logical(expr, operator, right);
    }

    return expr;
  }

  /**
   * Match an 'equality' production.
   */
  private Expr equality() {
    Expr expr = comparison();

    while (match(EQUAL, NOT_EQUAL, STARTS_WITH, MATCHES)) {
      Token operator = previous();
      Expr right = comparison();
      expr = new Expr.Binary(expr, operator, right);
    }

    return expr;
  }

  /**
   * Match a 'comparison' production.
   */
  private Expr comparison() {
    Expr expr = unary();

    while (match(GREATER, GREATER_EQUAL, LESS, LESS_EQUAL)) {
      Token operator = previous();
      Expr right = unary();
      expr = new Expr.Binary(expr, operator, right);
    }

    return expr;
  }

  /**
   * Match a 'unary' production.
   */
  private Expr unary() {
    if (match(NOT)) {
      Token operator = previous();
      Expr right = unary();
      return new Expr.Unary(operator, right);
    }

    return primary();
  }

  /**
   * Match a 'primary' production.
   */
  private Expr primary() {
    if (match(FALSE)) return new Expr.Literal(false);
    if (match(TRUE)) return new Expr.Literal(true);

    if (match(NUMBER, STRING, PATTERN)) {
      return new Expr.Literal(previous().literal);
    }

    if (match(IDENTIFIER)) {
      return new Expr.Identifier(previous());
    }

    if (match(LEFT_PAREN)) {
      Expr expr = expression();
      consume(RIGHT_PAREN, ParseError.Type.EXPECTED_RIGHT_PAREN);
      return new Expr.Grouping(expr);
    }

    throw new ParseError(peek(), ParseError.Type.EXPECTED_EXPRESSION);
  }

  /**
   * Attempt to match a token of one of the input types.
   *
   * Consumes the token and returns true if there is a match. Returns false
   * without consuming any tokens otherwise.
   */
  private boolean match(TokenType... types) {
    for (var type: types) {
      if (check(type)) {
        advance();
        return true;
      }
    }

    return false;
  }

  /**
   * Consume a token of the expected type.
   *
   * Returns the token if the next token in the input is of the expected type.
   *
   * @throws ParseError if the next token does not match the expected type.
   */
  private Token consume(TokenType expected, ParseError.Type errorType) {
    if (check(expected)) return advance();

    throw new ParseError(peek(), errorType);
  }

  /**
   * Check if the next token is of the expected type.
   *
   * Returns true if the type is as expected, false otherwise.
   */
  private boolean check(TokenType type) {
    if (eof()) return false;
    return peek().type ==  type;
  }

  /**
   * Returns the token one before the current position in the input token list.
   */
  private Token previous() {
    return tokens.get(current - 1);
  }

  /**
   * Advance the pointer in the token list.
   *
   * Returns the token that the pointer was pointing to.
   */
  private Token advance() {
    if (!eof()) current++;
    return previous();
  }

  /**
   * Returns true if the next token that would be returned by 'advance' is EOF.
   * False otherwise.
   */
  private boolean eof() {
    return peek().type == EOF;
  }

  /**
   * Return the next token in the input token list without advancing.
   */
  private Token peek() {
    return tokens.get(current);
  }
}
