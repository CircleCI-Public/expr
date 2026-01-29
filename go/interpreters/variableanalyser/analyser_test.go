package variableanalyser_test

import (
	"testing"

	"github.com/circleci/expr/go/interpreters/variableanalyser"
	"github.com/circleci/expr/go/parsers"
	"github.com/circleci/expr/go/scanners"
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
