// Reports whether Go's regexp package accepts patterns, and whether they
// match inputs as expr's Go implementation does. See update.ts for the
// protocol.
package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/CircleCI-Public/expr/interpreters"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1<<20), 1<<20)
	out := bufio.NewWriter(os.Stdout)

	var pattern *interpreters.Val
	for in.Scan() {
		kind, data, _ := strings.Cut(in.Text(), "\t")
		value, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return err
		}
		switch kind {
		case "P":
			pattern = nil
			re, err := regexp.Compile(string(value))
			if err != nil {
				fmt.Fprintf(out, "V\t0\t%s\n", base64.StdEncoding.EncodeToString([]byte(err.Error())))
				continue
			}
			v, err := interpreters.BoxVal(re)
			if err != nil {
				return err
			}
			pattern = &v
			fmt.Fprintln(out, "V\t1")
		case "I":
			if pattern == nil {
				fmt.Fprintln(out, "M\t-")
				continue
			}
			input, err := interpreters.BoxVal(string(value))
			if err != nil {
				return err
			}
			// The same call as the interpreter's matches builtin.
			matches, err := pattern.Matches(input)
			if err != nil {
				return err
			}
			if matches {
				fmt.Fprintln(out, "M\t1")
			} else {
				fmt.Fprintln(out, "M\t0")
			}
		}
	}
	if err := in.Err(); err != nil {
		return err
	}
	return out.Flush()
}
