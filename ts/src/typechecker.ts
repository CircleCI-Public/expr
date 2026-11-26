import type { BinaryExpr, Expr, InfixExpr } from './expr.ts';
import { Builtin, builtinForLexeme } from './builtin.ts';
import { ExprError } from './errors.ts';
import { type Token, TokenType } from './token.ts';
import { Pattern } from './val.ts';

export const TypeCheckErrorType = {
  EXPECTED_NUMERIC_OPERAND: 'EXPECTED_NUMERIC_OPERAND',
  EXPECTED_STRING_OPERAND: 'EXPECTED_STRING_OPERAND',
  EXPECTED_PATTERN_OPERAND: 'EXPECTED_PATTERN_OPERAND',
  UNKNOWN_VARIABLE: 'UNKNOWN_VARIABLE',
} as const;

export type TypeCheckErrorType =
  (typeof TypeCheckErrorType)[keyof typeof TypeCheckErrorType];

/** An operator given operands of the wrong type, or a variable with no known type. */
export class TypeCheckError extends ExprError {
  readonly type: TypeCheckErrorType;
  /** The operator, or for UNKNOWN_VARIABLE the identifier. */
  readonly token: Token;
  /**
   * The wrong-typed operand as the message names it: its variable name, or
   * "the left operand" or "the right operand". Undefined for UNKNOWN_VARIABLE.
   */
  readonly operand: string | undefined;
  /** The wrong-typed operand's type. Undefined for UNKNOWN_VARIABLE. */
  readonly operandType: string | undefined;

  constructor(
    type: TypeCheckErrorType,
    token: Token,
    operand?: string,
    operandType?: string,
  ) {
    super(`${type}: <token ${token.type} ${token.lexeme}>[${token.charPos}]`);
    this.name = 'TypeCheckError';
    this.type = type;
    this.token = token;
    this.operand = operand;
    this.operandType = operandType;
  }

  get from(): number {
    return this.token.charPos;
  }

  get to(): number {
    return this.token.charPos + this.token.lexeme.length;
  }

  describe(): string {
    const lexeme = this.token.lexeme;
    const actual = `but ${this.operand} is a ${this.operandType}:`;
    switch (this.type) {
      case TypeCheckErrorType.EXPECTED_NUMERIC_OPERAND:
        return `"${lexeme}" needs numbers, ${actual}`;
      case TypeCheckErrorType.EXPECTED_STRING_OPERAND:
        return builtinForLexeme(lexeme) === Builtin.MATCHES
          ? `"${lexeme}" needs a string on the left, ${actual}`
          : `"${lexeme}" needs strings, ${actual}`;
      case TypeCheckErrorType.EXPECTED_PATTERN_OPERAND:
        return `"${lexeme}" needs a pattern on the right, ${actual}`;
      case TypeCheckErrorType.UNKNOWN_VARIABLE:
        return `Referred to a variable "${lexeme}" that has no known type:`;
    }
  }
}

type Type = 'string' | 'boolean' | 'number' | 'pattern' | undefined;

/** Each variable's type: "string", "boolean", "number" or "pattern". */
export type VariableTypes =
  | ReadonlyMap<string, string>
  | Readonly<Record<string, string>>;

/**
 * Check that every operator's operands have types the operator accepts, given
 * each variable's type. An operand of unknown type passes. Checks operators
 * that short-circuiting would skip, and throws for the first failure in
 * evaluation order.
 *
 * Throws TypeCheckError, or a RangeError if the expression is too deeply
 * nested, as checking recurses for each level of nesting.
 */
export function checkTypes(expr: Expr, types: VariableTypes): void {
  const require = (
    expr: BinaryExpr | InfixExpr,
    side: 'left' | 'right',
    t: Type,
    want: Type,
    err: TypeCheckErrorType,
  ) => {
    if (t !== undefined && t !== want) {
      const operand = expr[side];
      const name =
        operand.kind === 'identifier'
          ? operand.name.lexeme
          : `the ${side} operand`;
      throw new TypeCheckError(err, expr.operator, name, t);
    }
  };

  const visit = (expr: Expr): Type => {
    switch (expr.kind) {
      case 'logical': {
        const left = visit(expr.left);
        return left === visit(expr.right) ? left : undefined;
      }
      case 'binary': {
        const left = visit(expr.left);
        const right = visit(expr.right);
        const t = expr.operator.type;
        if (t !== TokenType.EQUAL && t !== TokenType.NOT_EQUAL) {
          const e = TypeCheckErrorType.EXPECTED_NUMERIC_OPERAND;
          require(expr, 'left', left, 'number', e);
          require(expr, 'right', right, 'number', e);
        }
        return 'boolean';
      }
      case 'infix': {
        const left = visit(expr.left);
        const right = visit(expr.right);
        const s = TypeCheckErrorType.EXPECTED_STRING_OPERAND;
        if (expr.builtin === Builtin.MATCHES) {
          require(expr, 'right', right, 'pattern', TypeCheckErrorType.EXPECTED_PATTERN_OPERAND);
          require(expr, 'left', left, 'string', s);
        } else {
          require(expr, 'left', left, 'string', s);
          require(expr, 'right', right, 'string', s);
        }
        return 'boolean';
      }
      case 'unary':
        visit(expr.right);
        return 'boolean';
      case 'literal':
        switch (typeof expr.value) {
          case 'boolean':
            return 'boolean';
          case 'string':
            return 'string';
          case 'bigint':
            return 'number';
        }
        return expr.value instanceof Pattern ? 'pattern' : undefined;
      case 'identifier': {
        const name = expr.name.lexeme;
        const t =
          types instanceof Map
            ? types.get(name)
            : Object.hasOwn(types, name)
              ? (types as Readonly<Record<string, string>>)[name]
              : undefined;
        if (
          t !== 'string' &&
          t !== 'boolean' &&
          t !== 'number' &&
          t !== 'pattern'
        ) {
          throw new TypeCheckError(
            TypeCheckErrorType.UNKNOWN_VARIABLE,
            expr.name,
          );
        }
        return t;
      }
      case 'grouping':
        return visit(expr.expression);
    }
  };

  visit(expr);
}
