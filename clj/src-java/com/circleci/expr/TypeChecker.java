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

import java.util.Locale;
import java.util.Map;
import com.google.re2j.Pattern;

import static com.circleci.expr.Errors.ErrorMessage;

/**
 * A Expr.Visitor implementation that checks each operator's operands against
 * the variable types given to the constructor, without knowing the variables'
 * values.
 */
public class TypeChecker implements Expr.Visitor<TypeChecker.ValueType> {

  /**
   * `token` is the operator, or the identifier for UNKNOWN_VARIABLE. `operand`
   * and `operandType` are null for UNKNOWN_VARIABLE.
   */
  public static class Error extends RuntimeException implements ErrorMessage {
    static final long serialVersionUID = 1;

    public static enum Type {
      EXPECTED_NUMERIC_OPERAND("Expected numeric value."),
      EXPECTED_STRING_OPERAND("Expected string value."),
      EXPECTED_PATTERN_OPERAND("Expected regular expression value."),
      UNKNOWN_VARIABLE("Referred to a variable that has no known type.");

      public final String message;
      private Type(String message) {
        this.message = message;
      }
    }

    public final Token token;
    public final Type type;
    public final String operand;
    public final String operandType;

    private Error(Token token, Type type, String operand, String operandType) {
      super(type.message);
      this.token = token;
      this.type = type;
      this.operand = operand;
      this.operandType = operandType;
    }

    @Override
    public String asErrorMessage(String expression) {
      var message = type == Type.UNKNOWN_VARIABLE
          ? String.format("Referred to a variable \"%s\" that has no known type:", token.lexeme)
          : String.format("\"%s\" needs %s, but %s is a %s:", token.lexeme, needs(), operand, operandType);
      return Errors.errorMessage(message, expression, token.charPos, token.lexeme.length());
    }

    private String needs() {
      if (type == Type.EXPECTED_NUMERIC_OPERAND) return "numbers";
      if (type == Type.EXPECTED_PATTERN_OPERAND) return "a pattern on the right";
      if (Builtin.forLexeme(token.lexeme) == Builtin.MATCHES) return "a string on the left";
      return "strings";
    }
  }

  enum ValueType { STRING, BOOLEAN, NUMBER, PATTERN, UNKNOWN }

  private final Map<String, ?> types;

  public TypeChecker(Map<String, ?> types) {
    this.types = types;
  }

  /**
   * Check the operands of every operator in the AST rooted at `expr`, without
   * short-circuiting `and` or `or`.
   *
   * @throws TypeChecker.Error for the first operand of the wrong type, or the
   * first variable with no known type, in evaluation order.
   */
  public void checkTypes(Expr expr) {
    check(expr);
  }

  @Override
  public ValueType visitLogicalExpr(Expr.Logical expr) {
    var left = check(expr.left);
    var right = check(expr.right);
    return left == right ? left : ValueType.UNKNOWN;
  }

  @Override
  public ValueType visitBinaryExpr(Expr.Binary expr) {
    var left = check(expr.left);
    var right = check(expr.right);

    switch (expr.operator.type) {
      case GREATER, GREATER_EQUAL, LESS, LESS_EQUAL -> {
        expect(ValueType.NUMBER, left, expr.left, "the left operand", expr.operator, Error.Type.EXPECTED_NUMERIC_OPERAND);
        expect(ValueType.NUMBER, right, expr.right, "the right operand", expr.operator, Error.Type.EXPECTED_NUMERIC_OPERAND);
      }
      default -> {}
    }
    return ValueType.BOOLEAN;
  }

  @Override
  public ValueType visitInfixExpr(Expr.Infix expr) {
    var left = check(expr.left);
    var right = check(expr.right);

    return switch (expr.builtin) {
      case CONTAINS, STARTS_WITH -> {
        expect(ValueType.STRING, left, expr.left, "the left operand", expr.operator, Error.Type.EXPECTED_STRING_OPERAND);
        expect(ValueType.STRING, right, expr.right, "the right operand", expr.operator, Error.Type.EXPECTED_STRING_OPERAND);
        yield ValueType.BOOLEAN;
      }
      case MATCHES -> {
        expect(ValueType.PATTERN, right, expr.right, "the right operand", expr.operator, Error.Type.EXPECTED_PATTERN_OPERAND);
        expect(ValueType.STRING, left, expr.left, "the left operand", expr.operator, Error.Type.EXPECTED_STRING_OPERAND);
        yield ValueType.BOOLEAN;
      }
    };
  }

  @Override
  public ValueType visitUnaryExpr(Expr.Unary expr) {
    check(expr.right);
    return ValueType.BOOLEAN;
  }

  @Override
  public ValueType visitLiteralExpr(Expr.Literal expr) {
    return switch (expr.value) {
      case Long l -> ValueType.NUMBER;
      case String s -> ValueType.STRING;
      case Boolean b -> ValueType.BOOLEAN;
      case Pattern p -> ValueType.PATTERN;
      default -> ValueType.UNKNOWN;
    };
  }

  @Override
  public ValueType visitIdentifierExpr(Expr.Identifier expr) {
    if (!(types.get(expr.name.lexeme) instanceof String type)) {
      throw unknownVariable(expr.name);
    }

    return switch (type) {
      case "string" -> ValueType.STRING;
      case "boolean" -> ValueType.BOOLEAN;
      case "number" -> ValueType.NUMBER;
      case "pattern" -> ValueType.PATTERN;
      default -> throw unknownVariable(expr.name);
    };
  }

  @Override
  public ValueType visitGroupingExpr(Expr.Grouping expr) {
    return check(expr.expression);
  }

  private ValueType check(Expr expr) {
    return expr.accept(this);
  }

  private static Error unknownVariable(Token name) {
    return new Error(name, Error.Type.UNKNOWN_VARIABLE, null, null);
  }

  private static void expect(ValueType expected, ValueType actual, Expr operand, String position, Token operator, Error.Type error) {
    if (actual != expected && actual != ValueType.UNKNOWN) {
      var name = operand instanceof Expr.Identifier identifier ? identifier.name.lexeme : position;
      throw new Error(operator, error, name, actual.name().toLowerCase(Locale.ROOT));
    }
  }
}
