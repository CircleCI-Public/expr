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

package scanners

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"

	"github.com/CircleCI-Public/expr/tokens"
)

func TestScansKeywords(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		expression string
		expected   []tokens.Token
	}{
		{"and",
			[]tokens.Token{
				{Type: tokens.AND, Lexeme: "and", CharPos: 0, Literal: "and"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 3}}},

		{"AND",
			[]tokens.Token{
				{Type: tokens.AND, Lexeme: "AND", CharPos: 0, Literal: "AND"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 3}}},

		{"or",
			[]tokens.Token{
				{Type: tokens.OR, Lexeme: "or", CharPos: 0, Literal: "or"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 2}}},
		{"OR",
			[]tokens.Token{
				{Type: tokens.OR, Lexeme: "OR", CharPos: 0, Literal: "OR"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 2}}},

		{"not",
			[]tokens.Token{
				{Type: tokens.NOT, Lexeme: "not", CharPos: 0, Literal: "not"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 3}}},
		{"NOT",
			[]tokens.Token{
				{Type: tokens.NOT, Lexeme: "NOT", CharPos: 0, Literal: "NOT"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 3}}},

		{"true",
			[]tokens.Token{
				{Type: tokens.TRUE, Lexeme: "true", CharPos: 0, Literal: "true"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 4}}},
		{"TRUE",
			[]tokens.Token{
				{Type: tokens.TRUE, Lexeme: "TRUE", CharPos: 0, Literal: "TRUE"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 4}}},

		{"false",
			[]tokens.Token{
				{Type: tokens.FALSE, Lexeme: "false", CharPos: 0, Literal: "false"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 5}}},
		{"FALSE",
			[]tokens.Token{
				{Type: tokens.FALSE, Lexeme: "FALSE", CharPos: 0, Literal: "FALSE"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 5}}},

		{"contains",
			[]tokens.Token{
				{Type: tokens.BUILTIN, Lexeme: "contains", CharPos: 0, Literal: "contains"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 8}}},
		{"CONTAINS",
			[]tokens.Token{
				{Type: tokens.BUILTIN, Lexeme: "CONTAINS", CharPos: 0, Literal: "CONTAINS"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 8}}},

		{"starts-with",
			[]tokens.Token{
				{Type: tokens.BUILTIN, Lexeme: "starts-with", CharPos: 0, Literal: "starts-with"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 11}}},
		{"STARTS-WITH",
			[]tokens.Token{
				{Type: tokens.BUILTIN, Lexeme: "STARTS-WITH", CharPos: 0, Literal: "STARTS-WITH"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 11}}},

		{"matches",
			[]tokens.Token{
				{Type: tokens.BUILTIN, Lexeme: "matches", CharPos: 0, Literal: "matches"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 7}}},
		{"MATCHES",
			[]tokens.Token{
				{Type: tokens.BUILTIN, Lexeme: "MATCHES", CharPos: 0, Literal: "MATCHES"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 7}}},
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
		expected   []tokens.Token
	}{
		{"()",
			[]tokens.Token{
				{Type: tokens.LEFT_PAREN, Lexeme: "(", CharPos: 0},
				{Type: tokens.RIGHT_PAREN, Lexeme: ")", CharPos: 1},
				{Type: tokens.EOF, Lexeme: "", CharPos: 2}}},

		{"!",
			[]tokens.Token{
				{Type: tokens.NOT, Lexeme: "!", CharPos: 0},
				{Type: tokens.EOF, Lexeme: "", CharPos: 1}}},

		{"!= ==",
			[]tokens.Token{
				{Type: tokens.NOT_EQUAL, Lexeme: "!=", CharPos: 0},
				{Type: tokens.EQUAL, Lexeme: "==", CharPos: 3},
				{Type: tokens.EOF, Lexeme: "", CharPos: 5}}},
		{"> >=",
			[]tokens.Token{
				{Type: tokens.GREATER, Lexeme: ">", CharPos: 0},
				{Type: tokens.GREATER_EQUAL, Lexeme: ">=", CharPos: 2},
				{Type: tokens.EOF, Lexeme: "", CharPos: 4}}},

		{"< <=",
			[]tokens.Token{
				{Type: tokens.LESS, Lexeme: "<", CharPos: 0},
				{Type: tokens.LESS_EQUAL, Lexeme: "<=", CharPos: 2},
				{Type: tokens.EOF, Lexeme: "", CharPos: 4}}},
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
		expected   []tokens.Token
	}{
		{"\"\"",
			[]tokens.Token{
				{Type: tokens.STRING, Lexeme: "\"\"", CharPos: 0, Literal: ""},
				{Type: tokens.EOF, Lexeme: "", CharPos: 2}}},

		{"\"a string\"",
			[]tokens.Token{
				{Type: tokens.STRING, Lexeme: "\"a string\"", CharPos: 0, Literal: "a string"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 10}}},

		{"\"an \\\"escaped\\\" string\"",
			[]tokens.Token{
				{Type: tokens.STRING, Lexeme: "\"an \\\"escaped\\\" string\"", CharPos: 0, Literal: "an \"escaped\" string"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 23}}},

		{"\"backslash \\\\escapes\"",
			[]tokens.Token{
				{Type: tokens.STRING, Lexeme: "\"backslash \\\\escapes\"", CharPos: 0, Literal: "backslash \\escapes"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 21}}},
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

func TestScansPatterns(t *testing.T) {
	t.Parallel()
	t.Run("well-formed patterns", func(t *testing.T) {
		var tests = []struct {
			expression string
			expected   []tokens.Token
		}{
			{"//",
				[]tokens.Token{
					{Type: tokens.PATTERN, Lexeme: "//", CharPos: 0, Literal: regexp.MustCompile("")},
					{Type: tokens.EOF, Lexeme: "", CharPos: 2}}},

			{"/a pattern/",
				[]tokens.Token{
					{Type: tokens.PATTERN, Lexeme: "/a pattern/", CharPos: 0, Literal: regexp.MustCompile("a pattern")},
					{Type: tokens.EOF, Lexeme: "", CharPos: 11}}},

			{"/an \\/escaped\\/ pattern/",
				[]tokens.Token{
					{Type: tokens.PATTERN, Lexeme: "/an \\/escaped\\/ pattern/", CharPos: 0, Literal: regexp.MustCompile("an /escaped/ pattern")},
					{Type: tokens.EOF, Lexeme: "", CharPos: 24}}},

			{"/backslash \\\\escapes/",
				[]tokens.Token{
					{Type: tokens.PATTERN, Lexeme: "/backslash \\\\escapes/", CharPos: 0, Literal: regexp.MustCompile(`backslash \\escapes`)},
					{Type: tokens.EOF, Lexeme: "", CharPos: 21}}},

			// escaped pattern metacharacters
			{`/\*/`,
				[]tokens.Token{
					{Type: tokens.PATTERN, Lexeme: `/\*/`, CharPos: 0, Literal: regexp.MustCompile(`\*`)},
					{Type: tokens.EOF, Lexeme: "", CharPos: 4}}},

			{`/\+/`,
				[]tokens.Token{
					{Type: tokens.PATTERN, Lexeme: `/\+/`, CharPos: 0, Literal: regexp.MustCompile(`\+`)},
					{Type: tokens.EOF, Lexeme: "", CharPos: 4}}},

			{`/\?/`,
				[]tokens.Token{
					{Type: tokens.PATTERN, Lexeme: `/\?/`, CharPos: 0, Literal: regexp.MustCompile(`\?`)},
					{Type: tokens.EOF, Lexeme: "", CharPos: 4}}},

			{`/\|/`,
				[]tokens.Token{
					{Type: tokens.PATTERN, Lexeme: `/\|/`, CharPos: 0, Literal: regexp.MustCompile(`\|`)},
					{Type: tokens.EOF, Lexeme: "", CharPos: 4}}},

			{`/\(\)/`,
				[]tokens.Token{
					{Type: tokens.PATTERN, Lexeme: `/\(\)/`, CharPos: 0, Literal: regexp.MustCompile(`\(\)`)},
					{Type: tokens.EOF, Lexeme: "", CharPos: 6}}},
		}

		for _, tt := range tests {
			t.Run(tt.expression, func(t *testing.T) {
				t.Parallel()

				s := New(tt.expression)

				tokens, err := s.Scan()
				assert.NilError(t, err)

				assert.DeepEqual(t, tt.expected, tokens, cmp.Transformer("regexps", func(v *regexp.Regexp) string { return v.String() }))
			})
		}
	})

	t.Run("non-ascii characters are disallowed", func(t *testing.T) {
		t.Parallel()

		_, err := New("/⏰/").Scan()
		assert.Error(t, err, "error scanning expression: Invalid pattern character scanning '⏰' at 1")
	})

	t.Run("unicode escape sequences are disallowed", func(t *testing.T) {
		t.Parallel()

		_, err := New("/\u23f0/").Scan()
		assert.Error(t, err, "error scanning expression: Invalid pattern character scanning '\u23f0' at 1")
	})

	t.Run("passing a unicode escape through to the regular expression engine is disallowed", func(t *testing.T) {
		t.Parallel()

		_, err := New("/\\u23f0/").Scan()
		assert.Error(t, err, "error scanning expression: Invalid pattern character scanning '\\' at 1")
	})

	t.Run("unterminated patterns", func(t *testing.T) {
		t.Parallel()

		_, err := New("text /an unterminated pattern").Scan()
		assert.Error(t, err, "error scanning expression: Unterminated pattern scanning '/' at 5")
	})
}

func TestScansDigits(t *testing.T) {
	t.Parallel()

	for i := range 200 {
		iStr := fmt.Sprintf("%d", i)

		s := New(iStr)
		toks, err := s.Scan()
		assert.NilError(t, err)

		expected := []tokens.Token{
			{Type: tokens.NUMBER, Lexeme: iStr, CharPos: 0, Literal: int64(i)},
			{Type: tokens.EOF, Lexeme: "", CharPos: len(iStr)}}

		assert.DeepEqual(t, expected, toks)
	}
}

func TestScansIdentifiers(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		expression string
		expected   []tokens.Token
	}{
		{"foo",
			[]tokens.Token{
				{Type: tokens.IDENTIFIER, Lexeme: "foo", CharPos: 0, Literal: "foo"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 3}}},

		{"foo.bar",
			[]tokens.Token{
				{Type: tokens.IDENTIFIER, Lexeme: "foo.bar", CharPos: 0, Literal: "foo.bar"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 7}}},

		{"a.b.c",
			[]tokens.Token{
				{Type: tokens.IDENTIFIER, Lexeme: "a.b.c", CharPos: 0, Literal: "a.b.c"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 5}}},

		{"foo-bar",
			[]tokens.Token{
				{Type: tokens.IDENTIFIER, Lexeme: "foo-bar", CharPos: 0, Literal: "foo-bar"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 7}}},

		{"foo_bar",
			[]tokens.Token{
				{Type: tokens.IDENTIFIER, Lexeme: "foo_bar", CharPos: 0, Literal: "foo_bar"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 7}}},

		{"foo_bar?",
			[]tokens.Token{
				{Type: tokens.IDENTIFIER, Lexeme: "foo_bar?", CharPos: 0, Literal: "foo_bar?"},
				{Type: tokens.EOF, Lexeme: "", CharPos: 8}}},
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

	for _, c := range "&*^%$£?#~:;@'" {
		s := New(string(c))
		_, err := s.Scan()

		assert.Error(t, err, fmt.Sprintf("error scanning expression: Unexpected character scanning '%c' at 0", c))
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

	t.Run("Invalid numeric literal", func(t *testing.T) {
		t.Parallel()
		expression := "foo < 9223372036854775808"

		s := New(expression)
		_, err := s.Scan()
		assert.ErrorContains(t, err, "error scanning expression")

		expected := strings.Join(
			[]string{
				"Invalid numeric literal, numbers can range from 0 to 2^63 - 1:",
				"foo < 9223372036854775808",
				"      ^",
			},
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

	t.Run("Unterminated patterns", func(t *testing.T) {
		t.Parallel()
		expression := "foo matches /an unterminated pattern"

		s := New(expression)
		_, err := s.Scan()
		assert.ErrorContains(t, err, "error scanning expression")

		expected := strings.Join(
			[]string{
				"Unterminated pattern starting here:",
				"foo matches /an unterminated pattern",
				"            ^"},
			"\n")

		assert.Equal(t, expected, err.(Error).AsErrorMessage(expression))
	})

	t.Run("Invalid pattern character", func(t *testing.T) {
		t.Parallel()
		expression := "foo matches /hello \u23f0/"

		s := New(expression)
		_, err := s.Scan()
		assert.ErrorContains(t, err, "error scanning expression")

		expected := strings.Join(
			[]string{
				"Invalid pattern character, only ASCII and Latin-1 are allowed in patterns:",
				"foo matches /hello \u23f0/",
				"                   ^"},
			"\n")

		assert.Equal(t, expected, err.(Error).AsErrorMessage(expression))
	})

	t.Run("Bad pattern syntax", func(t *testing.T) {
		t.Parallel()
		expression := "foo matches /hello (world/"

		s := New(expression)
		_, err := s.Scan()
		assert.ErrorContains(t, err, "error scanning expression")

		expected := strings.Join(
			[]string{
				"Syntax error in pattern:",
				"foo matches /hello (world/",
				"            ^"},
			"\n")

		assert.Equal(t, expected, err.(Error).AsErrorMessage(expression))
	})

	t.Run("Overly long pattern", func(t *testing.T) {
		t.Parallel()
		expression := "foo matches /xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/"

		s := New(expression)
		_, err := s.Scan()
		assert.ErrorContains(t, err, "error scanning expression")

		expected := strings.Join(
			[]string{
				"Pattern length exceeded, limit is 256 characters:",
				"foo matches /xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/",
				"            ^",
			},
			"\n")

		assert.Equal(t, expected, err.(Error).AsErrorMessage(expression))
	})
}
