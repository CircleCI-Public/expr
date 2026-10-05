// Property tests: scanning, parsing, analysing, interpreting and the editor
// helpers either succeed or fail with one of expr's errors, for any
// expression. FUZZ_RUNS sets the number of expressions to try; e.g.
//
//   FUZZ_RUNS=1000000 node --test src/fuzz.test.ts

import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import fc from 'fast-check';

import {
  expression,
  testCase,
  toEnvironment,
} from '../scripts/fuzz/arbitraries.ts';
import {
  ExprError,
  compile,
  complete,
  diagnose,
  evaluate,
  gatherVariables,
  interpret,
  isTruthy,
} from './index.ts';

const numRuns = Number(process.env['FUZZ_RUNS'] ?? 2000);

/**
 * Checks that an error message shows the line of the expression containing
 * the error, with a marker under it.
 */
function checkError(expression: string, error: ExprError): void {
  assert.ok(
    // An error at EOF can mark one character past the end.
    error.from >= 0 &&
      error.from <= error.to &&
      error.from <= expression.length,
    `error at ${error.from}-${error.to} outside expression of length ${expression.length}`,
  );
  const message = error.asErrorMessage(expression);

  const pos = error.from;
  if (expression[pos] === '\n') {
    // An error on a newline shows the rest of the expression.
    return;
  }
  const lineStart = expression.lastIndexOf('\n', pos - 1) + 1;
  const lineEnd = expression.indexOf('\n', pos);
  // The preamble can quote a lexeme with newlines in it, so check the end of
  // the message. The marker is as long as the token, which is empty at EOF.
  const want = `\n${expression.slice(lineStart, lineEnd < 0 ? undefined : lineEnd)}\n${' '.repeat(pos - lineStart)}`;
  assert.ok(
    message.replace(/\^+$/, '').endsWith(want),
    `message:\n${message}\nwant it to end with:\n${want}`,
  );
}

describe('fuzz', () => {
  it('compiles and evaluates any expression, or throws an ExprError', () => {
    fc.assert(
      fc.property(testCase, ({ expression, environment }) => {
        const env = toEnvironment(environment);
        let expr;
        try {
          expr = compile(expression);
        } catch (e) {
          assert.ok(e instanceof ExprError, `unexpected ${String(e)}`);
          checkError(expression, e);
          return;
        }
        for (const v of gatherVariables(expr)) {
          assert.equal(
            expression.slice(v.charPos, v.charPos + v.lexeme.length),
            v.lexeme,
          );
        }
        let value;
        try {
          value = evaluate(expr, env);
        } catch (e) {
          assert.ok(e instanceof ExprError, `unexpected ${String(e)}`);
          checkError(expression, e);
          assert.throws(() => interpret(expr, env), ExprError);
          return;
        }
        assert.equal(interpret(expr, env), isTruthy(value));
      }),
      { numRuns },
    );
  });

  it('diagnoses and completes any expression', () => {
    fc.assert(
      fc.property(
        expression,
        fc.boolean(),
        fc.double({ min: 0, max: 1, noNaN: true }),
        (expression, explicit, at) => {
          const variables = [
            'b',
            'pipeline.git.branch',
            { name: 'old', deprecated: 'old' },
          ];
          for (const d of diagnose(expression, { variables })) {
            assert.ok(
              d.from >= 0 && d.from <= d.to && d.to <= expression.length,
              JSON.stringify(d),
            );
          }
          const pos = Math.floor(at * expression.length);
          const result = complete(expression, pos, { variables, explicit });
          if (result !== null) {
            assert.ok(
              result.from >= 0 &&
                result.from <= result.to &&
                result.to <= expression.length,
              JSON.stringify(result),
            );
          }
        },
      ),
      { numRuns },
    );
  });
});
