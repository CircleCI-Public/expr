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

/* Expressions:
 * Logical
 * Binary
 * Unary
 * Literal
 * Identifier
 * Grouping
 */

/**
 * Represents expressions that together form an abstract syntax tree.
 *
 * Exprs are simple data types with public members. They provide an `accept`
 * method which cooperates with the `Visitor` interface to facilitate walking
 * the AST.
 */
public abstract class Expr {
  abstract <T> T accept(Visitor<T> visitor);

  public interface Visitor<T> {
    T visitLogicalExpr(Logical expr);
    T visitBinaryExpr(Binary expr);
    T visitInfixExpr(Infix expr);
    T visitUnaryExpr(Unary expr);
    T visitLiteralExpr(Literal expr);
    T visitIdentifierExpr(Identifier expr);
    T visitGroupingExpr(Grouping expr);
  }

  /**
   * A logical `and` / `or` node in the AST.
   */
  public static class Logical extends Expr {
    public final Expr left;
    public final Token operator;
    public final Expr right;

    public Logical(Expr left, Token operator, Expr right) {
      this.left = left;
      this.operator = operator;
      this.right = right;
    }

    @Override
    public <T> T accept(Visitor<T> visitor) {
      return visitor.visitLogicalExpr(this);
    }
  }

  /**
   * A binary operator node in the AST.
   */
  public static class Binary extends Expr {
    public final Expr left;
    public final Token operator;
    public final Expr right;

    public Binary(Expr left, Token operator, Expr right) {
      this.left = left;
      this.operator = operator;
      this.right = right;
    }

    @Override
    public <T> T accept(Visitor<T> visitor) {
      return visitor.visitBinaryExpr(this);
    }
  }

  /**
   * An infix builtin function node in the AST.
   */
  public static class Infix extends Expr {
    public final Expr left;
    public final Token operator;
    public final Builtin builtin;
    public final Expr right;

    public Infix(Expr left, Token operator, Builtin builtin, Expr right) {
      this.left = left;
      this.operator = operator;
      this.builtin = builtin;
      this.right = right;
    }

    @Override
    public <T> T accept(Visitor<T> visitor) {
      return visitor.visitInfixExpr(this);
    }
  }

  /**
   * A unary operator node in the AST.
   */
  public static class Unary extends Expr {
    public final Token operator;
    public final Expr right;

    public Unary(Token operator, Expr right) {
      this.operator = operator;
      this.right = right;
    }

    @Override
    public <T> T accept(Visitor<T> visitor) {
      return visitor.visitUnaryExpr(this);
    }
  }

  /**
   * A literal value node in the AST.
   *
   * Literal nodes are leaf nodes.
   */
  public static class Literal extends Expr {
    public final Object value;

    public Literal(Object value) {
      this.value = value;
    }

    @Override
    public <T> T accept(Visitor<T> visitor) {
      return visitor.visitLiteralExpr(this);
    }
  }

  /**
   * An identifier node in the AST.
   *
   * Identifier nodes are leaf nodes.
   */
  public static class Identifier extends Expr {
    public final Token name;

    public Identifier(Token name) {
      this.name = name;
    }

    @Override
    public <T> T accept(Visitor<T> visitor) {
      return visitor.visitIdentifierExpr(this);
    }
  }

  /**
   * A grouping node in the AST.
   *
   * Grouping wraps any other expression node.
   */
  public static class Grouping extends Expr {
    public final Expr expression;

    public Grouping(Expr expression) {
      this.expression = expression;
    }

    @Override
    public <T> T accept(Visitor<T> visitor) {
      return visitor.visitGroupingExpr(this);
    }
  }
}
