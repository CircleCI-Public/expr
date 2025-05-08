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
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/circleci/expr/go/parser"
	"github.com/circleci/expr/go/scanner"
)

func interpret(expression string, env map[string]any) (bool, error) {
	envVals := make(map[string]Val, len(env))
	for k, v := range env {
		b, err := BoxVal(v)
		if err != nil {
			return false, err
		}
		envVals[k] = b
	}

	s := scanner.New(expression)

	tokens, err := s.Scan()
	if err != nil {
		return false, err
	}

	p := parser.New(tokens)
	expr, err := p.Parse()

	if err != nil {
		return false, err
	}

	i := New(envVals)
	result, err := i.Interpret(expr)

	if err != nil {
		return false, err
	}

	return result, nil
}

func TestLiterals(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		expression string
		expected   bool
	}{
		// All literals are true
		{"0", true},
		{"1", true},
		{"\"string\"", true},
		{"ident", true},
		{"true", true},
		// Except for 'false'
		{"false", false},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			t.Parallel()

			env := map[string]any{"ident": 5}

			result, err := interpret(tt.expression, env)
			assert.NilError(t, err)

			assert.DeepEqual(t, tt.expected, result)
		})
	}
}

func TestLogicalExpressions(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		expression  string
		environment map[string]any
		expected    bool
	}{
		// ===
		// and
		// ===
		{"true and true", map[string]any{}, true},
		{"true and false", map[string]any{}, false},
		{"false and true", map[string]any{}, false},
		{"false and false", map[string]any{}, false},

		// 'and' short-circuits, shown here by not needing to reference the
		// 'foo' variable that doesn't exist in the environment
		{"false and foo", map[string]any{}, false},

		// Alias is accepted
		{"true AND false", map[string]any{}, false},

		// Environment lookups work
		{"true and foo", map[string]any{"foo": true}, true},
		{"foo and false", map[string]any{"foo": true}, false},
		{"true and foo", map[string]any{"foo": false}, false},

		// ==
		// or
		// ==
		{"true or true", map[string]any{}, true},
		{"true or false", map[string]any{}, true},
		{"false or true", map[string]any{}, true},
		{"false or false", map[string]any{}, false},

		// 'or' short-circuits, shown here by not needing to reference the 'foo'
		// variable that doesn't exist in the environment
		{"true or foo", map[string]any{}, true},

		// Alias is accepted
		{"false OR true", map[string]any{}, true},

		// Environment lookups work
		{"false or foo", map[string]any{"foo": true}, true},
		{"foo or foo", map[string]any{"foo": false}, false},

		// ===
		// not
		// ===
		{"!true", map[string]any{}, false},
		{"!false", map[string]any{}, true},
		// not is an alias for !
		{"not true", map[string]any{}, false},
		{"not false", map[string]any{}, true},

		// Environment works
		{"not foo", map[string]any{"foo": true}, false},
		{"!foo", map[string]any{"foo": true}, false},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			t.Parallel()

			result, err := interpret(tt.expression, tt.environment)
			assert.NilError(t, err)

			assert.DeepEqual(t, tt.expected, result)
		})
	}
}

func TestBinaryExpressions(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		expression  string
		environment map[string]any
		expected    bool
	}{
		// ===
		// and
		// ===
		{"true and true", map[string]any{}, true},
		{"true and false", map[string]any{}, false},
		{"false and true", map[string]any{}, false},
		{"false and false", map[string]any{}, false},

		// 'and' short-circuits, shown here by not needing to reference the
		// 'foo' variable that doesn't exist in the environment
		{"false and foo", map[string]any{}, false},

		// Alias is accepted
		{"true AND false", map[string]any{}, false},

		// Environment lookups work
		{"true and foo", map[string]any{"foo": true}, true},
		{"foo and false", map[string]any{"foo": true}, false},
		{"true and foo", map[string]any{"foo": false}, false},

		// ==
		// or
		// ==
		{"true or true", map[string]any{}, true},
		{"true or false", map[string]any{}, true},
		{"false or true", map[string]any{}, true},
		{"false or false", map[string]any{}, false},

		// 'or' short-circuits, shown here by not needing to reference the 'foo'
		// variable that doesn't exist in the environment
		{"true or foo", map[string]any{}, true},

		// Alias is accepted
		{"false OR true", map[string]any{}, true},

		// Environment lookups work
		{"false or foo", map[string]any{"foo": true}, true},
		{"foo or foo", map[string]any{"foo": false}, false},

		// ===
		// not
		// ===
		{"!true", map[string]any{}, false},
		{"!false", map[string]any{}, true},
		// not is an alias for !
		{"not true", map[string]any{}, false},
		{"not false", map[string]any{}, true},

		// Environment works
		{"not foo", map[string]any{"foo": true}, false},
		{"!foo", map[string]any{"foo": true}, false},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			t.Parallel()

			result, err := interpret(tt.expression, tt.environment)
			assert.NilError(t, err)

			assert.DeepEqual(t, tt.expected, result)
		})
	}
}

func TestNumericExpressionsRequireNumericOperands(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		expression  string
		environment map[string]any
	}{
		{"5 > true", map[string]any{}},
		{"false > 3", map[string]any{}},
		{"\"hi\" > \"hi\"", map[string]any{}},
		{"false > false", map[string]any{}},
		{"foo > 10", map[string]any{"foo": "\"string\""}},

		{"5 >= true", map[string]any{}},
		{"false >= 3", map[string]any{}},
		{"\"hi\" >= \"hi\"", map[string]any{}},
		{"false >= false", map[string]any{}},
		{"foo >= 10", map[string]any{"foo": "\"string\""}},

		{"5 < true", map[string]any{}},
		{"false < 3", map[string]any{}},
		{"\"hi\" < \"hi\"", map[string]any{}},
		{"false < false", map[string]any{}},
		{"foo < 10", map[string]any{"foo": "\"string\""}},

		{"5 <= true", map[string]any{}},
		{"false <= 3", map[string]any{}},
		{"\"hi\" <= \"hi\"", map[string]any{}},
		{"false <= false", map[string]any{}},
		{"foo <= 10", map[string]any{"foo": "\"string\""}},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			t.Parallel()

			_, err := interpret(tt.expression, tt.environment)
			assert.ErrorContains(t, err, "Expected numeric value")
		})
	}
}

func TestNumericTypeHandling(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		expression  string
		environment map[string]any
	}{
		{"fooint == 5", map[string]any{"fooint": int(5)}},
		{"fooint8 == 5", map[string]any{"fooint8": int8(5)}},
		{"fooint16 == 5", map[string]any{"fooint16": int16(5)}},
		{"fooint32 == 5", map[string]any{"fooint32": int32(5)}},
		{"fooint64 == 5", map[string]any{"fooint64": int64(5)}},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			t.Parallel()

			result, err := interpret(tt.expression, tt.environment)
			assert.NilError(t, err)

			assert.Assert(t, result)
		})
	}
}

func TestPrettyInterpreterErrors(t *testing.T) {
	t.Parallel()

	t.Run("Expected a numeric operand", func(t *testing.T) {
		t.Parallel()
		expression := "foo <= true or false"

		_, err := interpret(expression, map[string]any{"foo": int64(5)})
		assert.ErrorContains(t, err, "Expected numeric value")

		expected := strings.Join(
			[]string{
				"Expected numeric operands to \"<=\" operator:",
				"foo <= true or false",
				"    ^^"},
			"\n")

		assert.Equal(t, expected, err.(Error).AsErrorMessage(expression))
	})

	t.Run("Expected a string operand", func(t *testing.T) {
		t.Parallel()
		expression := "foo starts-with \"api\""

		_, err := interpret(expression, map[string]any{"foo": int64(5)})
		assert.ErrorContains(t, err, "Expected string value")

		expected := strings.Join(
			[]string{
				"Expected string operands to \"starts-with\" operator:",
				"foo starts-with \"api\"",
				"    ^^^^^^^^^^^"},
			"\n")

		assert.Equal(t, expected, err.(Error).AsErrorMessage(expression))
	})

	t.Run("Unknown variable", func(t *testing.T) {
		t.Parallel()
		expression := "1 > 1 or \"main\" != foo and false"

		_, err := interpret(expression, map[string]any{})
		assert.ErrorContains(t, err, "Referred to a variable that is not set")

		expected := strings.Join(
			[]string{
				"Referred to a variable \"foo\" that does not exist:",
				"1 > 1 or \"main\" != foo and false",
				"                   ^^^"},
			"\n")

		assert.Equal(t, expected, err.(Error).AsErrorMessage(expression))
	})
}
