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

package variableanalyser_test

import (
	"testing"

	"github.com/CircleCI-Public/expr/interpreters/variableanalyser"
	"github.com/CircleCI-Public/expr/parsers"
	"github.com/CircleCI-Public/expr/scanners"
	"gotest.tools/v3/assert"
)

func TestIdentifierVisitor(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		identifiers []string
	}{
		{
			name:        "simple literal number",
			input:       "42",
			identifiers: nil,
		},
		{
			name:        "simple literal string",
			input:       `"hello"`,
			identifiers: nil,
		},
		{
			name:        "simple identifier",
			input:       "foo",
			identifiers: []string{"foo"},
		},
		{
			name:        "binary expression",
			input:       "a == 1",
			identifiers: []string{"a"},
		},
		{
			name:        "infix builtin function",
			input:       "a matches /hi/",
			identifiers: []string{"a"},
		},
		{
			name:        "logical and",
			input:       "a == 1 and b == 2",
			identifiers: []string{"a", "b"},
		},
		{
			name:        "logical or",
			input:       "a == 1 or b == 2",
			identifiers: []string{"a", "b"},
		},
		{
			name:        "unary not",
			input:       "not true",
			identifiers: nil,
		},
		{
			name:        "grouping",
			input:       "(a == 1)",
			identifiers: []string{"a"},
		},
		{
			name:        "complex expression",
			input:       "(a == 1 and b == 2) or c == 3",
			identifiers: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scanners.New(tt.input)
			tokens, err := s.Scan()
			assert.NilError(t, err)

			p := parsers.New(tokens)
			expr, err := p.Parse()
			assert.NilError(t, err)

			v := variableanalyser.New()
			identifiers, err := v.GatherVariables(expr)
			assert.NilError(t, err)
			assert.DeepEqual(t, tt.identifiers, identifiers)
		})
	}
}
