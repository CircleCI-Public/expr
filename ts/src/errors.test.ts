import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { compile } from './compile.ts';
import { ExprError, errorMessage, quoteRune } from './errors.ts';

// Ported from errors/errors_test.go.
describe('errorMessage', () => {
  it('pinpoints errors in single-line expressions', () => {
    assert.equal(
      errorMessage('Preamble message:', 'some.number > baz or true', 14, 3),
      [
        'Preamble message:',
        'some.number > baz or true',
        '              ^^^',
      ].join('\n'),
    );
  });

  it('pinpoints errors in multi-line expressions', () => {
    assert.equal(
      errorMessage(
        'Preamble message:',
        ['foo.bar == "main" and', 'some.number > baz or true'].join('\n'),
        36,
        3,
      ),
      [
        'Preamble message:',
        'some.number > baz or true',
        '              ^^^',
      ].join('\n'),
    );
  });

  it('marks an error on the end of a line', () => {
    assert.equal(
      errorMessage(
        'Preamble message:',
        ['foo.bar == "main" and', 'some.number > baz or t', '1 <= 15'].join(
          '\n',
        ),
        43,
        1,
      ),
      [
        'Preamble message:',
        'some.number > baz or t',
        '                     ^',
      ].join('\n'),
    );
  });

  it('marks an error after the end of a line', () => {
    assert.equal(
      errorMessage('Preamble message:', 'foo.bar.baz == (1 > 5', 21, 1),
      [
        'Preamble message:',
        'foo.bar.baz == (1 > 5',
        '                     ^',
      ].join('\n'),
    );
  });
});

describe('errors', () => {
  it('throw a RangeError for deeply nested expressions', () => {
    const n = 100_000;
    assert.throws(
      () => compile(`${'('.repeat(n)}true${')'.repeat(n)}`),
      RangeError,
    );
  });

  it('quote characters as Go does', () => {
    // From Go's fmt.Sprintf("%q", r).
    const cases: [string, string][] = [
      [' ', "' '"],
      ['\u00a0', "'\\u00a0'"],
      ['\u200b', "'\\u200b'"],
      ['\u0085', "'\\u0085'"],
      ['\x00', "'\\x00'"],
      ['\x7f', "'\\x7f'"],
      ['\u{e0001}', "'\\U000e0001'"],
      ['é', "'é'"],
      ['£', "'£'"],
      ['\u2028', "'\\u2028'"],
      ['\ufeff', "'\\ufeff'"],
      ['😀', "'😀'"],
      ["'", "'\\''"],
    ];
    for (const [c, quoted] of cases) {
      assert.equal(quoteRune(c), quoted);
    }
  });

  it('render as messages', () => {
    try {
      compile('2 > 5 && false');
      assert.fail('expected an error');
    } catch (error) {
      assert.ok(error instanceof ExprError);
      assert.equal(
        error.asErrorMessage('2 > 5 && false'),
        "Unexpected character '&':\n2 > 5 && false\n      ^",
      );
    }
  });
});
