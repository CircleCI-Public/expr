package variableanalyser

import "github.com/circleci/expr/go/parsers"

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
