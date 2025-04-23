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

package parser

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/circleci/expr/go/scanner"
)

type sExpVisitor struct{}

func (v sExpVisitor) VisitLogicalExpr(expr Logical) (any, error) {
	l, err := expr.Left.Accept(v)
	if err != nil {
		return "", err
	}

	r, err := expr.Right.Accept(v)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("(%s %s %s)", expr.Operator.Lexeme, l, r), nil
}

func (v sExpVisitor) VisitBinaryExpr(expr Binary) (any, error) {
	l, err := expr.Left.Accept(v)
	if err != nil {
		return "", err
	}

	r, err := expr.Right.Accept(v)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("(%s %s %s)", expr.Operator.Lexeme, l, r), nil
}

func (v sExpVisitor) VisitUnaryExpr(expr Unary) (any, error) {
	r, err := expr.Right.Accept(v)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("(%s %s)", expr.Operator.Lexeme, r), nil
}

func (v sExpVisitor) VisitLiteralExpr(expr Literal) (any, error) {
	return fmt.Sprintf("(literal %v)", expr.Value), nil
}

func (v sExpVisitor) VisitIdentifierExpr(expr Identifier) (any, error) {
	return fmt.Sprintf("(identifier %s)", expr.Name.Lexeme), nil
}

func (v sExpVisitor) VisitGroupingExpr(expr Grouping) (any, error) {
	g, err := expr.Expression.Accept(v)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("(grouping %s)", g), nil
}

func normalise(sExp []string) string {
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(strings.Join(sExp, " "), " ")
}

func parse(expression string) (string, error) {
	s := scanner.New(expression)

	tokens, err := s.Scan()
	if err != nil {
		return "", err
	}

	p := New(tokens)
	expr, err := p.Parse()

	if err != nil {
		return "", err
	}

	v, err := expr.Accept(sExpVisitor{})
	if err != nil {
		return "", err
	}

	if v, ok := v.(string); ok {
		return v, nil
	}

	return "", fmt.Errorf("Didn't get a string")
}

func TestLiterals(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		expression string
		expected   string
	}{
		{"foo", "(identifier foo)"},
		{"14", "(literal 14)"},
		{"true", "(literal true)"},
		{"false", "(literal false)"},
		{"\"a string\"", "(literal a string)"},
		{"\"a \\\"str\\\\ing\"", "(literal a \"str\\ing)"},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			t.Parallel()

			sExp, err := parse(tt.expression)
			assert.NilError(t, err)

			assert.DeepEqual(t, tt.expected, sExp)
		})
	}
}

func TestLogicalExpressions(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		expression string
		expected   string
	}{
		{"true or false", "(or (literal true) (literal false))"},
		{"false and false", "(and (literal false) (literal false))"},
		{"14 and true or \"hello\"",
			normalise([]string{
				"(or (and (literal 14)",
				"         (literal true))",
				"    (literal hello))"})},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			t.Parallel()

			sExp, err := parse(tt.expression)
			assert.NilError(t, err)

			assert.DeepEqual(t, tt.expected, sExp)
		})
	}
}

func TestBinaryExpressions(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		expression string
		expected   string
	}{
		{"\"str\" == 14", "(== (literal str) (literal 14))"},
		{"3 != 14", "(!= (literal 3) (literal 14))"},
		{"4 > 3 < 2", "(< (> (literal 4) (literal 3)) (literal 2))"},
		{"18 >= 10 <= 16", "(<= (>= (literal 18) (literal 10)) (literal 16))"},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			t.Parallel()

			sExp, err := parse(tt.expression)
			assert.NilError(t, err)

			assert.DeepEqual(t, tt.expected, sExp)
		})
	}
}

func TestExpressions(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		expression string
		expected   string
	}{
		{"18 >= (10 <= 16)",
			normalise([]string{
				"(>= (literal 18)",
				"    (grouping (<= (literal 10)",
				"                  (literal 16))))"})},
		{"foo.bar == \"main\" and 1 <= 4 or 5 < baz",
			normalise([]string{
				"(or (and (== (identifier foo.bar)",
				"             (literal main))",
				"         (<= (literal 1)",
				"             (literal 4)))",
				"    (< (literal 5)",
				"       (identifier baz)))"})},
		{"(5 < 4 or not foo) and \"s\" != 42",
			normalise([]string{
				"(and (grouping",
				"       (or (< (literal 5)",
				"              (literal 4))",
				"           (not (identifier foo))))",
				"     (!= (literal s)",
				"         (literal 42)))"})},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			t.Parallel()

			sExp, err := parse(tt.expression)
			assert.NilError(t, err)

			assert.DeepEqual(t, tt.expected, sExp)
		})
	}
}

func TestParseErrors(t *testing.T) {
	t.Parallel()

	t.Run("requires expressions", func(t *testing.T) {
		t.Parallel()

		_, err := parse("and")
		assert.Error(t, err, "error parsing: EXPECTED_EXPRESSION at <token AND and and>[0]")
	})

	t.Run("additional input is an error", func(t *testing.T) {
		t.Parallel()

		_, err := parse("foo and bar baz")
		assert.Error(t, err, "error parsing: UNEXPECTED_ADDITIONAL_INPUT at <token IDENTIFIER baz baz>[12]")
	})

	t.Run("grouping parens must be balanced", func(t *testing.T) {
		t.Parallel()

		_, err := parse("(1 > 2 3")
		assert.Error(t, err, "error parsing: EXPECTED_RIGHT_PAREN at <token NUMBER 3 3>[7]")

		_, err = parse("1 > 2) 3")
		assert.Error(t, err, "error parsing: UNEXPECTED_ADDITIONAL_INPUT at <token RIGHT_PAREN ) <nil>>[5]")

		_, err = parse("((foo and bar)")
		assert.ErrorContains(t, err, "error parsing: EXPECTED_RIGHT_PAREN at")

		_, err = parse("0 <= (1 > (2 < 3) >= 5")
		assert.ErrorContains(t, err, "error parsing: EXPECTED_RIGHT_PAREN at")
	})

	t.Run("expressions must be well-formed", func(t *testing.T) {
		var tests = []struct {
			expression string
		}{
			{""},
			{"and"},
			{"foo or"},
			{"and foo"},
			{"!"},
			{"5 <"},
			{">= 4"},
			{"("},
			{"foo and ("},
		}

		for _, tt := range tests {
			t.Run(tt.expression, func(t *testing.T) {
				t.Parallel()

				_, err := parse(tt.expression)
				assert.ErrorContains(t, err, "error parsing: EXPECTED_EXPRESSION")
			})
		}
	})
}

func TestPrettyParseErrors(t *testing.T) {
	t.Parallel()

	t.Run("Unexpected additional input", func(t *testing.T) {
		t.Parallel()
		expression := "5 > 4 foo"

		_, err := parse(expression)
		assert.ErrorContains(t, err, "error parsing:")

		expected := strings.Join(
			[]string{
				"Unexpected additional input, found \"foo\", expected EOF:",
				"5 > 4 foo",
				"      ^^^"},
			"\n")

		assert.Equal(t, expected, err.(Error).AsErrorMessage(expression))
	})

	t.Run("Expected an expression", func(t *testing.T) {
		t.Parallel()
		expression := "foo and ) bar"

		_, err := parse(expression)
		assert.ErrorContains(t, err, "error parsing:")

		expected := strings.Join(
			[]string{
				"Expected expression, found \")\":",
				"foo and ) bar",
				"        ^"},
			"\n")

		assert.Equal(t, expected, err.(Error).AsErrorMessage(expression))
	})

	t.Run("Expected a right parenthesis", func(t *testing.T) {
		t.Parallel()
		expression := "foo and (bar > 3"

		_, err := parse(expression)
		assert.ErrorContains(t, err, "error parsing:")

		expected := strings.Join(
			[]string{
				"Expected ')' after expression:",
				"foo and (bar > 3",
				"                ^"},
			"\n")

		assert.Equal(t, expected, err.(Error).AsErrorMessage(expression))
	})
}
