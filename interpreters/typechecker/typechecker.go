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

package typechecker

import (
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/CircleCI-Public/expr/builtins"
	"github.com/CircleCI-Public/expr/errors"
	"github.com/CircleCI-Public/expr/parsers"
	"github.com/CircleCI-Public/expr/tokens"
)

// The variable types a checker accepts.
const (
	String  = "string"
	Boolean = "boolean"
	Number  = "number"
	Pattern = "pattern"
)

type errorType string

const (
	EXPECTED_NUMERIC_OPERAND errorType = "Expected numeric value."
	EXPECTED_STRING_OPERAND  errorType = "Expected string value."
	EXPECTED_PATTERN_OPERAND errorType = "Expected regular expression value."
	UNKNOWN_VARIABLE         errorType = "Referred to a variable with no known type."
)

func (et errorType) Symbol() string {
	switch et {
	case EXPECTED_NUMERIC_OPERAND:
		return "EXPECTED_NUMERIC_OPERAND"
	case EXPECTED_STRING_OPERAND:
		return "EXPECTED_STRING_OPERAND"
	case EXPECTED_PATTERN_OPERAND:
		return "EXPECTED_PATTERN_OPERAND"
	case UNKNOWN_VARIABLE:
		return "UNKNOWN_VARIABLE"
	}

	panic("Encountered unknown TypeChecker errorType value")
}

// Error is the first type error in an expression. Token is the operator, or
// the identifier for UNKNOWN_VARIABLE. Operand is how the message names the
// wrong operand: its variable name when it is a bare identifier, otherwise its
// side. Operand and OperandType are empty for UNKNOWN_VARIABLE.
type Error struct {
	Type        errorType
	Token       tokens.Token
	Operand     string
	OperandType string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Token.String())
}

func (e Error) AsErrorMessage(expression string) string {
	message := fmt.Sprintf("Referred to a variable \"%s\" that has no known type:", e.Token.Lexeme)
	if e.Type != UNKNOWN_VARIABLE {
		message = fmt.Sprintf("\"%s\" needs %s, but %s is a %s:", e.Token.Lexeme, e.need(), e.Operand, e.OperandType)
	}

	return errors.ErrorMessage(message, expression, e.Token.CharPos, utf8.RuneCountInString(e.Token.Lexeme))
}

func (e Error) need() string {
	switch {
	case e.Type == EXPECTED_NUMERIC_OPERAND:
		return "numbers"
	case e.Type == EXPECTED_PATTERN_OPERAND:
		return "a pattern on the right"
	case builtins.ForLexeme(e.Token.Lexeme) == builtins.MATCHES:
		return "a string on the left"
	}
	return "strings"
}

type checker struct {
	types map[string]string
}

// New returns a checker for expressions over variables of the given types,
// keyed by variable name.
func New(types map[string]string) checker {
	return checker{types: types}
}

// CheckTypes returns the first Error in expr, in evaluation order, or nil when
// every variable has a known type and no operand has a known wrong type.
func (c checker) CheckTypes(expr parsers.Expr) error {
	_, err := c.check(expr)
	return err
}

func (c checker) VisitLogicalExpr(expr parsers.Logical) (string, error) {
	l, r, err := c.operands(expr.Left, expr.Right)
	if err != nil || l != r {
		return "", err
	}
	return l, nil
}

func (c checker) VisitBinaryExpr(expr parsers.Binary) (string, error) {
	l, r, err := c.operands(expr.Left, expr.Right)
	if err != nil {
		return "", err
	}

	switch expr.Operator.Type {
	case tokens.EQUAL, tokens.NOT_EQUAL:
		return Boolean, nil
	}
	if err := numericOperands.require(expr.Operator, expr.Left, leftOperand, l); err != nil {
		return "", err
	}
	return Boolean, numericOperands.require(expr.Operator, expr.Right, rightOperand, r)
}

func (c checker) VisitInfixExpr(expr parsers.Infix) (string, error) {
	l, r, err := c.operands(expr.Left, expr.Right)
	if err != nil {
		return "", err
	}

	switch expr.Builtin {
	case builtins.CONTAINS, builtins.STARTS_WITH:
		if err := stringOperands.require(expr.Operator, expr.Left, leftOperand, l); err != nil {
			return "", err
		}
		return Boolean, stringOperands.require(expr.Operator, expr.Right, rightOperand, r)
	case builtins.MATCHES:
		// Pattern side first: reordering changes the error for `5 matches 6`.
		if err := patternOperands.require(expr.Operator, expr.Right, rightOperand, r); err != nil {
			return "", err
		}
		return Boolean, stringOperands.require(expr.Operator, expr.Left, leftOperand, l)
	}
	return "", nil
}

func (c checker) VisitUnaryExpr(expr parsers.Unary) (string, error) {
	_, err := c.check(expr.Right)
	return Boolean, err
}

func (c checker) VisitLiteralExpr(expr parsers.Literal) (string, error) {
	switch expr.Value.(type) {
	case bool:
		return Boolean, nil
	case string:
		return String, nil
	case int64:
		return Number, nil
	case *regexp.Regexp:
		return Pattern, nil
	}
	return "", nil
}

func (c checker) VisitIdentifierExpr(expr parsers.Identifier) (string, error) {
	switch t := c.types[expr.Name.Lexeme]; t {
	case String, Boolean, Number, Pattern:
		return t, nil
	}
	return "", Error{Type: UNKNOWN_VARIABLE, Token: expr.Name}
}

func (c checker) VisitGroupingExpr(expr parsers.Grouping) (string, error) {
	return c.check(expr.Expression)
}

func (c checker) check(expr parsers.Expr) (string, error) {
	v := parsers.Visitable[string]{Expression: expr}
	return v.Accept(c)
}

func (c checker) operands(left, right parsers.Expr) (string, string, error) {
	l, err := c.check(left)
	if err != nil {
		return "", "", err
	}
	r, err := c.check(right)
	return l, r, err
}

type requirement struct {
	errType errorType
	want    string
}

var (
	numericOperands = requirement{errType: EXPECTED_NUMERIC_OPERAND, want: Number}
	stringOperands  = requirement{errType: EXPECTED_STRING_OPERAND, want: String}
	patternOperands = requirement{errType: EXPECTED_PATTERN_OPERAND, want: Pattern}
)

const (
	leftOperand  = "the left operand"
	rightOperand = "the right operand"
)

func (rq requirement) require(op tokens.Token, operand parsers.Expr, side, got string) error {
	if got == "" || got == rq.want {
		return nil
	}
	name := side
	if id, ok := operand.(parsers.Identifier); ok {
		name = id.Name.Lexeme
	}
	return Error{Type: rq.errType, Token: op, Operand: name, OperandType: got}
}
