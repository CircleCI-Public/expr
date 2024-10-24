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

import java.util.Map;

import static com.circleci.expr.TokenType.*;

/**
 * A Expr.Visitor implementation that interprets expressions.
 */
public class Interpreter implements Expr.Visitor<Object> {

  public static class Error extends RuntimeException {
    static final long serialVersionUID = 1;

    public static enum Type {
      EXPECTED_NUMERIC_OPERAND("Expected numeric value."),
      EXPECTED_STRING_OPERAND("Expected string value."),
      UNKNOWN_VARIABLE("Referred to a variable that is not set.");

      public final String message;
      private Type(String message) {
        this.message = message;
      }
    }

    public final Token token;
    public final Type type;

    public Error(Token token, Type type) {
      super(type.message);
      this.token = token;
      this.type = type;
    }
  }

  // A lookup table from known variable names to values
  private final Map<String, Object> environment;

  public Interpreter(Map<String, Object> environment) {
    this.environment = environment;
  }

  /**
   * Interpret the expression represented by the AST rooted at `expr` in the
   * Interpreter's configured `environment`.
   *
   * Returns the boolean value of `expr`.
   *
   * @throws Interpreter.Error if an error is encountered while interpreting `expr`.
   */
  public Boolean interpret(Expr expr) {
    return isTruthy(expr.accept(this));
  }

  @Override
  public Object visitLogicalExpr(Expr.Logical expr) {
    Object left = evaluate(expr.left);

    if (expr.operator.type == OR) {
      if (isTruthy(left)) return left;
    }
    else {
      if (!isTruthy(left)) return left;
    }

    return evaluate(expr.right);
  }

  @Override
  public Object visitBinaryExpr(Expr.Binary expr) {
    // evaluate left and right, apply the operator, return it
    Object left = evaluate(expr.left);
    Object right = evaluate(expr.right);

    // This implementation uses `null` for undefined variables. Undefined
    // variables "infect" binary expressions, if either operand is undefined
    // then the result of the operator is undefined, no matter what the
    // operator is.
    if (left == null || right == null) {
      return null;
    }

    switch (expr.operator.type) {
      case EQUAL:
        return isEqual(left, right);
      case NOT_EQUAL:
        return !isEqual(left, right);
      case STARTS_WITH:
        assertStringOperands(expr.operator, left, right);
        return ((String) left).startsWith((String) right);
      case GREATER:
        assertNumberOperands(expr.operator, left, right);
        return (long) left > (long) right;
      case GREATER_EQUAL:
        assertNumberOperands(expr.operator, left, right);
        return (long) left >= (long) right;
      case LESS:
        assertNumberOperands(expr.operator, left, right);
        return (long) left < (long) right;
      case LESS_EQUAL:
        assertNumberOperands(expr.operator, left, right);
        return (long) left <= (long) right;
    }
    return null;
  }

  @Override
  public Object visitUnaryExpr(Expr.Unary expr) {
    Object value = evaluate(expr.right);

    switch (expr.operator.type) {
      case NOT:
        return !isTruthy(value);
    }

    return null;
  }

  @Override
  public Object visitLiteralExpr(Expr.Literal expr) {
    return expr.value;
  }

  @Override
  public Object visitIdentifierExpr(Expr.Identifier expr) {
    if (!environment.containsKey(expr.name.lexeme)) {
      throw new Error(expr.name, Error.Type.UNKNOWN_VARIABLE);
    }

    return environment.get(expr.name.lexeme);
  }

  @Override
  public Object visitGroupingExpr(Expr.Grouping expr) {
    return evaluate(expr.expression);
  }

  private Object evaluate(Expr expr) {
    return expr.accept(this);
  }

  private boolean isTruthy(Object val) {
    if (Boolean.FALSE.equals(val) || val == null) return false;
    return true;
  }

  private boolean isEqual(Object a, Object b) {
    if (a == null && b == null) return true;
    if (a == null) return false;

    if (a instanceof Long && b instanceof Long) {
      return ((Long) a).longValue() == ((Long) b).longValue();
    }

    return a.equals(b);
  }

  private boolean assertStringOperands(Token operator, Object left, Object right) {
    if (left instanceof String && right instanceof String) return true;
    throw new Error(operator, Error.Type.EXPECTED_STRING_OPERAND);
  }

  private boolean assertNumberOperands(Token operator, Object left, Object right) {
    if (left instanceof Long && right instanceof Long) return true;
    throw new Error(operator, Error.Type.EXPECTED_NUMERIC_OPERAND);
  }
}
