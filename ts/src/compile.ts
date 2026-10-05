import type { Expr } from './expr.ts';
import { parse } from './parser.ts';
import { scan } from './scanner.ts';

/**
 * Scan and parse an expression.
 *
 * Throws ScanError or ParseError if the expression is invalid.
 */
export function compile(expression: string): Expr {
  const { tokens, error } = scan(expression);
  if (error !== undefined) {
    throw error;
  }
  return parse(tokens);
}
