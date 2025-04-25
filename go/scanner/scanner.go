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

package scanner

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/circleci/expr/go/errors"
	"github.com/circleci/expr/go/token"
)

type errorType string

const (
	UNEXPECTED_CHARACTER errorType = "Unexpected character"
	INCOMPLETE_EQUALS    errorType = "Incomplete token, expected \"==\""
	UNTERMINATED_STRING  errorType = "Unterminated string"
)

func (et errorType) Symbol() string {
	switch et {
	case UNEXPECTED_CHARACTER:
		return "UNEXPECTED_CHARACTER"
	case INCOMPLETE_EQUALS:
		return "INCOMPLETE_EQUALS"
	case UNTERMINATED_STRING:
		return "UNTERMINATED_STRING"
	}

	panic("Encountered unknown Scanner errorType value")
}

type Error struct {
	Type errorType
	Char rune
	Pos  int
}

func (e Error) Error() string {
	return fmt.Sprintf("error scanning expression: %s scanning %q at %d", e.Type, e.Char, e.Pos)
}

// Return a multiline error message describing this error.
//
// Returns a string of the form:
//
// explanation
// line of expression containing the error
// marker line pointing at the error token
//
// E.g.
// "Unexpected character '&':\n" +
// "2 > 5 && false\n" +
// "      ^"
func (e Error) AsErrorMessage(expression string) string {
	switch e.Type {
	case UNEXPECTED_CHARACTER:
		return errors.ErrorMessage(fmt.Sprintf("Unexpected character %q:", e.Char),
			expression,
			e.Pos,
			1)
	case INCOMPLETE_EQUALS:
		return errors.ErrorMessage(fmt.Sprintf("Incomplete token, expected \"==\", found %q:", e.Char),
			expression,
			e.Pos,
			1)
	case UNTERMINATED_STRING:
		return errors.ErrorMessage("Unterminated string starting here:",
			expression,
			e.Pos,
			1)
	default:
		return fmt.Sprintf("Unknown error scanning expression; '%s'", expression)
	}
}

var keywords = map[string]token.TokenType{
	"starts-with": token.STARTS_WITH,
	"STARTS-WITH": token.STARTS_WITH,
	"and":         token.AND,
	"AND":         token.AND,
	"or":          token.OR,
	"OR":          token.OR,
	"not":         token.NOT,
	"NOT":         token.NOT,
	"true":        token.TRUE,
	"TRUE":        token.TRUE,
	"false":       token.FALSE,
	"FALSE":       token.FALSE,
}

type scanner struct {
	source  []rune
	start   int
	current int
	tokens  []token.Token
}

func New(source string) *scanner {
	return &scanner{
		source: []rune(source),
	}
}

// Scans the source string and produces a List of Tokens scanned from the
// String.
//
// Returns an Error if an error is detected while scanning the source
// string. See errorType for possible error cases.
func (s *scanner) Scan() ([]token.Token, error) {
	for !s.eof() {
		s.start = s.current
		err := s.scanToken()
		if err != nil {
			return nil, err
		}
	}

	s.tokens = append(s.tokens, token.Token{Type: token.EOF, Lexeme: "", Literal: nil, CharPos: s.current})

	tokens := make([]token.Token, len(s.tokens))
	copy(tokens, s.tokens)

	return tokens, nil
}

// Scan a single token.
//
// Consumes characters and adds created tokens to the `tokens` list.
func (s *scanner) scanToken() error {
	c := s.advance()
	switch c {
	// grouping
	case '(':
		s.addToken(token.LEFT_PAREN)
	case ')':
		s.addToken(token.RIGHT_PAREN)
	// operators
	case '!':
		if s.match('=') {
			s.addToken(token.NOT_EQUAL)
		} else {
			s.addToken(token.NOT)
		}
	case '=':
		if err := s.equal(); err != nil {
			return err
		}
	case '>':
		if s.match('=') {
			s.addToken(token.GREATER_EQUAL)
		} else {
			s.addToken(token.GREATER)
		}
	case '<':
		if s.match('=') {
			s.addToken(token.LESS_EQUAL)
		} else {
			s.addToken(token.LESS)
		}

		// Ignore whitespace
	case ' ':
		break
	case '\t':
		break
	case '\r':
		break
	case '\n':
		break

	case '"':
		if err := s.string(); err != nil {
			return err
		}
	default:
		if isDigit(c) {
			if err := s.number(); err != nil {
				return err
			}
		} else if isAlpha(c) {
			s.identifier()
		} else {
			return Error{Type: UNEXPECTED_CHARACTER, Char: c, Pos: s.current - 1}
		}
	}

	return nil
}

// Match an "==" token.
//
// Consumes the token's characters and adds a Token to the `tokens` list.
func (s *scanner) equal() error {
	c := s.peek()
	if c != '=' {
		return Error{Type: INCOMPLETE_EQUALS, Char: c, Pos: s.current}
	}
	s.advance()
	s.addToken(token.EQUAL)

	return nil
}

// Match a string token.
//
// Consumes all characters from the starting " to the terminating ". Adds a
// Token with the string content (excluding surrounding double-quotes) to the
// `tokens` list.
//
// Double-quote and backslach characters can be embedded by escaping with
// `\`: e.g. `\"`, `\\`
//
// Returns an Error if the string is not terminated.
func (s *scanner) string() error {
	for s.peek() != '"' && !s.eof() {
		// Deal with escape characters
		if s.peek() == '\\' && isEscapableChar(s.peekNext()) {
			s.advance()
		}
		s.advance()
	}

	if s.eof() {
		return Error{Type: UNTERMINATED_STRING, Char: '"', Pos: s.start}
	}

	s.advance()
	v := string(s.source[s.start+1 : s.current-1])
	v = strings.ReplaceAll(v, "\\\"", "\"")
	v = strings.ReplaceAll(v, "\\\\", "\\")

	s.addTokenLiteral(token.STRING, v)

	return nil
}

// Returns true if `c` is a character that could be escaped.
//
// False otherwise.
func isEscapableChar(c rune) bool {
	return c == '"' || c == '\\'
}

func (s *scanner) number() error {
	for isDigit(s.peek()) {
		s.advance()
	}

	v, err := strconv.ParseInt(s.lexeme(), 10, 64)
	if err != nil {
		return err
	}

	s.addTokenLiteral(token.NUMBER, v)
	return nil
}

// Match an identifier.
//
// Consumes the token's characters and adds a Token to the `tokens` list.
func (s *scanner) identifier() {
	if isIdentifierTail(s.peek()) {
		s.advance()
		for isIdentifierTail(s.peek()) || (s.peek() == '.' && s.peekNext() != '.') {
			s.advance()
		}
	}

	identifier := s.lexeme()
	tokenType := keywords[identifier]
	if tokenType == token.NOT_FOUND {
		tokenType = token.IDENTIFIER
	}

	s.addTokenLiteral(tokenType, identifier)
}

// Add a token without a literal value to the `tokens` list.
func (s *scanner) addToken(t token.TokenType) {
	s.tokens = append(s.tokens, token.Token{
		Type:    t,
		Lexeme:  s.lexeme(),
		Literal: nil,
		CharPos: s.start,
	})
}

// Add a token with a literal value to the `tokens` list.
func (s *scanner) addTokenLiteral(t token.TokenType, literal any) {
	s.tokens = append(s.tokens, token.Token{
		Type:    t,
		Lexeme:  s.lexeme(),
		Literal: literal,
		CharPos: s.start,
	})
}

// Returns true if `c` is allowed in the "tail" of an identifier.
//
// The "tail" is every part of the identifier other than the first character.
// Returns true if `c` is alpha-numeric, or '-', or '_', false otherwise.
func isIdentifierTail(c rune) bool {
	return isAlpha(c) || isDigit(c) || c == '-' || c == '_' || c == '?'
}

// Returns true if `c` is an alphabetic character false otherwise.
//
// Accepts:
// - 'a' to 'z'
// - 'A' to 'Z'
// - '_'
func isAlpha(c rune) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// Returns true if `c` is any of the characters '0' through '9' (inclusive).
//
// False otherwise.
func isDigit(c rune) bool {
	return c >= '0' && c <= '9'
}

// Returns true if the next character in the source is the same as 'expected'.
// If there is a match it is consumed.
// If there is no match the character source is unchanged.
func (s *scanner) match(expected rune) bool {
	if s.eof() {
		return false
	}
	if s.peek() != expected {
		return false
	}

	s.advance()
	return true
}

// Gets the next character in the source without consuming.
//
// Returns \0 if there is no character to peek.
func (s *scanner) peek() rune {
	if s.eof() {
		return '\x00'
	}

	return s.source[s.current]
}

// Gets the next + 1 character in the source without consuming.
//
// Returns \0 if there is no character to peekNext.
func (s *scanner) peekNext() rune {
	if s.eofN(s.current + 1) {
		return '\x00'
	}

	return s.source[s.current+1]
}

// Consume and return the next character from source.
func (s *scanner) advance() rune {
	c := s.source[s.current]
	s.current += 1
	return c
}

// Returns true if the end of source has been reached.
// False otherwise
func (s *scanner) eof() bool {
	return s.eofN(len(s.source))
}

// Returns true if pos is beyond the end of source.
// False otherwise.
func (s *scanner) eofN(pos int) bool {
	return s.current >= pos
}

// Return the current lexeme.
func (s *scanner) lexeme() string {
	return string(s.source[s.start:s.current])
}
