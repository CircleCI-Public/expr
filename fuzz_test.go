package expr

import (
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/CircleCI-Public/expr/errors"
	"github.com/CircleCI-Public/expr/interpreters"
	"github.com/CircleCI-Public/expr/interpreters/variableanalyser"
	"github.com/CircleCI-Public/expr/parsers"
	"github.com/CircleCI-Public/expr/scanners"
)

// fuzzEnv has a variable of each kind of value, so fuzzed expressions that
// refer to them reach every operator's type checks.
func fuzzEnv() map[string]interpreters.Val {
	box := func(v any) interpreters.Val {
		b, err := interpreters.BoxVal(v)
		if err != nil {
			panic(err)
		}
		return b
	}
	return map[string]interpreters.Val{
		"b": box(true),
		"f": box(false),
		"n": box(int64(42)),
		"s": box("foo"),
		"e": box(""),
		"p": box(regexp.MustCompile("fo+")),
		"u": interpreters.UndefinedVal(),
	}
}

// checkErrorMessage checks that an error message shows the line of the
// expression containing the error at pos, a position in runes, with a marker
// under it.
func checkErrorMessage(t *testing.T, expression string, pos int, msg string) {
	t.Helper()
	runes := []rune(expression)
	if pos < len(runes) && runes[pos] == '\n' {
		// An error on a newline shows the rest of the expression.
		return
	}
	lineStart := pos
	for lineStart > 0 && runes[lineStart-1] != '\n' {
		lineStart--
	}
	lineEnd := pos
	for lineEnd < len(runes) && runes[lineEnd] != '\n' {
		lineEnd++
	}

	// The preamble can quote a lexeme with newlines in it, so check the end of
	// the message. The marker is as long as the token, which is empty at EOF.
	wantSuffix := "\n" + string(runes[lineStart:lineEnd]) + "\n" + strings.Repeat(" ", pos-lineStart)
	if !strings.HasSuffix(strings.TrimRight(msg, "^"), wantSuffix) {
		t.Fatalf("error at %d in %q has message:\n%s\nwant it to end with:\n%s", pos, expression, msg, wantSuffix)
	}
}

// FuzzExpr checks that scanning, parsing, analysing and interpreting any
// expression either succeeds or returns one of expr's errors, and that the
// error's message can be formatted.
//
// Run with: task fuzz
func FuzzExpr(f *testing.F) {
	for _, dir := range []string{"clj/dev-resources/test-corpus", "clj/dev-resources/evaluator-test-corpus"} {
		tests, err := loadTestData(dir)
		if err != nil {
			f.Fatal(err)
		}
		for _, tt := range tests {
			f.Add(tt.Input.Expression)
		}
	}
	f.Add(`b and not f or s == "foo" and n > 1 and p matches /x/ and s contains e and u`)
	f.Add(`s matches p or (n >= 1 and s starts-with "f")`)

	env := fuzzEnv()

	f.Fuzz(func(t *testing.T, expression string) {
		if !utf8.ValidString(expression) {
			// Expressions come from YAML config, which is always valid UTF-8.
			t.Skip()
		}
		length := utf8.RuneCountInString(expression)

		checkError := func(err error, pos int) {
			t.Helper()
			if pos < 0 || pos > length {
				t.Fatalf("error position %d outside expression of length %d: %v", pos, length, err)
			}
			e, ok := err.(errors.Errors)
			if !ok {
				t.Fatalf("unexpected error type %T: %v", err, err)
			}
			checkErrorMessage(t, expression, pos, e.AsErrorMessage(expression))
		}

		toks, err := scanners.New(expression).Scan()
		if err != nil {
			e, ok := err.(scanners.Error)
			if !ok {
				t.Fatalf("unexpected scanner error type %T: %v", err, err)
			}
			checkError(e, e.Pos)
			return
		}

		e, err := parsers.New(toks).Parse()
		if err != nil {
			pe, ok := err.(parsers.Error)
			if !ok {
				t.Fatalf("unexpected parser error type %T: %v", err, err)
			}
			checkError(pe, pe.Token.CharPos)
			return
		}

		if _, err := variableanalyser.New().GatherVariables(e); err != nil {
			t.Fatalf("analyser failed: %v", err)
		}

		i := interpreters.New(env)
		got, err := i.Evaluate(e)
		if err != nil {
			ie, ok := err.(interpreters.Error)
			if !ok {
				t.Fatalf("unexpected interpreter error type %T: %v", err, err)
			}
			checkError(ie, ie.Token.CharPos)
		}

		truthy, ierr := i.Interpret(e)
		if (err == nil) != (ierr == nil) {
			t.Fatalf("Evaluate error %v but Interpret error %v", err, ierr)
		}
		if err == nil && truthy != got.IsTruthy() {
			t.Fatalf("Interpret returned %v but Evaluate returned %v", truthy, got.Unbox())
		}
	})
}
