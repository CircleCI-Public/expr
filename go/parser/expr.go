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

type Visitor interface {
	VisitLogicalExpr(expr Logical) (any, error)
	VisitBinaryExpr(expr Binary) (any, error)
	VisitUnaryExpr(expr Unary) (any, error)
	VisitLiteralExpr(expr Literal) (any, error)
	VisitIdentifierExpr(expr Identifier) (any, error)
	VisitGroupingExpr(expr Grouping) (any, error)
}

// Represents expressions that together form an abstract syntax tree.
//
// Exprs are simple data types with public members. They provide an `accept`
// method which cooperates with the `Visitor` interface to facilitate walking
// the AST.
type Expr interface {
	Accept(visitor Visitor) (any, error)
}

// A logical `and` / `or` node in the AST.
type Logical struct {
	Left     Expr
	Operator token.Token
	Right    Expr
}

func (l Logical) Accept(visitor Visitor) (any, error) {
	return visitor.VisitLogicalExpr(l)
}

// A binary operator node in the AST.
type Binary struct {
	Left     Expr
	Operator token.Token
	Right    Expr
}

func (b Binary) Accept(visitor Visitor) (any, error) {
	return visitor.VisitBinaryExpr(b)
}

// A unary operator node in the AST.
type Unary struct {
	Operator token.Token
	Right    Expr
}

func (u Unary) Accept(visitor Visitor) (any, error) {
	return visitor.VisitUnaryExpr(u)
}

// A literal value node in the AST.
//
// Literal nodes are leaf nodes.
type Literal struct {
	Value any
}

func (l Literal) Accept(visitor Visitor) (any, error) {
	return visitor.VisitLiteralExpr(l)
}

// An identifier node in the AST.
//
// Identifier nodes are leaf nodes.
type Identifier struct {
	Name token.Token
}

func (i Identifier) Accept(visitor Visitor) (any, error) {
	return visitor.VisitIdentifierExpr(i)
}

// A grouping node in the AST.
//
// Grouping wraps any other expression node.
type Grouping struct {
	Expression Expr
}

func (g Grouping) Accept(visitor Visitor) (any, error) {
	return visitor.VisitGroupingExpr(g)
}
