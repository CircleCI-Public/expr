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
import java.util.List;

/**
 * A Expr.Visitor implementation that gathers variables referred to in
 * expressions.
 */
public class VariableAnalyser implements Expr.Visitor<List<Token>> {
  /**
   * Gather the variables used by the AST rooted at `expr`.
   *
   * Returns a `List` of `Token` objects that represent uses of a variable in
   * the expressions.
   */
  public List<Token> gatherVariables(Expr expr) {
    return check(expr);
  }

  @Override
  public List<Token> visitLogicalExpr(Expr.Logical expr) {
    var l = check(expr.left);
    l.addAll(check(expr.right));
    return l;
  }

  @Override
  public List<Token> visitBinaryExpr(Expr.Binary expr) {
    var l = check(expr.left);
    l.addAll(check(expr.right));
    return l;
  }

  @Override
  public List<Token> visitUnaryExpr(Expr.Unary expr) {
    return check(expr.right);
  }

  @Override
  public List<Token> visitLiteralExpr(Expr.Literal expr) {
    return new ArrayList<Token>(0);
  }

  @Override
  public List<Token> visitIdentifierExpr(Expr.Identifier expr) {
    var l = new ArrayList<Token>(1);
    l.add(expr.name);
    return l;
  }

  @Override
  public List<Token> visitGroupingExpr(Expr.Grouping expr) {
    return check(expr.expression);
  }

  private List<Token> check(Expr expr) {
    return expr.accept(this);
  }
}
