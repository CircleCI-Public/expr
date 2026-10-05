// Evaluates expressions with expr's Go implementation for the differential
// fuzzer. See differential.ts for the protocol.
package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/CircleCI-Public/expr/interpreters"
	"github.com/CircleCI-Public/expr/parsers"
	"github.com/CircleCI-Public/expr/scanners"
)

func main() {
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1<<20), 1<<20)
	out := bufio.NewWriter(os.Stdout)
	for in.Scan() {
		fmt.Fprintln(out, run(in.Text()))
		if err := out.Flush(); err != nil {
			panic(err)
		}
	}
	if err := in.Err(); err != nil {
		panic(err)
	}
}

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

func unb64(s string) string {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func run(line string) (response string) {
	defer func() {
		if r := recover(); r != nil {
			response = "X\t" + b64(fmt.Sprint("panic: ", r))
		}
	}()

	fields := strings.Split(line, "\t")
	expression := unb64(fields[0])
	env := map[string]interpreters.Val{}
	for _, f := range fields[1:] {
		parts := strings.Split(f, ",")
		name, kind, value := unb64(parts[0]), parts[1], unb64(parts[2])
		var v any
		switch kind {
		case "b":
			v = value == "true"
		case "s":
			v = value
		case "n":
			n, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				panic(err)
			}
			v = n
		case "p":
			v = regexp.MustCompile(value)
		case "u":
			v = nil
		}
		boxed, err := interpreters.BoxVal(v)
		if err != nil {
			panic(err)
		}
		env[name] = boxed
	}

	toks, err := scanners.New(expression).Scan()
	if err != nil {
		return failure(expression, err)
	}
	e, err := parsers.New(toks).Parse()
	if err != nil {
		return failure(expression, err)
	}
	v, err := interpreters.New(env).Evaluate(e)
	if err != nil {
		return failure(expression, err)
	}

	switch u := v.Unbox().(type) {
	case nil:
		return "R\tu\t"
	case bool:
		return "R\tb\t" + b64(strconv.FormatBool(u))
	case string:
		return "R\ts\t" + b64(u)
	case int64:
		return "R\tn\t" + b64(strconv.FormatInt(u, 10))
	case *regexp.Regexp:
		return "R\tp\t" + b64(u.String())
	default:
		return "X\t" + b64(fmt.Sprintf("unexpected value %T", u))
	}
}

func failure(expression string, err error) string {
	switch e := err.(type) {
	case scanners.Error:
		return errorResponse("Scanner/"+e.Type.Symbol(), "-", string(e.Char), e.Pos, e.AsErrorMessage(expression))
	case parsers.Error:
		return errorResponse("Parser/"+e.Type.Symbol(), e.Token.Type.String(), e.Token.Lexeme, e.Token.CharPos,
			e.AsErrorMessage(expression))
	case interpreters.Error:
		return errorResponse("Interpreter/"+e.Type.Symbol(), e.Token.Type.String(), e.Token.Lexeme, e.Token.CharPos,
			e.AsErrorMessage(expression))
	}
	return "X\t" + b64(fmt.Sprintf("unexpected error %T: %v", err, err))
}

func errorResponse(errorType, tokenType, lexeme string, pos int, message string) string {
	return strings.Join([]string{"E", errorType, tokenType, b64(lexeme), strconv.Itoa(pos), b64(message)}, "\t")
}
