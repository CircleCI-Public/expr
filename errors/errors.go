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
	"slices"
	"strings"
)

type Errors interface {
	AsErrorMessage(expression string) string
}

// ErrorMessage returns a multi-line message pointing at an error. errorPos
// and errorLength are in runes, as token positions are.
func ErrorMessage(preamble, expression string, errorPos, errorLength int) string {
	runes := []rune(expression)

	// This `+ 1` needs a little bit of explaining.
	// The call to `lastIndex` returns the index of the last occurrence of a
	// newline before the error position. If there isn't a newline preceding
	// the error position then `lastIndex` returns -1
	//
	// In either case, incrementing the result of `lastIndex` gets the start
	// index of the line the error occurred on:
	// 1. A newline was found: `lastIndex` is the index of the newline, the
	//    next index is the start of the line.
	// 2. A newline was not found: `lastIndex` is -1, the next index (0) is
	//    the start of the line.
	lineStart := lastIndex(runes[:errorPos], '\n') + 1
	lineEndOffset := slices.Index(runes[errorPos:], '\n')

	var errorEnd int
	if lineEndOffset > 0 {
		// lineEndOffset is the offset of the first newline from the error
		// position. Add errorPos to lineEnd to find the index of the newline in
		// the expression
		errorEnd = errorPos + lineEndOffset
	} else {
		errorEnd = len(runes)
	}

	errorLine := string(runes[lineStart:errorEnd])

	markerLine := strings.Repeat(" ", errorPos-lineStart) + strings.Repeat("^", errorLength)

	return strings.Join([]string{preamble, errorLine, markerLine}, "\n")
}

func lastIndex(runes []rune, r rune) int {
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == r {
			return i
		}
	}
	return -1
}
