import type { Builtin } from './builtin.ts';
import type { Literal, Token } from './token.ts';

/** A node in the abstract syntax tree of an expression. */
export type Expr =
  | LogicalExpr
  | BinaryExpr
  | InfixExpr
  | UnaryExpr
  | LiteralExpr
  | IdentifierExpr
  | GroupingExpr;

/** A logical `and` / `or` node. */
export interface LogicalExpr {
  readonly kind: 'logical';
  readonly left: Expr;
  readonly operator: Token;
  readonly right: Expr;
}

/** A binary operator node. */
export interface BinaryExpr {
  readonly kind: 'binary';
  readonly left: Expr;
  readonly operator: Token;
  readonly right: Expr;
}

/** An infix builtin function node. */
export interface InfixExpr {
  readonly kind: 'infix';
  readonly left: Expr;
  readonly operator: Token;
  readonly builtin: Builtin;
  readonly right: Expr;
}

/** A unary operator node. */
export interface UnaryExpr {
  readonly kind: 'unary';
  readonly operator: Token;
  readonly right: Expr;
}

/** A literal value, a leaf node. */
export interface LiteralExpr {
  readonly kind: 'literal';
  readonly value: boolean | Literal;
}

/** A variable reference, a leaf node. */
export interface IdentifierExpr {
  readonly kind: 'identifier';
  readonly name: Token;
}

/** A parenthesised expression. */
export interface GroupingExpr {
  readonly kind: 'grouping';
  readonly expression: Expr;
}
