package com.circleci.expr;

import java.nio.charset.StandardCharsets;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import com.google.re2j.Pattern;

/**
 * A Jazzer fuzz target that checks that scanning, parsing, analysing, type
 * checking and interpreting any expression either succeeds or throws one of
 * expr's errors, that the error's message can be formatted, and that an
 * expression whose types check interprets without error.
 *
 * Run with: mkdir -p target/fuzz-corpus target/fuzz-findings && lein with-profile +fuzz fuzz
 */
public class ExprFuzzer {
  // A variable of each kind of value, so fuzzed expressions that refer to
  // them reach every operator's type checks.
  private static final Map<String, Object> ENV = new HashMap<>();
  static {
    ENV.put("b", true);
    ENV.put("f", false);
    ENV.put("n", 42L);
    ENV.put("s", "foo");
    ENV.put("e", "");
    ENV.put("p", Pattern.compile("fo+"));
    ENV.put("u", null);
  }

  // ENV's types; u, being undefined, has none. Change it with ENV, or the
  // fuzzer reports type checks that the interpreter contradicts.
  private static final Map<String, String> TYPES = Map.of(
      "b", "boolean",
      "f", "boolean",
      "n", "number",
      "s", "string",
      "e", "string",
      "p", "pattern");

  public static void fuzzerTestOneInput(byte[] data) {
    var expression = new String(data, StandardCharsets.UTF_8);

    List<Token> tokens;
    try {
      tokens = new Scanner(expression).scan();
    }
    catch (Scanner.ScanError e) {
      checkError(expression, e.errorPos, e);
      return;
    }

    Expr expr;
    try {
      expr = new Parser(tokens).parse();
    }
    catch (Parser.ParseError e) {
      checkError(expression, e.token.charPos, e);
      return;
    }

    new VariableAnalyser().gatherVariables(expr);

    var typesCheck = true;
    try {
      new TypeChecker(TYPES).checkTypes(expr);
    }
    catch (TypeChecker.Error e) {
      checkError(expression, e.token.charPos, e);
      typesCheck = false;
    }

    var interpreter = new Interpreter(ENV);
    Object value;
    try {
      value = interpreter.evaluate(expr);
    }
    catch (Interpreter.Error e) {
      checkError(expression, e.token.charPos, e);
      if (typesCheck && !hasLogicalOperand(expr)) {
        throw new IllegalStateException("types check but evaluate threw " + e.getMessage());
      }
      return;
    }

    var truthy = interpreter.interpret(expr);
    if (truthy != (value != null && !Boolean.FALSE.equals(value))) {
      throw new IllegalStateException("interpret returned " + truthy + " but evaluate returned " + value);
    }
  }

  /**
   * Whether a type-constrained operator takes an and/or directly, as in
   * (b or 0) >= n. The checker can't tell which side the or yields, so it
   * passes these, and interpreting them can still fail.
   */
  private static boolean hasLogicalOperand(Expr e) {
    if (e instanceof Expr.Binary b) {
      var constrained = b.operator.type != TokenType.EQUAL && b.operator.type != TokenType.NOT_EQUAL;
      return (constrained && (isLogical(b.left) || isLogical(b.right)))
          || hasLogicalOperand(b.left) || hasLogicalOperand(b.right);
    }
    if (e instanceof Expr.Infix i) {
      return isLogical(i.left) || isLogical(i.right) || hasLogicalOperand(i.left) || hasLogicalOperand(i.right);
    }
    if (e instanceof Expr.Logical l) {
      return hasLogicalOperand(l.left) || hasLogicalOperand(l.right);
    }
    if (e instanceof Expr.Unary u) {
      return hasLogicalOperand(u.right);
    }
    if (e instanceof Expr.Grouping g) {
      return hasLogicalOperand(g.expression);
    }
    return false;
  }

  private static boolean isLogical(Expr e) {
    while (e instanceof Expr.Grouping g) {
      e = g.expression;
    }
    return e instanceof Expr.Logical;
  }

  /**
   * Checks that an error message shows the line of the expression containing
   * the error at pos, a position in UTF-16 code units, with a marker under it.
   */
  private static void checkError(String expression, int pos, Errors.ErrorMessage e) {
    if (pos < 0 || pos > expression.length()) {
      throw new IllegalStateException("error position " + pos + " outside expression of length " + expression.length());
    }

    var msg = e.asErrorMessage(expression);

    if (pos < expression.length() && expression.charAt(pos) == '\n') {
      // An error on a newline shows the rest of the expression.
      return;
    }
    var lineStart = expression.lastIndexOf('\n', pos - 1) + 1;
    var lineEnd = expression.indexOf('\n', pos);
    if (lineEnd < 0) {
      lineEnd = expression.length();
    }

    // The preamble can quote a lexeme with newlines in it, so check the end
    // of the message. The marker is as long as the token, which is empty at
    // EOF.
    var wantSuffix = "\n" + expression.substring(lineStart, lineEnd) + "\n" + " ".repeat(pos - lineStart);
    if (!msg.replaceAll("\\^+$", "").endsWith(wantSuffix)) {
      throw new IllegalStateException("error at " + pos + " has message:\n" + msg + "\nwant it to end with:\n" + wantSuffix);
    }
  }
}
