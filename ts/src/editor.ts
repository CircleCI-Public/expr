/*
 * Helpers for editing expressions, e.g. in CodeMirror.
 *
 * The results have the shapes CodeMirror's lint and autocomplete packages
 * use, so they can be returned directly from a lint or completion source,
 * without this package depending on CodeMirror.
 */

import { gatherVariables } from './analyser.ts';
import type { Expr } from './expr.ts';
import { compile } from './compile.ts';
import { ExprError } from './errors.ts';
import { ParseError } from './parser.ts';
import { ScanError, ScanErrorType, scan } from './scanner.ts';
import { TokenType } from './token.ts';

/** A variable that can be used in expressions. */
export interface Variable {
  /**
   * The variable name. A name ending `.*`, e.g. `pipeline.parameters.*`,
   * stands for any name with that prefix.
   */
  readonly name: string;
  /** A short description, e.g. the type, shown next to completions. */
  readonly detail?: string;
  /** Documentation shown alongside completions. */
  readonly info?: string;
  /**
   * If set, the variable is deprecated: using it produces a warning with this
   * message, and it's ranked below other completions.
   */
  readonly deprecated?: string;
  /**
   * If set, the variable will be deprecated: using it produces an info
   * diagnostic with this message.
   */
  readonly pendingDeprecation?: string;
  /** A hidden variable is valid but isn't offered as a completion. */
  readonly hidden?: boolean;
}

export interface Diagnostic {
  readonly from: number;
  readonly to: number;
  readonly severity: 'error' | 'warning' | 'info';
  readonly message: string;
  readonly source: 'expr';
}

export interface DiagnoseOptions {
  /**
   * The variables the expression can refer to. If given, references to any
   * other variable are errors. Names can also be strings.
   */
  readonly variables?: Iterable<Variable | string>;
}

/**
 * Check an expression for errors.
 *
 * Reports the first scan or parse error, or if the expression is valid, any
 * references to unknown or deprecated variables. An empty expression is
 * reported as an error, as the interpreter rejects it; a consumer that allows
 * empty input should check for that before calling diagnose.
 *
 * Never throws for any expression: one too deeply nested to parse is reported
 * as an error.
 */
export function diagnose(
  expression: string,
  options: DiagnoseOptions = {},
): Diagnostic[] {
  let expr: Expr;
  try {
    expr = compile(expression);
  } catch (error) {
    if (error instanceof ExprError) {
      return [errorDiagnostic(error, expression)];
    }
    // The parser recurses for each level of nesting.
    if (error instanceof RangeError) {
      return [
        {
          from: 0,
          to: expression.length,
          severity: 'error',
          message: 'Expression is too deeply nested',
          source: 'expr',
        },
      ];
    }
    throw error;
  }

  if (options.variables === undefined) {
    return [];
  }
  const variables = new Variables(options.variables);

  const diagnostics: Diagnostic[] = [];
  for (const token of gatherVariables(expr)) {
    const variable = variables.lookup(token.lexeme);
    const range = {
      from: token.charPos,
      to: token.charPos + token.lexeme.length,
      source: 'expr' as const,
    };
    if (variable === undefined) {
      diagnostics.push({
        ...range,
        severity: 'error',
        message: `Unknown variable "${token.lexeme}"`,
      });
    } else if (variable.deprecated !== undefined) {
      diagnostics.push({
        ...range,
        severity: 'warning',
        message: variable.deprecated,
      });
    } else if (variable.pendingDeprecation !== undefined) {
      diagnostics.push({
        ...range,
        severity: 'info',
        message: variable.pendingDeprecation,
      });
    }
  }
  return diagnostics;
}

function errorDiagnostic(error: ExprError, expression: string): Diagnostic {
  const clamp = (n: number): number => Math.min(n, expression.length);
  let to = error.to;
  // Highlight the whole of an unterminated string or pattern.
  if (
    error instanceof ScanError &&
    (error.type === ScanErrorType.UNTERMINATED_STRING ||
      error.type === ScanErrorType.UNTERMINATED_PATTERN)
  ) {
    to = expression.length;
  }
  return {
    from: clamp(error.from),
    to: clamp(to),
    severity: 'error',
    message: diagnosticMessage(error, expression),
    source: 'expr',
  };
}

/**
 * The error's description, without the trailing colon, and saying "end of
 * expression" rather than showing an empty or NUL lexeme at the end.
 */
function diagnosticMessage(error: ExprError, expression: string): string {
  const atEnd = error.from >= expression.length;
  if (
    atEnd &&
    error instanceof ScanError &&
    error.type === ScanErrorType.INCOMPLETE_EQUALS
  ) {
    return 'Incomplete token, expected "==", found end of expression';
  }
  const message = error.describe().replace(/:$/, '');
  if (
    atEnd &&
    error instanceof ParseError &&
    error.token.type === TokenType.EOF
  ) {
    return message.replace(/found ""$/, 'found end of expression');
  }
  return message;
}

export interface Completion {
  readonly label: string;
  readonly type: 'variable' | 'namespace' | 'keyword' | 'function';
  readonly detail?: string;
  readonly info?: string;
  /** Ranks the completion above (positive) or below (negative) others. */
  readonly boost?: number;
}

export interface CompletionResult {
  /** The start of the text the completion replaces. */
  readonly from: number;
  /** The end of the text the completion replaces, the cursor position. */
  readonly to: number;
  readonly options: Completion[];
  /** Matches text the options remain valid for as the user types. */
  readonly validFor: RegExp;
}

export interface CompleteOptions {
  /** The variables to offer where an operand is expected. */
  readonly variables?: Iterable<Variable | string>;
  /**
   * Whether completion was explicitly requested. If not, completions are
   * only offered when there's a partial word or operator before the cursor,
   * or where an operator is expected after a space.
   */
  readonly explicit?: boolean;
}

const operandKeywords: readonly Completion[] = [
  { label: 'true', type: 'keyword' },
  { label: 'false', type: 'keyword' },
  { label: 'not', type: 'keyword', detail: 'negation' },
];

const operators: readonly Completion[] = [
  { label: '==', type: 'keyword', detail: 'equal to' },
  { label: '!=', type: 'keyword', detail: 'not equal to' },
  { label: '<', type: 'keyword', detail: 'less than' },
  { label: '<=', type: 'keyword', detail: 'less than or equal to' },
  { label: '>', type: 'keyword', detail: 'greater than' },
  { label: '>=', type: 'keyword', detail: 'greater than or equal to' },
  { label: 'and', type: 'keyword' },
  { label: 'or', type: 'keyword' },
  {
    label: 'contains',
    type: 'function',
    detail: 'string contains substring',
  },
  {
    label: 'starts-with',
    type: 'function',
    detail: 'string starts with prefix',
  },
  {
    label: 'matches',
    type: 'function',
    detail: 'string matches /pattern/',
  },
];

// Tokens after which an operand is expected.
const operandFollows = new Set<string>([
  TokenType.LEFT_PAREN,
  TokenType.NOT,
  TokenType.AND,
  TokenType.OR,
  TokenType.EQUAL,
  TokenType.NOT_EQUAL,
  TokenType.GREATER,
  TokenType.GREATER_EQUAL,
  TokenType.LESS,
  TokenType.LESS_EQUAL,
  TokenType.BUILTIN,
]);

const partialWord = /[A-Za-z][\w?-]*(?:\.[\w?-]*)*$/;
const partialOperator = /[=!<>]=?$/;

const validForOperand = /^[A-Za-z][\w?.-]*$/;
const validForOperator = /^(?:[A-Za-z][\w?.-]*|[=!<>]=?)$/;

/**
 * Complete the word or operator before position `pos` in an expression.
 *
 * Offers variables and literals where an operand is expected, and comparison
 * and logical operators and builtin functions after an operand, including
 * after an operand and a space with nothing typed yet. Returns null when
 * there's nothing to complete, e.g. inside a string, or the context can't be
 * known because of an error earlier in the expression.
 */
export function complete(
  expression: string,
  pos: number,
  options: CompleteOptions = {},
): CompletionResult | null {
  const before = expression.slice(0, pos);
  const word = partialWord.exec(before);
  const operator = word === null ? partialOperator.exec(before) : null;
  const typed = word?.[0] ?? operator?.[0] ?? '';
  const explicit = options.explicit === true;
  // After a space, an operator is likely next, so it's worth offering them
  // unasked; an operand could be anything, e.g. a string.
  if (typed === '' && !explicit && !/\s$/.test(before)) {
    return null;
  }
  const from = pos - typed.length;

  const { tokens, error } = scan(expression.slice(0, from));
  if (error !== undefined) {
    return null;
  }
  const last = tokens.filter((t) => t.type !== TokenType.EOF).at(-1);

  if (last === undefined || operandFollows.has(last.type)) {
    if (operator !== null || (typed === '' && !explicit)) {
      return null;
    }
    const variables = [...new Variables(options.variables ?? []).visible()];
    return {
      from,
      to: pos,
      options: [...variables.map(variableCompletion), ...operandKeywords],
      validFor: validForOperand,
    };
  }

  return {
    from,
    to: pos,
    options: [...operators],
    validFor: validForOperator,
  };
}

function variableCompletion(v: Variable): Completion {
  const wildcard = v.name.endsWith('.*');
  const note = v.deprecated ?? v.pendingDeprecation;
  const detail =
    v.deprecated === undefined
      ? v.detail
      : [v.detail, '(deprecated)'].filter(Boolean).join(' ');
  const info =
    note === undefined ? v.info : [note, v.info].filter(Boolean).join('\n\n');
  return {
    label: wildcard ? v.name.slice(0, -1) : v.name,
    type: wildcard ? 'namespace' : 'variable',
    ...(detail === undefined ? {} : { detail }),
    ...(info === undefined ? {} : { info }),
    ...(v.deprecated === undefined ? {} : { boost: -50 }),
  };
}

class Variables {
  private readonly exact = new Map<string, Variable>();
  private readonly prefixes: Variable[] = [];

  constructor(variables: Iterable<Variable | string>) {
    for (const v of variables) {
      const variable = typeof v === 'string' ? { name: v } : v;
      if (variable.name.endsWith('.*')) {
        this.prefixes.push(variable);
      } else {
        this.exact.set(variable.name, variable);
      }
    }
  }

  lookup(name: string): Variable | undefined {
    return (
      this.exact.get(name) ??
      this.prefixes.find(
        (v) =>
          name.startsWith(v.name.slice(0, -1)) &&
          name.length > v.name.length - 1,
      )
    );
  }

  *visible(): Iterable<Variable> {
    for (const v of [...this.exact.values(), ...this.prefixes]) {
      if (v.hidden !== true) {
        yield v;
      }
    }
  }
}
