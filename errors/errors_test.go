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

package errors

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestErrorMessage(t *testing.T) {
	t.Parallel()

	t.Run("Pinpoints errors in single-line expressions", func(t *testing.T) {
		t.Parallel()

		expected := strings.Join(
			[]string{
				"Preamble message:",
				"some.number > baz or true",
				"              ^^^"},
			"\n")
		assert.Equal(t, expected, ErrorMessage("Preamble message:", "some.number > baz or true", 14, 3))
	})

	t.Run("Pinpoints errors in multi-line expressions", func(t *testing.T) {
		t.Parallel()

		expected := strings.Join(
			[]string{
				"Preamble message:",
				"some.number > baz or true",
				"              ^^^"},
			"\n")
		assert.Equal(t, expected, ErrorMessage(
			"Preamble message:",
			strings.Join([]string{
				"foo.bar == \"main\" and",
				"some.number > baz or true"},
				"\n"),
			36,
			3))
	})

	t.Run("Error is on the end of a line", func(t *testing.T) {
		t.Parallel()

		expected := strings.Join(
			[]string{
				"Preamble message:",
				"some.number > baz or t",
				"                     ^"},
			"\n")
		assert.Equal(t, expected, ErrorMessage(
			"Preamble message:",
			strings.Join([]string{
				"foo.bar == \"main\" and",
				"some.number > baz or t",
				"1 <= 15"},
				"\n"),
			43,
			1))
	})

	t.Run("Error is after the end of a line", func(t *testing.T) {
		t.Parallel()

		expected := strings.Join(
			[]string{
				"Preamble message:",
				"foo.bar.baz == (1 > 5",
				"                     ^"},
			"\n")
		assert.Equal(t, expected, ErrorMessage("Preamble message:", "foo.bar.baz == (1 > 5", 21, 1))
	})

	t.Run("Positions are in runes", func(t *testing.T) {
		t.Parallel()

		expected := strings.Join(
			[]string{
				"Preamble message:",
				"\"é\" or /ÿ/",
				"       ^^^"},
			"\n")
		assert.Equal(t, expected, ErrorMessage(
			"Preamble message:",
			strings.Join([]string{
				"\"日本語\" and",
				"\"é\" or /ÿ/"},
				"\n"),
			17,
			3))
	})
}
