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

package expr

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/circleci/expr/go/interpreter"
	"github.com/circleci/expr/go/parser"
	"github.com/circleci/expr/go/scanner"
)

type rawInput struct {
	Expression  string         `json:"expression"`
	Environment map[string]any `json:"environment"`
}

type input struct {
	Expression  string                     `json:"expression"`
	Environment map[string]interpreter.Val `json:"environment"`
}

func (i *input) UnmarshallJSON(input []byte) error {
	var raw rawInput
	err := json.Unmarshal(input, &raw)
	if err != nil {
		return err
	}

	i.Expression = raw.Expression

	for k, v := range raw.Environment {
		newV, err := interpreter.BoxVal(v)
		if err != nil {
			return err
		}
		i.Environment[k] = newV
	}

	return nil
}

type expectedError struct {
	TokenType string `json:"tokenType"`
	ErrorType string `json:"errorType"`
	Lexeme    string `json:"lexeme"`
	CharPos   int    `json:"charPos"`
}

type expected struct {
	Error  *expectedError   `json:"error,omitempty"`
	Result *interpreter.Val `json:"result,omitempty"`
}

type test struct {
	Name     string `json:",omitempty"`
	Input    input
	Expected expected
}

func loadTest(path string) (test, error) {
	var t test

	data, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}

	err = json.Unmarshal(data, &t)

	if t.Expected.Error == nil && t.Expected.Result == nil {
		u := interpreter.UndefinedVal()
		t.Expected.Result = &u
	}

	return t, err
}

func loadTestData(dir string) (map[string]test, error) {
	tests := make(map[string]test)

	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			test, err := loadTest(path)
			if err != nil {
				return err
			}

			tests[path] = test
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return tests, nil
}

func scanAndParse(expression string) (parser.Expr, error) {
	s := scanner.New(expression)
	tokens, err := s.Scan()
	if err != nil {
		return nil, err
	}
	p := parser.New(tokens)
	e, err := p.Parse()
	if err != nil {
		return nil, err
	}
	return e, nil
}

func mapExpected(res *interpreter.Val, err error) (expected, error) {
	if e, ok := err.(scanner.Error); ok {
		return expected{
			Error: &expectedError{
				ErrorType: fmt.Sprintf("Scanner/%s", e.Type.Symbol()),
				Lexeme:    string(e.Char),
				CharPos:   e.Pos,
			},
		}, nil
	}

	if e, ok := err.(parser.Error); ok {
		return expected{
			Error: &expectedError{
				ErrorType: fmt.Sprintf("Parser/%s", e.Type.Symbol()),
				TokenType: e.Token.Type.String(),
				Lexeme:    e.Token.Lexeme,
				CharPos:   e.Token.CharPos,
			},
		}, nil
	}

	if e, ok := err.(interpreter.Error); ok {
		return expected{
			Error: &expectedError{
				ErrorType: fmt.Sprintf("Interpreter/%s", e.Type.Symbol()),
				TokenType: e.Token.Type.String(),
				Lexeme:    e.Token.Lexeme,
				CharPos:   e.Token.CharPos,
			},
		}, nil
	}

	// An error trying to run the test rather than an error the test expects
	if err != nil {
		return expected{}, err
	}

	return expected{
		// Have to implement an ExprObject in order to return a value
		Result: res,
	}, nil
}

func TestCorpus(t *testing.T) {
	// This matters, since child tests can not be run in parallel if the parent test
	// is not enabled for parallel running
	t.Parallel()

	tests, err := loadTestData("../dev-resources/test-corpus")
	assert.NilError(t, err, "Unable to load test corpus")

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var got interpreter.Val
			e, err := scanAndParse(tt.Input.Expression)
			if err == nil {
				var result bool
				result, err = interpreter.New(tt.Input.Environment).Interpret(e)
				got = interpreter.BoxBool(result)
			}

			res, err := mapExpected(&got, err)
			assert.NilError(t, err)
			assert.DeepEqual(t, tt.Expected, res)
		})
	}
}

func TestEvaluatorCorpus(t *testing.T) {
	t.Parallel()

	tests, err := loadTestData("../dev-resources/evaluator-test-corpus")
	assert.NilError(t, err, "Unable to load evaluator test corpus")

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var got interpreter.Val
			e, err := scanAndParse(tt.Input.Expression)
			if err == nil {
				got, err = interpreter.New(tt.Input.Environment).Evaluate(e)
			}

			res, err := mapExpected(&got, err)
			assert.NilError(t, err)
			assert.DeepEqual(t, tt.Expected, res)
		})
	}
}
