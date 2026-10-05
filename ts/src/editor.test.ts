import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { complete, diagnose } from './editor.ts';
import { pipelineValueVariables } from './pipeline-values.ts';

describe('diagnose', () => {
  it('accepts valid expressions', () => {
    assert.deepEqual(diagnose('foo and bar == "baz"'), []);
  });

  it('reports scan errors', () => {
    assert.deepEqual(diagnose('2 > 5 && false'), [
      {
        from: 6,
        to: 7,
        severity: 'error',
        message: "Unexpected character '&'",
        source: 'expr',
      },
    ]);
  });

  it('reports parse errors', () => {
    assert.deepEqual(diagnose('foo and ) bar'), [
      {
        from: 8,
        to: 9,
        severity: 'error',
        message: 'Expected expression, found ")"',
        source: 'expr',
      },
    ]);
  });

  it('reports unknown builtin functions', () => {
    assert.deepEqual(diagnose('foo containz "x"'), [
      {
        from: 4,
        to: 12,
        severity: 'error',
        message: 'Unknown infix function',
        source: 'expr',
      },
    ]);
  });

  it('highlights to the end of unterminated strings', () => {
    assert.deepEqual(diagnose('foo == "bar'), [
      {
        from: 7,
        to: 11,
        severity: 'error',
        message: 'Unterminated string starting here',
        source: 'expr',
      },
    ]);
  });

  it('keeps errors at the end of the expression within it', () => {
    const [diagnostic] = diagnose('(foo');
    assert.equal(diagnostic?.from, 4);
    assert.equal(diagnostic?.to, 4);
    assert.equal(diagnostic?.message, "Expected ')' after expression");
  });

  it('says "end of expression" for errors at the end', () => {
    const messages = (expression: string) =>
      diagnose(expression).map((d) => [d.from, d.to, d.message]);
    assert.deepEqual(messages('foo ='), [
      [5, 5, 'Incomplete token, expected "==", found end of expression'],
    ]);
    assert.deepEqual(messages('foo and'), [
      [7, 7, 'Expected expression, found end of expression'],
    ]);
    // Empty expressions are errors; consumers that allow them check first.
    assert.deepEqual(messages(''), [
      [0, 0, 'Expected expression, found end of expression'],
    ]);
  });

  it('shows invisible characters', () => {
    for (const [c, shown] of [
      ['\u00a0', '\\u00a0'],
      ['\u200b', '\\u200b'],
      ['\u0085', '\\u0085'],
    ]) {
      assert.equal(
        diagnose(`a ==${c}b`)[0]?.message,
        `Unexpected character '${shown}'`,
      );
    }
  });

  it('reports expressions too deeply nested to parse', () => {
    for (const expression of [
      '('.repeat(5000) + 'x',
      'not '.repeat(20000) + 'x',
      '!'.repeat(20000) + 'x',
    ]) {
      assert.deepEqual(diagnose(expression), [
        {
          from: 0,
          to: expression.length,
          severity: 'error',
          message: 'Expression is too deeply nested',
          source: 'expr',
        },
      ]);
    }
  });

  it('handles long chains of operators', () => {
    const expression = 'a and '.repeat(20000) + 'b';
    assert.deepEqual(
      diagnose(expression, { variables: ['a'] }).map((d) => d.message),
      ['Unknown variable "b"'],
    );
  });

  it('reports invalid patterns', () => {
    assert.equal(
      diagnose('foo matches /(?=x)/')[0]?.message,
      'Syntax error in pattern',
    );
  });

  it('reports unknown and deprecated variables', () => {
    const variables = [
      'foo',
      { name: 'old', deprecated: 'use foo' },
      { name: 'params.*' },
      { name: 'soon', pendingDeprecation: 'use foo soon' },
    ];
    assert.deepEqual(
      diagnose('foo and old and bar or params.x or params or soon', {
        variables,
      }),
      [
        {
          from: 8,
          to: 11,
          severity: 'warning',
          message: 'use foo',
          source: 'expr',
        },
        {
          from: 16,
          to: 19,
          severity: 'error',
          message: 'Unknown variable "bar"',
          source: 'expr',
        },
        {
          from: 35,
          to: 41,
          severity: 'error',
          message: 'Unknown variable "params"',
          source: 'expr',
        },
        {
          from: 45,
          to: 49,
          severity: 'info',
          message: 'use foo soon',
          source: 'expr',
        },
      ],
    );
  });

  it('knows the pipeline values', () => {
    const variables = pipelineValueVariables({
      now: new Date('2026-10-05T12:00:00Z'),
    });
    assert.deepEqual(
      diagnose(
        'pipeline.git.branch == "main" and pipeline.parameters.deploy and pipeline.vcs.type == "github" and pipeline.trigger_parameters.gitlab.branch == "main"',
        { variables },
      ).map((d) => d.severity),
      ['warning', 'info'],
    );
  });
});

describe('complete', () => {
  const variables = [
    { name: 'pipeline.git.branch', detail: 'string' },
    { name: 'pipeline.parameters.*' },
    { name: 'pipeline.secret', hidden: true },
  ];
  const labels = (expression: string, pos = expression.length) =>
    complete(expression, pos, { variables })?.options.map((o) => o.label);

  it('offers operands at the start', () => {
    assert.deepEqual(labels('pi'), [
      'pipeline.git.branch',
      'pipeline.parameters.',
      'true',
      'false',
      'not',
    ]);
  });

  it('replaces the partial word, including dots', () => {
    const result = complete('not pipeline.gi', 15, { variables });
    assert.equal(result?.from, 4);
    assert.equal(result?.to, 15);
  });

  it('offers operators after an operand', () => {
    assert.deepEqual(labels('pipeline.git.branch st'), [
      'and',
      'or',
      'contains',
      'starts-with',
      'matches',
    ]);
    assert.deepEqual(labels('(a == "b") an'), [
      'and',
      'or',
      'contains',
      'starts-with',
      'matches',
    ]);
  });

  it('offers operands after operators', () => {
    for (const expression of [
      'a and p',
      'a == p',
      'a contains p',
      '(p',
      'not p',
    ]) {
      assert.equal(labels(expression)?.[0], 'pipeline.git.branch', expression);
    }
  });

  it('completes at the cursor, not the end', () => {
    assert.deepEqual(labels('a an b', 4)?.slice(0, 2), ['and', 'or']);
  });

  it('only completes without a word when explicit', () => {
    assert.equal(complete('a and ', 6, { variables }), null);
    assert.equal(complete('a and ', 6, { variables, explicit: true })?.from, 6);
  });

  it('does not complete inside strings or patterns', () => {
    assert.equal(complete('a == "pi', 8, { variables }), null);
    assert.equal(complete('a matches /pi', 13, { variables }), null);
  });

  it('does not complete after a scan error', () => {
    assert.equal(complete('foo & ba', 8, { variables }), null);
  });

  it('ranks deprecated variables last and marks them', () => {
    const options = complete('p', 1, {
      variables: [
        { name: 'p.old', detail: 'string', deprecated: 'use p.new' },
        { name: 'p.soon', info: 'Soon.', pendingDeprecation: 'use p.new soon' },
        { name: 'p.new', detail: 'string' },
      ],
    })?.options.filter((o) => o.type === 'variable');
    assert.deepEqual(options, [
      {
        label: 'p.old',
        type: 'variable',
        detail: 'string (deprecated)',
        info: 'use p.new',
        boost: -50,
      },
      { label: 'p.soon', type: 'variable', info: 'use p.new soon\n\nSoon.' },
      { label: 'p.new', type: 'variable', detail: 'string' },
    ]);
  });
});
