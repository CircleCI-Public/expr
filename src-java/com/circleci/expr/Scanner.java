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

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import com.google.re2j.Pattern;
import com.google.re2j.PatternSyntaxException;

import static com.circleci.expr.Errors.ErrorMessage;
import static com.circleci.expr.TokenType.*;

public class Scanner {
  public static class ScanError extends RuntimeException implements ErrorMessage {
    static final long serialVersionUID = 1;

    public static enum Type {
      UNEXPECTED_CHARACTER("Unexpected character."),
      INCOMPLETE_EQUALS("Incomplete token, expected \"==\"."),
      UNTERMINATED_STRING("Unterminated string."),
      UNTERMINATED_PATTERN("Unterminated pattern."),
      INVALID_PATTERN_CHARACTER("Invalid pattern character."),
      INVALID_PATTERN("Invalid pattern.");

      public final String message;
      private Type(String message) {
        this.message = message;
      }
    }

    public final Type type;
    public final char errorChar;
    public final int errorPos;

    public ScanError(char errorChar, int errorPos, Type type) {
      super(type.message);
      this.type = type;
      this.errorChar = errorChar;
      this.errorPos = errorPos;
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
     * "Unexpected character '&':\n" +
     * "2 > 5 && false\n" +
     * "      ^"
     */
    @Override
    public String asErrorMessage(String expression) {
      return switch (this.type) {
        case UNEXPECTED_CHARACTER ->
          Errors.errorMessage(String.format("Unexpected character '%s':" , this.errorChar),
              expression,
              this.errorPos,
              1);

        case INCOMPLETE_EQUALS ->
          Errors.errorMessage(String.format("Incomplete token, expected \"==\", found '%s':", this.errorChar),
              expression,
              this.errorPos,
              1);

        case UNTERMINATED_STRING ->
          Errors.errorMessage(String.format("Unterminated string starting here:", this.errorChar),
              expression,
              this.errorPos,
              1);

      case UNTERMINATED_PATTERN ->
        Errors.errorMessage(String.format("Unterminated pattern starting here:", this.errorChar),
            expression,
            this.errorPos,
            1);

      case INVALID_PATTERN_CHARACTER ->
        Errors.errorMessage(String.format("Invalid pattern character, only ASCII and Latin-1 are allowed in patterns:", this.errorChar),
            expression,
            this.errorPos,
            1);

      case INVALID_PATTERN ->
        Errors.errorMessage(String.format("Syntax error in pattern:", this.errorChar),
            expression,
            this.errorPos,
            1);
      };
    }
  }

  private static final Map<String, TokenType> keywords;
  static {
    keywords = new HashMap<>();
    keywords.put("starts-with", STARTS_WITH);
    keywords.put("STARTS-WITH", STARTS_WITH);
    keywords.put("matches", MATCHES);
    keywords.put("MATCHES", MATCHES);
    keywords.put("and", AND);
    keywords.put("AND", AND);
    keywords.put("or", OR);
    keywords.put("OR", OR);
    keywords.put("not", NOT);
    keywords.put("NOT", NOT);
    keywords.put("true", TRUE);
    keywords.put("TRUE", TRUE);
    keywords.put("false", FALSE);
    keywords.put("FALSE", FALSE);
  }

  private String source;
  // 'start' tracks the index of the start of the current lexeme in 'source'
  private int start = 0;
  // 'current' tracks the index of the current character in 'source'
  private int current = 0;

  private List<Token> tokens = new ArrayList<>();

  public Scanner(String source) {
    this.source = source;
  }

  /**
   * Scans the source string and produces a List of Tokens scanned from the
   * String.
   *
   * @throws ScanError if an error is detected while scanning the source
   * string. See ScanError.Type for possible error cases.
   *
   * @see Token
   * @see TokenType
   */
  public List<Token> scan() {
    while (!eof()) {
      start = current;
      scanToken();
    }

    tokens.add(new Token(EOF, "", null, current));

    return tokens;
  }

  /**
   * Scan a single token.
   *
   * Consumes characters and adds created tokens to the `tokens` list.
   */
  private void scanToken() {
    char c = advance();
    switch (c) {
      case '(': addToken(LEFT_PAREN); break;
      case ')': addToken(RIGHT_PAREN); break;
      case '!': addToken(match('=') ? NOT_EQUAL : NOT); break;
      case '=': equal(); break;
      case '>': addToken(match('=') ? GREATER_EQUAL : GREATER); break;
      case '<': addToken(match('=') ? LESS_EQUAL : LESS); break;

      // Ignore whitespace
      case ' ':
      case '\t':
      case '\r':
      case '\n':
        break;

      case '"': string(); break;
      case '/': pattern(); break;
      default:
        if (isDigit(c)) {
          number();
        }
        else if (isAlpha(c)) {
          identifier();
        }
        else {
          // after advance(), current points at the _next_ character
          throw new ScanError(c, current - 1, ScanError.Type.UNEXPECTED_CHARACTER);
        }
        break;
    }
  }

  /**
   * Match an "==" token.
   *
   * Consumes the token's characters and adds a Token to the `tokens` list.
   */
  private void equal() {
    char c = peek();
    if (c != '=') {
      throw new ScanError(c, current, ScanError.Type.INCOMPLETE_EQUALS);
    }
    advance();
    addToken(EQUAL);
  }

  /**
   * Match a string token.
   *
   * Consumes all characters from the starting " to the terminating ". Adds a
   * Token with the string content (excluding surrounding double-quotes) to the
   * `tokens` list.
   *
   * Double-quote and backslach characters can be embedded by escaping with
   * `\`: e.g. `\"`, `\\`
   *
   * @throws ScanError if the string is not terminated.
   */
  private void string() {
    while (peek() != '"' && !eof()) {
      // Deal with escape characters
      if (peek() == '\\' && isEscapableChar(peekNext())) {
        advance();
      }
      advance();
    }

    if (eof()) {
      throw new ScanError('"', start, ScanError.Type.UNTERMINATED_STRING);
    }

    // Consume the double-qoute, we only peek()ed above
    advance();
    String value = source.substring(start + 1, current - 1)
                         .replace("\\\"", "\"")
                         .replace("\\\\", "\\");
    addToken(STRING, value);
  }

  /**
   * Match a regular expression pattern token.
   *
   * Consumes all characters from the starting / to the terminating /. Adds a
   * Token with the string content (excluding surrounding slashes) to the
   * `tokens` list.
   *
   * Slash and backslach characters can be embedded by escaping with
   * `\`: e.g. `\/`, `\\`
   *
   * @throws ScanError if the pattern is not terminated.
   */
  private void pattern() {
    while (peek() != '/' && !eof()) {
      // Deal with escape characters
      var c = peek();
      var cnext = peekNext();
      if (c == '\\' && isEscapablePatternChar(cnext)) {
        advance();
      }
      else if (c == '\\' && cnext == 'u') {
        throw new ScanError('\\', current, ScanError.Type.INVALID_PATTERN_CHARACTER);
      }
      else if (c > 0xff) {
        throw new ScanError(c, current, ScanError.Type.INVALID_PATTERN_CHARACTER);
      }
      advance();
    }

    if (eof()) {
      throw new ScanError('/', start, ScanError.Type.UNTERMINATED_PATTERN);
    }

    // Consume the slash, we only peek()ed above
    advance();
    // Note: we do not evaluate escaped backslashes in the pattern. The regexp
    // engine also interprets backslash escape sequences, so the only character
    // we deal with ourselves is the one we added, the pattern delimiter `/`
    String value = source.substring(start + 1, current - 1)
                         .replace("\\/", "/");
    try {
      Pattern pattern = Pattern.compile(value);
      addToken(PATTERN, pattern);
    }
    catch (PatternSyntaxException e) {
      throw new ScanError('/', start, ScanError.Type.INVALID_PATTERN);
    }
  }

  /**
   * Returns true if `c` is a character that could be escaped.
   *
   * False otherwise.
   */
  private boolean isEscapableChar(char c) {
    return c == '"' || c == '\\';
  }

  private boolean isEscapablePatternChar(char c) {
    return c == '/' || c == '\\';
  }

  private void number() {
    while (isDigit(peek())) {
      advance();
    }

    addToken(NUMBER, Long.parseLong(lexeme()));
  }

  /**
   * Match an identifier.
   *
   * Consumes the token's characters and adds a Token to the `tokens` list.
   */
  private void identifier() {
    char c = peek();
    if (isIdentifierTail(c) || (c == '.' && isIdentifierTail(peekNext()))) {
      advance();
      c = peek();
      while (isIdentifierTail(c) || (c == '.' && isIdentifierTail(peekNext()))) {
        advance();
        c = peek();
      }
    }

    String identifier = lexeme();
    TokenType type = keywords.getOrDefault(identifier, IDENTIFIER);
    addToken(type, identifier);
  }

  /**
   * Add a token without a literal value to the `tokens` list.
   */
  private void addToken(TokenType type) {
    addToken(type, null);
  }

  /**
   * Add a token with a literal value to the `tokens` list.
   */
  private void addToken(TokenType type, Object literal) {
    tokens.add(new Token(type, lexeme(), literal, start));
  }

  /**
   * Returns true if `c` is allowed in the "tail" of an identifier.
   *
   * The "tail" is every part of the identifier other than the first character.
   * Returns true if `c` is alpha-numeric, or '-', or '_', false otherwise.
   *
   * @see isAlpha
   * @see isDigit
   */
  private boolean isIdentifierTail(char c) {
    return isAlpha(c) || isDigit(c) || c == '-' || c == '_' || c == '?';
  }

  /**
   * Returns true if `c` is an alphabetic character false otherwise.
   *
   * Accepts:
   * - 'a' to 'z'
   * - 'A' to 'Z'
   * - '_'
   */
  private boolean isAlpha(char c) {
    return c >= 'a' && c <= 'z'
            || c >= 'A' && c <= 'Z';
  }

  /**
   * Returns true if `c` is any of the characters '0' through '9' (inclusive).
   *
   * False otherwise.
   */
  private boolean isDigit(char c) {
    // NB this exploits the way that ASCII and UTF-8 lay out the characters for
    // digits in sequence.
    return c >= '0' && c <= '9';
  }

  /**
   * Returns true if the next character in the source is the same as 'expected'.
   * If there is a match it is consumed.
   * If there is no match the character source is unchanged.
   */
  private boolean match(char expected) {
    if (eof()) return false;
    if (peek() != expected) return false;

    advance();
    return true;
  }

  /**
   * Gets the next character in the source without consuming.
   *
   * Returns \0 if there is no character to peek.
   */
  private char peek() {
    if (eof()) return '\0';
    return source.charAt(current);
  }

  /**
   * Gets the next + 1 character in the source without consuming.
   *
   * Returns \0 if there is no character to peekNext.
   */
  private char peekNext() {
    if (eof(current + 1)) return '\0';
    return source.charAt(current + 1);
  }

  /**
   * Consume and return the next character from source.
   */
  private char advance() {
    return source.charAt(current++);
  }

  /**
   * Returns true if the end of source has been reached.
   * False otherwise
   */
  private boolean eof() {
    return eof(current);
  }

  /**
   * Returns true if `pos` is beyond the end of source.
   * False otherwise.
   */
  private boolean eof(int pos) {
    return pos >= source.length();
  }

  /**
   * Return the current lexeme.
   */
  private String lexeme() {
    return source.substring(start, current);
  }
}
