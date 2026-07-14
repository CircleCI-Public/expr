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
 * equality -> comparison ( ( "==" | "!=" | BUILTIN ) comparison )*;
 * comparison -> unary ( ( ">=" | ">" | "<=" | "<" ) unary)*;
 * unary -> "not" unary | primary;
 * primary -> "true" | "false" | NUMBER | STRING | IDENTIFIER | PATTERN | "(" expression ")"
 */

import java.util.List;

import com.circleci.expr.Builtin;

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
      EXPECTED_RIGHT_PAREN("Expected ')' after expression."),
      UNKNOWN_BUILTIN_FUNCTION("Unknown infix function.");

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

        case UNKNOWN_BUILTIN_FUNCTION ->
          Errors.errorMessage("Unknown infix function:",
              expression,
              this.token.charPos,
              errorString.length());
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
    if (!eof()) {
      checkUnrecognisedFunction();

      throw new ParseError(peek(), ParseError.Type.UNEXPECTED_ADDITIONAL_INPUT);
    }

    return expr;
  }

  /**
   * Throw ParseError if remaining tokens match the shape of a builtin function
   * call.
   *
   * If an invalid name is used for a builtin function the stream of tokens
   * will look like:
   * [...tokens that parse as `comparison`..., IDENT, ...more tokens...]
   *
   * The tokens to the left of IDENT can terminate an expression. IDENT and the
   * following tokens (if any) are extra input.
   *
   * If the tokens following IDENT also parse as a `comparison` then it's very
   * likely that IDENT is a misspelled builtin function name.
   */
  private void checkUnrecognisedFunction() {
    var pos = save();

    // Might be an unrecognised function name
    if (match(IDENTIFIER)) {
      Token potentialBuiltin = previous();

      try {
        // If the rest of the token stream parses as a valid operand then we
        // probably have an invalid function name in potentialBuiltin.
        Expr rest = comparison();
      }
      catch (ParseError pe) {
        // If the rest of the token stream doesn't parse as a valid operand
        // then we can't assume anything, undo any token consumption
        restore(pos);
        return;
      }

      throw new ParseError(potentialBuiltin, ParseError.Type.UNKNOWN_BUILTIN_FUNCTION);
    }
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

    while (match(EQUAL, NOT_EQUAL, BUILTIN)) {
      Token operator = previous();

      if (operator.type == BUILTIN) {
        var builtin = Builtin.forLexeme(operator.lexeme);

        Expr right = comparison();
        expr = new Expr.Infix(expr, operator, builtin, right);
      }
      else {
        Expr right = comparison();
        expr = new Expr.Binary(expr, operator, right);
      }
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
   * Restore the position in the token stream to one returned by save().
   */
  private void restore(int pos) {
    if (pos >= 0 && pos < tokens.size()) {
      current = pos;
    }
  }

  /**
   * Returns the current position in the token stream. Can revert to this
   * position in the stream with restore().
   */
  private int save() {
    return current;
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
