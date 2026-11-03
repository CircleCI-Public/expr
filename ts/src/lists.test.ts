import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import {
  Builtin,
  ParseError,
  ParseErrorType,
  ScanError,
  ScanErrorType,
  TokenType,
  compile,
  evaluate,
  gatherVariables,
  interpret,
  scan,
} from './index.ts';

const lists = { lists: true };

describe('lists', () => {
  it('scans brackets, commas and `in` only when enabled', () => {
    const { tokens } = scan('x IN [1, 2]', lists);
    assert.deepEqual(
      tokens.map((t) => t.type),
      [
        TokenType.IDENTIFIER,
        TokenType.BUILTIN,
        TokenType.LEFT_BRACKET,
        TokenType.NUMBER,
        TokenType.COMMA,
        TokenType.NUMBER,
        TokenType.RIGHT_BRACKET,
        TokenType.EOF,
      ],
    );

    assert.equal(scan('in').tokens[0]?.type, TokenType.IDENTIFIER);
    for (const source of ['[', ']', ',']) {
      assert.equal(
        scan(source).error?.type,
        ScanErrorType.UNEXPECTED_CHARACTER,
      );
    }
  });

  it('keeps `in` an identifier with lists off', () => {
    assert.equal(interpret(compile('in == 1'), { in: 1 }), true);
    assert.throws(
      () => compile('x in "a"'),
      (e) =>
        e instanceof ParseError &&
        e.type === ParseErrorType.UNKNOWN_BUILTIN_FUNCTION,
    );
    assert.throws(() => compile('x in ["a"]'), ScanError);
  });

  it('parses a list as the right operand of in', () => {
    const expr = compile('x in ["a", "b"]', lists);
    assert.equal(expr.kind, 'infix');
    assert.equal(expr.kind === 'infix' && expr.builtin, Builtin.IN);
    assert.equal(expr.kind === 'infix' && expr.right.kind, 'list');
  });

  it('evaluates membership', () => {
    const env = { s: 'b', n: 2, t: true };
    for (const [expression, expected] of [
      ['s in ["a", "b"]', true],
      ['s in ["a", "c"]', false],
      ['n in [1, 2, 2]', true],
      ['t in [false]', false],
      ['"a" IN ["a"]', true],
      ['not (s in ["a"])', true],
      ['s in ["b"] and n in [3]', false],
    ] as const) {
      assert.equal(
        evaluate(compile(expression, lists), env),
        expected,
        expression,
      );
    }
  });

  it('gives undefined for an undefined left operand', () => {
    const expr = compile('u in ["a"]', lists);
    assert.equal(evaluate(expr, { u: undefined }), undefined);
    assert.equal(interpret(expr, { u: undefined }), false);
  });

  it('finds variables only in the left operand', () => {
    const names = gatherVariables(compile('a in [1] or b in [2]', lists)).map(
      (t) => t.lexeme,
    );
    assert.deepEqual(names, ['a', 'b']);
  });

  it('rejects a list outside of in', () => {
    assert.throws(
      () => compile('[1] == x', lists),
      (e) =>
        e instanceof ParseError && e.type === ParseErrorType.UNEXPECTED_LIST,
    );
  });
});
