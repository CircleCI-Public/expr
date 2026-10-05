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

package parsers

/* Grammar:
 * expression -> logic_or
 * logic_or -> logic_and ( "or" logic_and )*;
 * logic_and -> equality ( "and" equality )*;
 * equality -> comparison ( ( "==" | "!=" | BUILTIN ) comparison )*;
 * comparison -> unary ( ( ">=" | ">" | "<=" | "<" ) unary)*;
 * unary -> "not" unary | primary;
 * primary -> "true" | "false" | NUMBER | STRING | IDENTIFIER | PATTERN | "(" expression ")"
 */

import (
	"fmt"
	"unicode/utf8"

	"github.com/CircleCI-Public/expr/builtins"
	"github.com/CircleCI-Public/expr/errors"
	"github.com/CircleCI-Public/expr/tokens"
)

type errorType string

const (
	UNEXPECTED_ADDITIONAL_INPUT errorType = "Unexpected additional input."
	EXPECTED_EXPRESSION         errorType = "Expected expression."
	EXPECTED_RIGHT_PAREN        errorType = "Expected ')' after expression."
	UNKNOWN_BUILTIN_FUNCTION    errorType = "Unknown infix function."
)

func (et errorType) Symbol() string {
	switch et {
	case UNEXPECTED_ADDITIONAL_INPUT:
		return "UNEXPECTED_ADDITIONAL_INPUT"
	case EXPECTED_EXPRESSION:
		return "EXPECTED_EXPRESSION"
	case EXPECTED_RIGHT_PAREN:
		return "EXPECTED_RIGHT_PAREN"
	case UNKNOWN_BUILTIN_FUNCTION:
		return "UNKNOWN_BUILTIN_FUNCTION"
	}

	panic("Encountered unknown Parser errorType value")
}

type Error struct {
	Type  errorType
	Token tokens.Token
}

func (e Error) Error() string {
	return fmt.Sprintf("error parsing: %s at %s", e.Type.Symbol(), e.Token.String())
}

// Return a multiline error message describing the error.
//
// Returns a string of the form:
//
// explanation
// line of expression containing the error
// marker line pointing at the error token
//
// E.g.
// "Expected expression, found \")\":\n" +
// "foo and ) bar\n" +
// "        ^"
func (e Error) AsErrorMessage(expression string) string {
	errorString := e.Token.Lexeme

	switch e.Type {
	case UNEXPECTED_ADDITIONAL_INPUT:
		return errors.ErrorMessage(fmt.Sprintf("Unexpected additional input, found \"%s\", expected EOF:", errorString),
			expression,
			e.Token.CharPos,
			utf8.RuneCountInString(errorString))

	case EXPECTED_EXPRESSION:
		return errors.ErrorMessage(fmt.Sprintf("Expected expression, found \"%s\":", errorString),
			expression,
			e.Token.CharPos,
			utf8.RuneCountInString(errorString))

	case EXPECTED_RIGHT_PAREN:
		return errors.ErrorMessage("Expected ')' after expression:",
			expression,
			e.Token.CharPos,
			1)

	case UNKNOWN_BUILTIN_FUNCTION:
		return errors.ErrorMessage("Unknown infix function:",
			expression,
			e.Token.CharPos,
			utf8.RuneCountInString(errorString))

	default:
		return fmt.Sprintf("Unknown error parsing expression: '%s'", expression)
	}
}

type parser struct {
	tokens  []tokens.Token
	current int
}

// Create a parser to parse the given list of Tokens.
func New(toks []tokens.Token) *parser {
	return &parser{
		tokens:  toks,
		current: 0,
	}
}

// Parse the `tokens` list into an abstract syntax tree.
//
// Returns the root node of the AST.
func (p *parser) Parse() (Expr, error) {
	expr, err := p.expression()
	if err != nil {
		return nil, err
	}

	if !p.eof() {
		if err := p.checkUnrecognisedFunction(); err != nil {
			return nil, err
		}
		return nil, Error{Type: UNEXPECTED_ADDITIONAL_INPUT, Token: p.peek()}
	}

	return expr, nil
}

// Throw ParseError if remaining tokens match the shape of a builtin function
// call.
//
// If an invalid name is used for a builtin function the stream of tokens
// will look like:
// [...tokens that parse as `comparison`..., IDENT, ...more tokens...]
//
// The tokens to the left of IDENT can terminate an expression. IDENT and the
// following tokens (if any) are extra input.
//
// If the tokens following IDENT also parse as a `comparison` then it's very
// likely that IDENT is a misspelled builtin function name.
func (p *parser) checkUnrecognisedFunction() error {
	pos := p.save()

	// Might be an unrecognised function name
	if p.match(tokens.IDENTIFIER) {
		potentialBuiltin := p.previous()

		_, err := p.comparison()
		if err == nil {
			// If the rest of the token stream parses as a valid operand then we
			// probably have an invalid function name in potentialBuiltin.
			return Error{Type: UNKNOWN_BUILTIN_FUNCTION, Token: potentialBuiltin}
		}
	}

	// If the rest of the token stream doesn't parse as a valid operand
	// then we can't assume anything, undo any token consumption
	p.restore(pos)
	return nil
}

// Match an 'expression' production.
func (p *parser) expression() (Expr, error) {
	expr, err := p.or()
	if err != nil {
		return nil, err
	}

	return expr, nil
}

// Match an 'or' production.
func (p *parser) or() (Expr, error) {
	expr, err := p.and()
	if err != nil {
		return nil, err
	}

	for p.match(tokens.OR) {
		operator := p.previous()
		right, err := p.and()
		if err != nil {
			return nil, err
		}
		expr = Logical{Left: expr, Operator: operator, Right: right}
	}

	return expr, nil
}

// Match an 'and' production.
func (p *parser) and() (Expr, error) {
	expr, err := p.equality()
	if err != nil {
		return nil, err
	}

	for p.match(tokens.AND) {
		operator := p.previous()
		right, err := p.equality()
		if err != nil {
			return nil, err
		}
		expr = Logical{Left: expr, Operator: operator, Right: right}
	}

	return expr, nil
}

// Match an 'equality' production.
func (p *parser) equality() (Expr, error) {
	expr, err := p.comparison()
	if err != nil {
		return nil, err
	}

	for p.match(tokens.EQUAL, tokens.NOT_EQUAL, tokens.BUILTIN) {
		operator := p.previous()

		if operator.Type == tokens.BUILTIN {
			builtin := builtins.ForLexeme(operator.Lexeme)
			right, err := p.comparison()
			if err != nil {
				return nil, err
			}
			expr = Infix{Left: expr, Operator: operator, Builtin: builtin, Right: right}
		} else {
			right, err := p.comparison()
			if err != nil {
				return nil, err
			}
			expr = Binary{Left: expr, Operator: operator, Right: right}
		}
	}

	return expr, nil
}

// Match a 'comparison' production.
func (p *parser) comparison() (Expr, error) {
	expr, err := p.unary()
	if err != nil {
		return nil, err
	}

	for p.match(tokens.GREATER, tokens.GREATER_EQUAL, tokens.LESS, tokens.LESS_EQUAL) {
		operator := p.previous()
		right, err := p.unary()
		if err != nil {
			return nil, err
		}
		expr = Binary{Left: expr, Operator: operator, Right: right}
	}

	return expr, nil
}

// Match a 'unary' production.
func (p *parser) unary() (Expr, error) {
	if p.match(tokens.NOT) {
		operator := p.previous()
		right, err := p.unary()
		if err != nil {
			return nil, err
		}
		return Unary{Operator: operator, Right: right}, nil
	}

	expr, err := p.primary()
	if err != nil {
		return nil, err
	}

	return expr, nil
}

// Match a 'primary' production.
func (p *parser) primary() (Expr, error) {
	if p.match(tokens.FALSE) {
		return Literal{Value: false}, nil
	}
	if p.match(tokens.TRUE) {
		return Literal{Value: true}, nil
	}

	if p.match(tokens.NUMBER, tokens.STRING, tokens.PATTERN) {
		return Literal{Value: p.previous().Literal}, nil
	}

	if p.match(tokens.IDENTIFIER) {
		return Identifier{Name: p.previous()}, nil
	}

	if p.match(tokens.LEFT_PAREN) {
		expr, err := p.expression()
		if err != nil {
			return nil, err
		}
		_, err = p.consume(tokens.RIGHT_PAREN, EXPECTED_RIGHT_PAREN)
		if err != nil {
			return nil, err
		}
		return Grouping{Expression: expr}, nil
	}

	return nil, Error{Type: EXPECTED_EXPRESSION, Token: p.peek()}
}

// Attempt to match a token of one of the input types.
//
// Consumes the token and returns true if there is a match. Returns false
// without consuming any tokens otherwise.
func (p *parser) match(types ...tokens.TokenType) bool {
	for _, t := range types {
		if p.check(t) {
			p.advance()
			return true
		}
	}

	return false
}

// Consume a token of the expected type.
//
// Returns the token if the next token in the input is of the expected type.
//
// Returns an Error if the next token does not match the expected type.
func (p *parser) consume(expected tokens.TokenType, et errorType) (tokens.Token, error) {
	if p.check(expected) {
		return p.advance(), nil
	}

	return tokens.Token{}, Error{Type: et, Token: p.peek()}
}

// Check if the next token is of the expected type.
//
// Returns true if the type is as expected, false otherwise.
func (p *parser) check(t tokens.TokenType) bool {
	if p.eof() {
		return false
	}

	return p.peek().Type == t
}

// Returns the token one before the current position in the input token list.
func (p *parser) previous() tokens.Token {
	return p.tokens[p.current-1]
}

// Advance the pointer in the token list.
//
// Returns the token that the pointer was pointing to.
func (p *parser) advance() tokens.Token {
	if !p.eof() {
		p.current += 1
	}

	return p.previous()
}

// Restore the position in the token stream to one returned by save().
func (p *parser) restore(pos int) {
	if pos >= 0 && pos < len(p.tokens) {
		p.current = pos
	}
}

// Returns the current position in the token stream. Can revert to this
// position in the stream with restore().
func (p *parser) save() int {
	return p.current
}

// Returns true if the next token that would be returned by 'advance' is EOF.
// False otherwise.
func (p *parser) eof() bool {
	return p.peek().Type == tokens.EOF
}

// Return the next token in the input token list without advancing.
func (p *parser) peek() tokens.Token {
	return p.tokens[p.current]
}
