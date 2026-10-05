import type { BinaryExpr, Expr, InfixExpr } from './expr.ts';
import { Builtin } from './builtin.ts';
import { ExprError } from './errors.ts';
import { type Token, TokenType } from './token.ts';
import {
  type Environment,
  Pattern,
  type Value,
  equal,
  isTruthy,
  toValue,
} from './val.ts';

export const InterpreterErrorType = {
  EXPECTED_NUMERIC_OPERAND: 'EXPECTED_NUMERIC_OPERAND',
  EXPECTED_STRING_OPERAND: 'EXPECTED_STRING_OPERAND',
  EXPECTED_PATTERN_OPERAND: 'EXPECTED_PATTERN_OPERAND',
  UNKNOWN_VARIABLE: 'UNKNOWN_VARIABLE',
} as const;

export type InterpreterErrorType =
  (typeof InterpreterErrorType)[keyof typeof InterpreterErrorType];

export class InterpreterError extends ExprError {
  readonly type: InterpreterErrorType;
  readonly token: Token;

  constructor(type: InterpreterErrorType, token: Token) {
    super(`${type}: <token ${token.type} ${token.lexeme}>[${token.charPos}]`);
    this.name = 'InterpreterError';
    this.type = type;
    this.token = token;
  }

  get from(): number {
    return this.token.charPos;
  }

  get to(): number {
    return this.token.charPos + this.token.lexeme.length;
  }

  describe(): string {
    const lexeme = this.token.lexeme;
    switch (this.type) {
      case InterpreterErrorType.EXPECTED_NUMERIC_OPERAND:
        return `Expected numeric operands to "${lexeme}" operator:`;
      case InterpreterErrorType.EXPECTED_STRING_OPERAND:
        return `Expected string operands to "${lexeme}" operator:`;
      case InterpreterErrorType.EXPECTED_PATTERN_OPERAND:
        return `Expected the right operand to "${lexeme}" operator to be a pattern:`;
      case InterpreterErrorType.UNKNOWN_VARIABLE:
        return `Referred to a variable "${lexeme}" that does not exist:`;
    }
  }
}

/**
 * Interpret an expression in an environment, returning the truthiness of its
 * value.
 *
 * Throws InterpreterError if the expression can't be interpreted, a TypeError
 * if it refers to a variable whose value has no expr equivalent (see
 * toValue), or a RangeError if it's too deeply nested, as evaluation recurses
 * for each level of nesting.
 */
export function interpret(expr: Expr, env: Environment): boolean {
  return isTruthy(evaluate(expr, env));
}

/**
 * Evaluate an expression in an environment, returning its value.
 *
 * Throws InterpreterError if the expression can't be evaluated, a TypeError
 * if it refers to a variable whose value has no expr equivalent (see
 * toValue), or a RangeError if it's too deeply nested, as evaluation recurses
 * for each level of nesting.
 */
export function evaluate(expr: Expr, env: Environment): Value {
  const lookup =
    env instanceof Map
      ? (name: string) => ({ found: env.has(name), value: env.get(name) })
      : (name: string) => ({
          found: Object.hasOwn(env, name),
          value: (env as Readonly<Record<string, unknown>>)[name],
        });

  const visit = (expr: Expr): Value => {
    switch (expr.kind) {
      case 'logical': {
        // Short-circuiting, returning the deciding operand's value.
        const left = visit(expr.left);
        if (
          expr.operator.type === TokenType.OR ? isTruthy(left) : !isTruthy(left)
        ) {
          return left;
        }
        return visit(expr.right);
      }
      case 'binary': {
        const left = visit(expr.left);
        const right = visit(expr.right);
        // Undefined values "infect" binary expressions: if either operand is
        // undefined the result is undefined, whatever the operator.
        if (left === undefined || right === undefined) {
          return undefined;
        }
        return binary(expr, left, right);
      }
      case 'infix': {
        const left = visit(expr.left);
        const right = visit(expr.right);
        if (left === undefined || right === undefined) {
          return undefined;
        }
        return infix(expr, left, right);
      }
      case 'unary':
        return !isTruthy(visit(expr.right));
      case 'literal':
        return expr.value;
      case 'identifier': {
        const { found, value } = lookup(expr.name.lexeme);
        if (!found) {
          throw new InterpreterError(
            InterpreterErrorType.UNKNOWN_VARIABLE,
            expr.name,
          );
        }
        return toValue(value);
      }
      case 'grouping':
        return visit(expr.expression);
    }
  };

  return visit(expr);
}

function binary(expr: BinaryExpr, left: Value, right: Value): Value {
  const type = expr.operator.type;
  if (type === TokenType.EQUAL) {
    return equal(left, right);
  }
  if (type === TokenType.NOT_EQUAL) {
    return !equal(left, right);
  }

  if (typeof left !== 'bigint' || typeof right !== 'bigint') {
    throw new InterpreterError(
      InterpreterErrorType.EXPECTED_NUMERIC_OPERAND,
      expr.operator,
    );
  }
  switch (type) {
    case TokenType.GREATER:
      return left > right;
    case TokenType.GREATER_EQUAL:
      return left >= right;
    case TokenType.LESS:
      return left < right;
    case TokenType.LESS_EQUAL:
      return left <= right;
  }
  return undefined;
}

function infix(expr: InfixExpr, left: Value, right: Value): Value {
  switch (expr.builtin) {
    case Builtin.CONTAINS:
    case Builtin.STARTS_WITH:
      if (typeof left !== 'string' || typeof right !== 'string') {
        throw new InterpreterError(
          InterpreterErrorType.EXPECTED_STRING_OPERAND,
          expr.operator,
        );
      }
      return expr.builtin === Builtin.CONTAINS
        ? left.includes(right)
        : left.startsWith(right);
    case Builtin.MATCHES:
      if (!(right instanceof Pattern)) {
        throw new InterpreterError(
          InterpreterErrorType.EXPECTED_PATTERN_OPERAND,
          expr.operator,
        );
      }
      if (typeof left !== 'string') {
        throw new InterpreterError(
          InterpreterErrorType.EXPECTED_STRING_OPERAND,
          expr.operator,
        );
      }
      return right.matches(left);
  }
}
