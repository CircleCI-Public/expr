import type { Expr } from './expr.ts';
import type { Token } from './token.ts';

/**
 * Gather the variables an expression refers to, in the order they appear.
 *
 * Returns the IDENTIFIER tokens, which have the variable name and position.
 */
export function gatherVariables(expr: Expr): Token[] {
  const variables: Token[] = [];
  // Walk iteratively: long chains of `and`/`or` make deep trees.
  const stack: Expr[] = [expr];
  for (let e = stack.pop(); e !== undefined; e = stack.pop()) {
    switch (e.kind) {
      case 'logical':
      case 'binary':
      case 'infix':
        stack.push(e.right, e.left);
        break;
      case 'unary':
        stack.push(e.right);
        break;
      case 'grouping':
        stack.push(e.expression);
        break;
      case 'identifier':
        variables.push(e.name);
        break;
      case 'literal':
        break;
    }
  }
  return variables;
}
