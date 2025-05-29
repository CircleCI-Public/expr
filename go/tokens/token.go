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

package tokens

import "fmt"

type TokenType int

const (
	NOT_FOUND TokenType = iota

	// Grouping
	LEFT_PAREN
	RIGHT_PAREN

	// Operators
	NOT_EQUAL
	EQUAL
	STARTS_WITH
	GREATER
	GREATER_EQUAL
	LESS
	LESS_EQUAL

	// Values
	IDENTIFIER
	STRING
	NUMBER

	// Keywords
	AND
	OR
	NOT
	TRUE
	FALSE

	EOF
)

func (tt TokenType) String() string {
	switch tt {
	case LEFT_PAREN:
		return "LEFT_PAREN"
	case RIGHT_PAREN:
		return "RIGHT_PAREN"

	// Operators
	case NOT_EQUAL:
		return "NOT_EQUAL"
	case EQUAL:
		return "EQUAL"
	case STARTS_WITH:
		return "STARTS_WITH"
	case GREATER:
		return "GREATER"
	case GREATER_EQUAL:
		return "GREATER_EQUAL"
	case LESS:
		return "LESS"
	case LESS_EQUAL:
		return "LESS_EQUAL"

	// Values
	case IDENTIFIER:
		return "IDENTIFIER"
	case STRING:
		return "STRING"
	case NUMBER:
		return "NUMBER"

	// Keywords
	case AND:
		return "AND"
	case OR:
		return "OR"
	case NOT:
		return "NOT"
	case TRUE:
		return "TRUE"
	case FALSE:
		return "FALSE"

	case EOF:
		return "EOF"
	}

	// This can only occur if the switch above isn't exhaustive, if it does,
	// that's a programming error.
	panic("Encountered unknown TokenType value")
}

type Token struct {
	Type    TokenType
	Lexeme  string
	Literal any // Enhancement, build a union type to represent an expr value
	CharPos int
}

func (t *Token) String() string {
	return fmt.Sprintf("<token %s %s %v>[%d]", t.Type, t.Lexeme, t.Literal, t.CharPos)
}
