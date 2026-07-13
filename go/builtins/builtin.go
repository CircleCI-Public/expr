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

package builtins

type Type int

const (
	NOT_FOUND Type = iota

	MATCHES
	STARTS_WITH
)

func (bt Type) String() string {
	switch bt {
	case MATCHES:
		return "MATCHES"
	case STARTS_WITH:
		return "STARTS_WITH"
	}

	// This can only occur if the switch above isn't exhaustive, if it does,
	// that's a programming error.
	panic("Encountered unknown builtin Type value")
}

func ForLexeme(lexeme string) Type {
	switch lexeme {
	case "matches":
		return MATCHES
	case "MATCHES":
		return MATCHES
	case "starts-with":
		return STARTS_WITH
	case "STARTS-WITH":
		return STARTS_WITH
	default:
		return NOT_FOUND
	}
}
