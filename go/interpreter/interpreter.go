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

package interpreter

import (
	"fmt"

	"github.com/circleci/expr/go/errors"
	"github.com/circleci/expr/go/parser"
	"github.com/circleci/expr/go/token"
)

type errorType string

const (
	EXPECTED_NUMERIC_OPERAND errorType = "Expected numeric value."
	EXPECTED_STRING_OPERAND  errorType = "Expected string value."
	UNKNOWN_VARIABLE         errorType = "Referred to a variable that is not set."
)

func (et errorType) Symbol() string {
	switch et {
	case EXPECTED_NUMERIC_OPERAND:
		return "EXPECTED_NUMERIC_OPERAND"
	case EXPECTED_STRING_OPERAND:
		return "EXPECTED_STRING_OPERAND"
	case UNKNOWN_VARIABLE:
		return "UNKNOWN_VARIABLE"
	}

	panic("Encountered unknown Interpreter errorType value")
}

type Error struct {
	Type  errorType
	Token token.Token
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Token.String())
}

// Return a multiline error message describing this error.
//
// Returns a string of the form:
//
// explanation
// line of expression containing the error
// marker line pointing at the error token
//
// E.g.
// "Unexpected character '&':\n" +
// "2 > 5 && false\n" +
// "      ^"
func (e Error) AsErrorMessage(expression string) string {
	errorString := e.Token.Lexeme

	switch e.Type {
	case EXPECTED_NUMERIC_OPERAND:
		return errors.ErrorMessage(fmt.Sprintf("Expected numeric operands to \"%s\" operator:", errorString),
			expression,
			e.Token.CharPos,
			len(errorString))

	case EXPECTED_STRING_OPERAND:
		return errors.ErrorMessage(fmt.Sprintf("Expected string operands to \"%s\" operator:", errorString),
			expression,
			e.Token.CharPos,
			len(errorString))

	case UNKNOWN_VARIABLE:
		return errors.ErrorMessage(fmt.Sprintf("Referred to a variable \"%s\" that does not exist:", errorString),
			expression,
			e.Token.CharPos,
			len(errorString))

	default:
		return fmt.Sprintf("Unknown error interpreting expression: '%s'", expression)
	}
}

type Interpreter struct {
	env map[string]Val
}

func New(env map[string]Val) Interpreter {
	return Interpreter{env: env}
}

// Interpret the expression represented by the AST rooted at expr in the
// Interpreter's configured environment.
//
// Returns the boolean value of expr.
//
// Returns Error if an error is encountered while interpreting expr.
func (i Interpreter) Interpret(expr parser.Expr) (bool, error) {
	res, err := i.Evaluate(expr)
	if err != nil {
		return false, err
	}

	return res.IsTruthy(), nil
}

// Evaluate the expression represented by the AST rooted at expr in the
// Interpreter's configured environment.
//
// Returns the value of expr, which may be any scalar type.
//
// Returns an Error if an error is encountered while interpreting expr.
func (i Interpreter) Evaluate(expr parser.Expr) (Val, error) {
	v := parser.Visitable[Val]{Expression: expr}
	res, err := v.Accept(i)

	if err != nil {
		return BoxBool(false), err
	}
	return res, nil
}

func (i Interpreter) VisitLogicalExpr(expr parser.Logical) (Val, error) {
	left, err := i.Evaluate(expr.Left)
	if err != nil {
		return UndefinedVal(), err
	}

	if expr.Operator.Type == token.OR {
		if left.IsTruthy() {
			return left, nil
		}
	} else {
		if !left.IsTruthy() {
			return left, nil
		}
	}

	r, err := i.Evaluate(expr.Right)
	if err != nil {
		return UndefinedVal(), err
	}
	return r, nil
}

func (i Interpreter) VisitBinaryExpr(expr parser.Binary) (Val, error) {
	// Evaluate left and right, apply the operator, return it
	left, err := i.Evaluate(expr.Left)
	if err != nil {
		return UndefinedVal(), err
	}
	right, err := i.Evaluate(expr.Right)
	if err != nil {
		return UndefinedVal(), err
	}

	// This implementation uses `NewUndefined()` for undefined variables.
	// Undefined variables "infect" binary expressions, if either operand is
	// undefined then the result of the operator is undefined, no matter what the
	// operator is.
	if left.Undefined() || right.Undefined() {
		return UndefinedVal(), nil
	}

	switch expr.Operator.Type {
	case token.EQUAL:
		return BoxBool(left.Equal(right)), nil
	case token.NOT_EQUAL:
		return BoxBool(!left.Equal(right)), nil
	case token.STARTS_WITH:
		res, err := left.StartsWith(right)
		if err != nil {
			return UndefinedVal(), Error{
				Type:  EXPECTED_STRING_OPERAND,
				Token: expr.Operator,
			}
		}
		return BoxBool(res), nil
	case token.GREATER:
		res, err := left.Greater(right)
		if err != nil {
			return UndefinedVal(), Error{
				Type:  EXPECTED_NUMERIC_OPERAND,
				Token: expr.Operator,
			}
		}
		return BoxBool(res), nil
	case token.GREATER_EQUAL:
		res, err := left.GreaterEqual(right)
		if err != nil {
			return UndefinedVal(), Error{
				Type:  EXPECTED_NUMERIC_OPERAND,
				Token: expr.Operator,
			}
		}
		return BoxBool(res), nil
	case token.LESS:
		res, err := left.Less(right)
		if err != nil {
			return UndefinedVal(), Error{
				Type:  EXPECTED_NUMERIC_OPERAND,
				Token: expr.Operator,
			}
		}
		return BoxBool(res), nil
	case token.LESS_EQUAL:
		res, err := left.LessEqual(right)
		if err != nil {
			return UndefinedVal(), Error{
				Type:  EXPECTED_NUMERIC_OPERAND,
				Token: expr.Operator,
			}
		}
		return BoxBool(res), nil
	default:
		return UndefinedVal(), nil
	}
}

func (i Interpreter) VisitUnaryExpr(expr parser.Unary) (Val, error) {
	v, err := i.Evaluate(expr.Right)
	if err != nil {
		return UndefinedVal(), err
	}
	if expr.Operator.Type == token.NOT {
		return BoxBool(!v.IsTruthy()), nil
	}
	return UndefinedVal(), nil
}

func (i Interpreter) VisitLiteralExpr(expr parser.Literal) (Val, error) {
	return BoxVal(expr.Value)
}

func (i Interpreter) VisitIdentifierExpr(expr parser.Identifier) (Val, error) {
	v, ok := i.env[expr.Name.Lexeme]
	if !ok {
		return UndefinedVal(), Error{
			Type:  UNKNOWN_VARIABLE,
			Token: expr.Name,
		}
	}

	// The only numeric type supported by expr is integers. The Val constructor
	// ensures that only integer types can be used.
	return v, nil
}

func (i Interpreter) VisitGroupingExpr(expr parser.Grouping) (Val, error) {
	v, err := i.Evaluate(expr.Expression)
	if err != nil {
		return UndefinedVal(), err
	}
	return v, nil
}
