package com.circleci.expr;

import java.util.StringJoiner;

public class Errors {
  public static String errorMessage(String preamble, String expression, int errorPos, int errorLength) {
    // This `+ 1` needs a little bit of explaining.
    // `String.lastIndexOf` returns the index of the last occurrence of a
    // newline working backwards from before the error position, which may
    // itself be a newline. If there isn't a newline preceding the error
    // position then `String.lastIndexOf` returns -1
    //
    // In either case, incrementing the result of `String.lastIndexOf` gets
    // the start index of the line the error occurred on:
    // 1. A newline was found: `.lastIndexOf` is the index of the newline,
    //    the next index is the start of the line.
    // 2. A newline was not found: `.lastIndexOf` is -1, the next index (0)
    //    is the start of the line.
    var lineStart = expression.lastIndexOf('\n', errorPos - 1) + 1;
    var lineEnd = expression.indexOf('\n', errorPos);
    // An error on a newline shows the rest of the expression.
    var errorEnd = lineEnd > errorPos ? lineEnd : expression.length();

    var errorLine = expression.substring(lineStart, errorEnd);

    StringJoiner sj = new StringJoiner("\n");
    return sj.add(preamble)
             .add(errorLine)
             .add(" ".repeat(errorPos - lineStart) + "^".repeat(errorLength))
             .toString();
  }

  /**
   * Format a character as a single-quoted literal, as Go's %q does:
   * characters that aren't printable, e.g. a non-breaking or zero-width
   * space, are escaped.
   */
  public static String quoteCodePoint(int c) {
    var escaped = switch (c) {
      case '\'' -> "\\'";
      case '\\' -> "\\\\";
      case 0x07 -> "\\a";
      case '\b' -> "\\b";
      case '\f' -> "\\f";
      case '\n' -> "\\n";
      case '\r' -> "\\r";
      case '\t' -> "\\t";
      case 0x0b -> "\\v";
      default -> isPrintable(c) ? Character.toString(c)
        : c < 0x20 || c == 0x7f ? String.format("\\x%02x", c)
        : c < 0x10000 ? String.format("\\u%04x", c)
        : String.format("\\U%08x", c);
    };
    return "'" + escaped + "'";
  }

  // Go's unicode.IsPrint: letters, marks, numbers, punctuation, symbols and
  // the ASCII space.
  private static boolean isPrintable(int c) {
    return c == ' ' || switch (Character.getType(c)) {
      case Character.UPPERCASE_LETTER, Character.LOWERCASE_LETTER,
           Character.TITLECASE_LETTER, Character.MODIFIER_LETTER,
           Character.OTHER_LETTER,
           Character.NON_SPACING_MARK, Character.ENCLOSING_MARK,
           Character.COMBINING_SPACING_MARK,
           Character.DECIMAL_DIGIT_NUMBER, Character.LETTER_NUMBER,
           Character.OTHER_NUMBER,
           Character.CONNECTOR_PUNCTUATION, Character.DASH_PUNCTUATION,
           Character.START_PUNCTUATION, Character.END_PUNCTUATION,
           Character.INITIAL_QUOTE_PUNCTUATION, Character.FINAL_QUOTE_PUNCTUATION,
           Character.OTHER_PUNCTUATION,
           Character.MATH_SYMBOL, Character.CURRENCY_SYMBOL,
           Character.MODIFIER_SYMBOL, Character.OTHER_SYMBOL -> true;
      default -> false;
    };
  }

  public static interface ErrorMessage {
    public String asErrorMessage(String expression);
  }
}
