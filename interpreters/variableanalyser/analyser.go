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

package variableanalyser

import "github.com/CircleCI-Public/expr/parsers"

type analyser struct{}

func New() analyser {
	return analyser{}
}

func (a analyser) GatherVariables(expr parsers.Expr) ([]string, error) {
	return a.check(expr)
}

func (a analyser) VisitLogicalExpr(expr parsers.Logical) ([]string, error) {
	l, err := a.check(expr.Left)
	if err != nil {
		return nil, err
	}

	r, err := a.check(expr.Right)
	if err != nil {
		return nil, err
	}

	return append(l, r...), nil
}

func (a analyser) VisitBinaryExpr(expr parsers.Binary) ([]string, error) {
	l, err := a.check(expr.Left)
	if err != nil {
		return nil, err
	}

	r, err := a.check(expr.Right)
	if err != nil {
		return nil, err
	}

	return append(l, r...), nil
}

func (a analyser) VisitInfixExpr(expr parsers.Infix) ([]string, error) {
	l, err := a.check(expr.Left)
	if err != nil {
		return nil, err
	}

	r, err := a.check(expr.Right)
	if err != nil {
		return nil, err
	}

	return append(l, r...), nil
}

func (a analyser) VisitUnaryExpr(expr parsers.Unary) ([]string, error) {
	return a.check(expr.Right)
}

func (a analyser) VisitLiteralExpr(_ parsers.Literal) ([]string, error) {
	return nil, nil
}

func (a analyser) VisitIdentifierExpr(expr parsers.Identifier) ([]string, error) {
	return []string{expr.Name.Lexeme}, nil
}

func (a analyser) VisitGroupingExpr(expr parsers.Grouping) ([]string, error) {
	return a.check(expr.Expression)
}

func (a analyser) check(expr parsers.Expr) ([]string, error) {
	v := parsers.Visitable[[]string]{Expression: expr}
	return v.Accept(a)
}
