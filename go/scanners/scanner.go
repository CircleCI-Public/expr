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

package scanners

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/circleci/expr/go/errors"
	"github.com/circleci/expr/go/tokens"
)

type errorType string

const (
	UNEXPECTED_CHARACTER      errorType = "Unexpected character"
	INCOMPLETE_EQUALS         errorType = "Incomplete token, expected \"==\""
	INVALID_NUMERIC_LITERAL   errorType = "Invalid numeric literal"
	UNTERMINATED_STRING       errorType = "Unterminated string"
	UNTERMINATED_PATTERN      errorType = "Unterminated pattern"
	INVALID_PATTERN_CHARACTER errorType = "Invalid pattern character"
	INVALID_PATTERN           errorType = "Invalid pattern"
	PATTERN_TOO_LONG          errorType = "Pattern too long"
)

const MAX_PATTERN_LENGTH = 128

func (et errorType) Symbol() string {
	switch et {
	case UNEXPECTED_CHARACTER:
		return "UNEXPECTED_CHARACTER"
	case INCOMPLETE_EQUALS:
		return "INCOMPLETE_EQUALS"
	case INVALID_NUMERIC_LITERAL:
		return "INVALID_NUMERIC_LITERAL"
	case UNTERMINATED_STRING:
		return "UNTERMINATED_STRING"
	case UNTERMINATED_PATTERN:
		return "UNTERMINATED_PATTERN"
	case INVALID_PATTERN_CHARACTER:
		return "INVALID_PATTERN_CHARACTER"
	case INVALID_PATTERN:
		return "INVALID_PATTERN"
	case PATTERN_TOO_LONG:
		return "PATTERN_TOO_LONG"
	}

	panic("Encountered unknown Scanner errorType value")
}

type Error struct {
	Type errorType
	Char rune
	Pos  int
}

func (e Error) Error() string {
	return fmt.Sprintf("error scanning expression: %s scanning '%c' at %d", e.Type, e.Char, e.Pos)
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
	case INVALID_NUMERIC_LITERAL:
		return errors.ErrorMessage("Invalid numeric literal, numbers can range from 0 to 2^63 - 1:",
			expression,
			e.Pos,
			1)
	case UNTERMINATED_STRING:
		return errors.ErrorMessage("Unterminated string starting here:",
			expression,
			e.Pos,
			1)
	case UNTERMINATED_PATTERN:
		return errors.ErrorMessage("Unterminated pattern starting here:",
			expression,
			e.Pos,
			1)
	case INVALID_PATTERN_CHARACTER:
		return errors.ErrorMessage("Invalid pattern character, only ASCII and Latin-1 are allowed in patterns:",
			expression,
			e.Pos,
			1)
	case INVALID_PATTERN:
		return errors.ErrorMessage("Syntax error in pattern:",
			expression,
			e.Pos,
			1)
	case PATTERN_TOO_LONG:
		return errors.ErrorMessage(fmt.Sprintf("Pattern length exceeded, limit is %d characters:", MAX_PATTERN_LENGTH),
			expression,
			e.Pos,
			1)
	default:
		return fmt.Sprintf("Unknown error scanning expression; '%s'", expression)
	}
}

var keywords = map[string]tokens.TokenType{
	"starts-with": tokens.STARTS_WITH,
	"STARTS-WITH": tokens.STARTS_WITH,
	"matches":     tokens.MATCHES,
	"MATCHES":     tokens.MATCHES,
	"and":         tokens.AND,
	"AND":         tokens.AND,
	"or":          tokens.OR,
	"OR":          tokens.OR,
	"not":         tokens.NOT,
	"NOT":         tokens.NOT,
	"true":        tokens.TRUE,
	"TRUE":        tokens.TRUE,
	"false":       tokens.FALSE,
	"FALSE":       tokens.FALSE,
}

type scanner struct {
	source  []rune
	start   int
	current int
	tokens  []tokens.Token
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
func (s *scanner) Scan() ([]tokens.Token, error) {
	for !s.eof() {
		s.start = s.current
		err := s.scanToken()
		if err != nil {
			return nil, err
		}
	}

	s.tokens = append(s.tokens, tokens.Token{Type: tokens.EOF, Lexeme: "", Literal: nil, CharPos: s.current})

	toks := make([]tokens.Token, len(s.tokens))
	copy(toks, s.tokens)

	return toks, nil
}

// Scan a single token.
//
// Consumes characters and adds created tokens to the `tokens` list.
func (s *scanner) scanToken() error {
	c := s.advance()
	switch c {
	// grouping
	case '(':
		s.addToken(tokens.LEFT_PAREN)
	case ')':
		s.addToken(tokens.RIGHT_PAREN)
	// operators
	case '!':
		if s.match('=') {
			s.addToken(tokens.NOT_EQUAL)
		} else {
			s.addToken(tokens.NOT)
		}
	case '=':
		if err := s.equal(); err != nil {
			return err
		}
	case '>':
		if s.match('=') {
			s.addToken(tokens.GREATER_EQUAL)
		} else {
			s.addToken(tokens.GREATER)
		}
	case '<':
		if s.match('=') {
			s.addToken(tokens.LESS_EQUAL)
		} else {
			s.addToken(tokens.LESS)
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
	case '/':
		if err := s.pattern(); err != nil {
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
	s.addToken(tokens.EQUAL)

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

	s.addTokenLiteral(tokens.STRING, v)

	return nil
}

// Match a regular expression pattern token.
//
// Consumes all characters from the starting / to the terminating /. Adds a
// Token with the string content (excluding surrounding slashes) to the
// `tokens` list.
//
// Slash and backslach characters can be embedded by escaping with
// `\`: e.g. `\/`, `\\`
//
// Returns an error if the pattern is not terminated.
func (s *scanner) pattern() error {
	for s.peek() != '/' && !s.eof() {
		// Deal with escape characters
		c := s.peek()
		cnext := s.peekNext()
		if c == '\\' && isEscapablePatternChar(cnext) {
			s.advance()
		} else if c == '\\' && (cnext == 'u' || cnext == 'x') {
			return Error{Type: INVALID_PATTERN_CHARACTER, Char: c, Pos: s.current}
		} else if c > 0xff {
			return Error{Type: INVALID_PATTERN_CHARACTER, Char: c, Pos: s.current}
		}
		s.advance()
	}

	if s.eof() {
		return Error{Type: UNTERMINATED_PATTERN, Char: '/', Pos: s.start}
	}

	// Consume the slash, we only peek()ed above
	s.advance()
	// Note: we do not evaluate escaped backslashes in the pattern. The regexp
	// engine also interprets backslash escape sequences, so the only character
	// we deal with ourselves is the one we added, the pattern delimiter `/`
	v := string(s.source[s.start+1 : s.current-1])
	v = strings.ReplaceAll(v, "\\/", "/")

	if len(v) > MAX_PATTERN_LENGTH {
		return Error{Type: PATTERN_TOO_LONG, Char: '/', Pos: s.start}
	}

	pattern, err := regexp.Compile(v)
	if err != nil {
		return Error{Type: INVALID_PATTERN, Char: '/', Pos: s.start}
	}
	s.addTokenLiteral(tokens.PATTERN, pattern)

	return nil
}

// Returns true if `c` is a character that could be escaped.
//
// False otherwise.
func isEscapableChar(c rune) bool {
	return c == '"' || c == '\\'
}

func isEscapablePatternChar(c rune) bool {
	return c == '/'
}

func (s *scanner) number() error {
	for isDigit(s.peek()) {
		s.advance()
	}

	v, err := strconv.ParseInt(s.lexeme(), 10, 64)
	if err != nil {
		c := s.source[s.start]
		return Error{Type: INVALID_NUMERIC_LITERAL, Char: c, Pos: s.start}
	}

	s.addTokenLiteral(tokens.NUMBER, v)
	return nil
}

// Match an identifier.
//
// Consumes the token's characters and adds a Token to the `tokens` list.
func (s *scanner) identifier() {
	c := s.peek()
	if isIdentifierTail(c) || (c == '.' && isIdentifierTail(s.peekNext())) {
		s.advance()
		c = s.peek()

		for isIdentifierTail(c) || (c == '.' && isIdentifierTail(s.peekNext())) {
			s.advance()
			c = s.peek()
		}
	}

	identifier := s.lexeme()
	tokenType := keywords[identifier]
	if tokenType == tokens.NOT_FOUND {
		tokenType = tokens.IDENTIFIER
	}

	s.addTokenLiteral(tokenType, identifier)
}

// Add a token without a literal value to the `tokens` list.
func (s *scanner) addToken(t tokens.TokenType) {
	s.tokens = append(s.tokens, tokens.Token{
		Type:    t,
		Lexeme:  s.lexeme(),
		Literal: nil,
		CharPos: s.start,
	})
}

// Add a token with a literal value to the `tokens` list.
func (s *scanner) addTokenLiteral(t tokens.TokenType, literal any) {
	s.tokens = append(s.tokens, tokens.Token{
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
	return s.eofN(s.current)
}

// Returns true if pos is beyond the end of source.
// False otherwise.
func (s *scanner) eofN(pos int) bool {
	return pos >= len(s.source)
}

// Return the current lexeme.
func (s *scanner) lexeme() string {
	return string(s.source[s.start:s.current])
}
