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

package interpreters

import (
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/CircleCI-Public/expr/parsers"
	"github.com/CircleCI-Public/expr/scanners"
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

	s := scanners.New(expression)

	tokens, err := s.Scan()
	if err != nil {
		return false, err
	}

	p := parsers.New(tokens)
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

	t.Run("Expected a pattern operand", func(t *testing.T) {
		t.Parallel()
		expression := "\"hello\" matches 5"

		_, err := interpret(expression, map[string]any{})
		assert.ErrorContains(t, err, "Expected regular expression value")

		expected := strings.Join(
			[]string{
				"Expected the right operand to \"matches\" operator to be a pattern:",
				"\"hello\" matches 5",
				"        ^^^^^^^"},
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

func TestPushingBoundaries(t *testing.T) {
	t.Parallel()

	t.Run("many nots", func(t *testing.T) {
		expression := strings.Repeat("not ", 1000) + "true"

		start := time.Now()
		result, err := interpret(expression, map[string]any{})
		end := time.Now()

		assert.NilError(t, err)

		assert.Assert(t, result)
		assert.Check(t, end.Sub(start) <= 100*time.Millisecond)
	})
}
