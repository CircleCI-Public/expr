package com.circleci.expr;

import java.util.StringJoiner;

public class Errors {
  public static String errorMessage(String preamble, String expression, int errorPos, int errorLength) {
    // This `+ 1` needs a little bit of explaining.
    // `String.lastIndexOf` returns the index of the last occurrence of a
    // newline working backwards from the error position. If there isn't a
    // newline preceding the error position then `String.lastIndexOf`
    // returns -1
    //
    // In either case, incrementing the result of `String.lastIndexOf` gets
    // the start index of the line the error occurred on:
    // 1. A newline was found: `.lastIndexOf` is the index of the newline,
    //    the next index is the start of the line.
    // 2. A newline was not found: `.lastIndexOf` is -1, the next index (0)
    //    is the start of the line.
    var lineStart = expression.lastIndexOf('\n', errorPos) + 1;
    var lineEnd = expression.indexOf('\n', errorPos);
    var errorEnd = lineEnd > 0 ? lineEnd : expression.length();

    var errorLine = expression.substring(lineStart, errorEnd);

    StringJoiner sj = new StringJoiner("\n");
    return sj.add(preamble)
             .add(errorLine)
             .add(" ".repeat(errorPos - lineStart) + "^".repeat(errorLength))
             .toString();
  }

  public static interface ErrorMessage {
    public String asErrorMessage(String expression);
  }
}
