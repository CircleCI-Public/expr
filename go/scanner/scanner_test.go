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

package scanner

import (
	"fmt"
	"strings"
	"testing"

	"github.com/circleci/expr/go/token"
	"gotest.tools/v3/assert"
)

func TestScansKeywords(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		expression string
		expected   []token.Token
	}{
		{"and",
			[]token.Token{
				{Type: token.AND, Lexeme: "and", CharPos: 0, Literal: "and"},
				{Type: token.EOF, Lexeme: "", CharPos: 3}}},

		{"AND",
			[]token.Token{
				{Type: token.AND, Lexeme: "AND", CharPos: 0, Literal: "AND"},
				{Type: token.EOF, Lexeme: "", CharPos: 3}}},

		{"or",
			[]token.Token{
				{Type: token.OR, Lexeme: "or", CharPos: 0, Literal: "or"},
				{Type: token.EOF, Lexeme: "", CharPos: 2}}},
		{"OR",
			[]token.Token{
				{Type: token.OR, Lexeme: "OR", CharPos: 0, Literal: "OR"},
				{Type: token.EOF, Lexeme: "", CharPos: 2}}},

		{"not",
			[]token.Token{
				{Type: token.NOT, Lexeme: "not", CharPos: 0, Literal: "not"},
				{Type: token.EOF, Lexeme: "", CharPos: 3}}},
		{"NOT",
			[]token.Token{
				{Type: token.NOT, Lexeme: "NOT", CharPos: 0, Literal: "NOT"},
				{Type: token.EOF, Lexeme: "", CharPos: 3}}},

		{"true",
			[]token.Token{
				{Type: token.TRUE, Lexeme: "true", CharPos: 0, Literal: "true"},
				{Type: token.EOF, Lexeme: "", CharPos: 4}}},
		{"TRUE",
			[]token.Token{
				{Type: token.TRUE, Lexeme: "TRUE", CharPos: 0, Literal: "TRUE"},
				{Type: token.EOF, Lexeme: "", CharPos: 4}}},

		{"false",
			[]token.Token{
				{Type: token.FALSE, Lexeme: "false", CharPos: 0, Literal: "false"},
				{Type: token.EOF, Lexeme: "", CharPos: 5}}},
		{"FALSE",
			[]token.Token{
				{Type: token.FALSE, Lexeme: "FALSE", CharPos: 0, Literal: "FALSE"},
				{Type: token.EOF, Lexeme: "", CharPos: 5}}},

		{"starts-with",
			[]token.Token{
				{Type: token.STARTS_WITH, Lexeme: "starts-with", CharPos: 0, Literal: "starts-with"},
				{Type: token.EOF, Lexeme: "", CharPos: 11}}},
		{"STARTS-WITH",
			[]token.Token{
				{Type: token.STARTS_WITH, Lexeme: "STARTS-WITH", CharPos: 0, Literal: "STARTS-WITH"},
				{Type: token.EOF, Lexeme: "", CharPos: 11}}},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			t.Parallel()

			s := New(tt.expression)

			tokens, err := s.Scan()
			assert.NilError(t, err)

			assert.DeepEqual(t, tt.expected, tokens)
		})
	}
}

func TestScansOperators(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		expression string
		expected   []token.Token
	}{
		{"()",
			[]token.Token{
				{Type: token.LEFT_PAREN, Lexeme: "(", CharPos: 0},
				{Type: token.RIGHT_PAREN, Lexeme: ")", CharPos: 1},
				{Type: token.EOF, Lexeme: "", CharPos: 2}}},

		{"!",
			[]token.Token{
				{Type: token.NOT, Lexeme: "!", CharPos: 0},
				{Type: token.EOF, Lexeme: "", CharPos: 1}}},

		{"!= ==",
			[]token.Token{
				{Type: token.NOT_EQUAL, Lexeme: "!=", CharPos: 0},
				{Type: token.EQUAL, Lexeme: "==", CharPos: 3},
				{Type: token.EOF, Lexeme: "", CharPos: 5}}},
		{"> >=",
			[]token.Token{
				{Type: token.GREATER, Lexeme: ">", CharPos: 0},
				{Type: token.GREATER_EQUAL, Lexeme: ">=", CharPos: 2},
				{Type: token.EOF, Lexeme: "", CharPos: 4}}},

		{"< <=",
			[]token.Token{
				{Type: token.LESS, Lexeme: "<", CharPos: 0},
				{Type: token.LESS_EQUAL, Lexeme: "<=", CharPos: 2},
				{Type: token.EOF, Lexeme: "", CharPos: 4}}},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			t.Parallel()

			s := New(tt.expression)

			tokens, err := s.Scan()
			assert.NilError(t, err)

			assert.DeepEqual(t, tt.expected, tokens)
		})
	}

	t.Run("improperly formed operators", func(t *testing.T) {
		t.Parallel()

		s := New("=!")
		_, err := s.Scan()
		assert.Error(t, err, "error scanning expression: Incomplete token, expected \"==\" scanning '!' at 1")
	})
}

func TestScansStrings(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		expression string
		expected   []token.Token
	}{
		{"\"\"",
			[]token.Token{
				{Type: token.STRING, Lexeme: "\"\"", CharPos: 0, Literal: ""},
				{Type: token.EOF, Lexeme: "", CharPos: 2}}},

		{"\"a string\"",
			[]token.Token{
				{Type: token.STRING, Lexeme: "\"a string\"", CharPos: 0, Literal: "a string"},
				{Type: token.EOF, Lexeme: "", CharPos: 10}}},

		{"\"an \\\"escaped\\\" string\"",
			[]token.Token{
				{Type: token.STRING, Lexeme: "\"an \\\"escaped\\\" string\"", CharPos: 0, Literal: "an \"escaped\" string"},
				{Type: token.EOF, Lexeme: "", CharPos: 23}}},

		{"\"backslash \\\\escapes\"",
			[]token.Token{
				{Type: token.STRING, Lexeme: "\"backslash \\\\escapes\"", CharPos: 0, Literal: "backslash \\escapes"},
				{Type: token.EOF, Lexeme: "", CharPos: 21}}},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			t.Parallel()

			s := New(tt.expression)

			tokens, err := s.Scan()
			assert.NilError(t, err)

			assert.DeepEqual(t, tt.expected, tokens)
		})
	}

	t.Run("unterminated string", func(t *testing.T) {
		t.Parallel()

		s := New("text \"an unterminated string")
		_, err := s.Scan()
		assert.Error(t, err, "error scanning expression: Unterminated string scanning '\"' at 5")
	})
}

func TestScansDigits(t *testing.T) {
	t.Parallel()

	for i := range 200 {
		iStr := fmt.Sprintf("%d", i)

		s := New(iStr)
		tokens, err := s.Scan()
		assert.NilError(t, err)

		expected := []token.Token{
			{Type: token.NUMBER, Lexeme: iStr, CharPos: 0, Literal: int64(i)},
			{Type: token.EOF, Lexeme: "", CharPos: len(iStr)}}

		assert.DeepEqual(t, expected, tokens)
	}
}

func TestScansIdentifiers(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		expression string
		expected   []token.Token
	}{
		{"foo",
			[]token.Token{
				{Type: token.IDENTIFIER, Lexeme: "foo", CharPos: 0, Literal: "foo"},
				{Type: token.EOF, Lexeme: "", CharPos: 3}}},

		{"foo.bar",
			[]token.Token{
				{Type: token.IDENTIFIER, Lexeme: "foo.bar", CharPos: 0, Literal: "foo.bar"},
				{Type: token.EOF, Lexeme: "", CharPos: 7}}},

		{"foo-bar",
			[]token.Token{
				{Type: token.IDENTIFIER, Lexeme: "foo-bar", CharPos: 0, Literal: "foo-bar"},
				{Type: token.EOF, Lexeme: "", CharPos: 7}}},

		{"foo_bar",
			[]token.Token{
				{Type: token.IDENTIFIER, Lexeme: "foo_bar", CharPos: 0, Literal: "foo_bar"},
				{Type: token.EOF, Lexeme: "", CharPos: 7}}},

		{"foo_bar?",
			[]token.Token{
				{Type: token.IDENTIFIER, Lexeme: "foo_bar?", CharPos: 0, Literal: "foo_bar?"},
				{Type: token.EOF, Lexeme: "", CharPos: 8}}},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			t.Parallel()

			s := New(tt.expression)

			tokens, err := s.Scan()
			assert.NilError(t, err)

			assert.DeepEqual(t, tt.expected, tokens)
		})
	}

	t.Run("identifiers cannot contain strings of '.'s", func(t *testing.T) {
		t.Parallel()

		s := New("foo..bar")
		_, err := s.Scan()
		assert.Error(t, err, "error scanning expression: Unexpected character scanning '.' at 3")
	})

	t.Run("identifiers cannot start with '.'", func(t *testing.T) {
		t.Parallel()

		s := New(".foo")
		_, err := s.Scan()
		assert.Error(t, err, "error scanning expression: Unexpected character scanning '.' at 0")
	})

	t.Run("identifiers cannot start with '-'", func(t *testing.T) {
		t.Parallel()

		s := New("-foo")
		_, err := s.Scan()
		assert.Error(t, err, "error scanning expression: Unexpected character scanning '-' at 0")
	})

	t.Run("identifiers cannot start with '_'", func(t *testing.T) {
		t.Parallel()

		s := New("_foo")
		_, err := s.Scan()
		assert.Error(t, err, "error scanning expression: Unexpected character scanning '_' at 0")
	})
}

func TestUnexpectedCharacters(t *testing.T) {
	t.Parallel()

	for _, c := range "&*^%$£?/#~:;@'" {
		s := New(string(c))
		_, err := s.Scan()

		assert.Error(t, err, fmt.Sprintf("error scanning expression: Unexpected character scanning %q at 0", c))
	}
}

func TestPrettyScanErrors(t *testing.T) {
	t.Parallel()

	t.Run("Unexpected characters", func(t *testing.T) {
		t.Parallel()
		expression := "2 > 5 && false"

		s := New(expression)
		_, err := s.Scan()
		assert.ErrorContains(t, err, "error scanning expression")

		expected := strings.Join(
			[]string{
				"Unexpected character '&':",
				"2 > 5 && false",
				"      ^"},
			"\n")

		assert.Equal(t, expected, err.(Error).AsErrorMessage(expression))
	})

	t.Run("Incomplete tokens", func(t *testing.T) {
		t.Parallel()
		expression := "foo = 58"

		s := New(expression)
		_, err := s.Scan()
		assert.ErrorContains(t, err, "error scanning expression")

		expected := strings.Join(
			[]string{
				"Incomplete token, expected \"==\", found ' ':",
				"foo = 58",
				"     ^"},
			"\n")

		assert.Equal(t, expected, err.(Error).AsErrorMessage(expression))
	})

	t.Run("Unterminated strings", func(t *testing.T) {
		t.Parallel()
		expression := "foo == \"an unterminated string"

		s := New(expression)
		_, err := s.Scan()
		assert.ErrorContains(t, err, "error scanning expression")

		expected := strings.Join(
			[]string{
				"Unterminated string starting here:",
				"foo == \"an unterminated string",
				"       ^"},
			"\n")

		assert.Equal(t, expected, err.(Error).AsErrorMessage(expression))
	})
}
