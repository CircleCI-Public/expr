import type { Expr } from './expr.ts';
import type { Options } from './options.ts';
import { parse } from './parser.ts';
import { scan } from './scanner.ts';

/**
 * Scan and parse an expression.
 *
 * Throws ScanError or ParseError if the expression is invalid, or a RangeError
 * if it's too deeply nested, as parsing recurses for each level of nesting.
 */
export function compile(expression: string, options?: Options): Expr {
  const { tokens, error } = scan(expression, options);
  if (error !== undefined) {
    throw error;
  }
  return parse(tokens, options);
}
