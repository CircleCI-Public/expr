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

package parser

import "github.com/circleci/expr/go/token"

type Visitor[T any] interface {
	VisitLogicalExpr(expr Logical) (T, error)
	VisitBinaryExpr(expr Binary) (T, error)
	VisitUnaryExpr(expr Unary) (T, error)
	VisitLiteralExpr(expr Literal) (T, error)
	VisitIdentifierExpr(expr Identifier) (T, error)
	VisitGroupingExpr(expr Grouping) (T, error)
}

// Represents expressions that together form an abstract syntax tree.
//
// Exprs are simple data types with public members.
type Expr interface {
	isExpr()
}

// Visitable is a parameterised "facilitator" type used to walk an Expr AST
// with a Visitor.
// Implementers of Visitor should create a Visitable with the correct concrete
// type for the Visitor's return value, wrapping the Expr they need to visit,
// then call the Visitable's Accept method.
type Visitable[T any] struct {
	Expression Expr
}

func (v *Visitable[T]) Accept(visitor Visitor[T]) (T, error) {
	switch expr := v.Expression.(type) {
	case Logical:
		return visitor.VisitLogicalExpr(expr)
	case Binary:
		return visitor.VisitBinaryExpr(expr)
	case Unary:
		return visitor.VisitUnaryExpr(expr)
	case Literal:
		return visitor.VisitLiteralExpr(expr)
	case Identifier:
		return visitor.VisitIdentifierExpr(expr)
	case Grouping:
		return visitor.VisitGroupingExpr(expr)
	default:
		panic("non-exhaustive switch")
	}
}

// A logical `and` / `or` node in the AST.
type Logical struct {
	Left     Expr
	Operator token.Token
	Right    Expr
}

func (l Logical) isExpr() {}

// A binary operator node in the AST.
type Binary struct {
	Left     Expr
	Operator token.Token
	Right    Expr
}

func (b Binary) isExpr() {}

// A unary operator node in the AST.
type Unary struct {
	Operator token.Token
	Right    Expr
}

func (u Unary) isExpr() {}

// A literal value node in the AST.
//
// Literal nodes are leaf nodes.
type Literal struct {
	Value any
}

func (l Literal) isExpr() {}

// An identifier node in the AST.
//
// Identifier nodes are leaf nodes.
type Identifier struct {
	Name token.Token
}

func (i Identifier) isExpr() {}

// A grouping node in the AST.
//
// Grouping wraps any other expression node.
type Grouping struct {
	Expression Expr
}

func (g Grouping) isExpr() {}
