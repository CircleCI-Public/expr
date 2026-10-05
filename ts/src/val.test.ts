// Checks patterns behave as in the Java implementation, comparing against the
// verdicts of expr's Go implementation and re2j recorded in
// testdata/re2-cases.json by scripts/re2-oracle/update.ts.

import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { describe, it } from 'node:test';

import { RE2JSSyntaxException } from 're2js';

import { Pattern, toValue } from './val.ts';

interface Verdict {
  valid: boolean;
  error?: string;
  matches?: boolean[];
}

interface Case {
  pattern: string;
  inputs?: string[];
  go: Verdict;
  re2j: Verdict;
}

const cases = JSON.parse(
  readFileSync(new URL('testdata/re2-cases.json', import.meta.url), 'utf8'),
) as Case[];

// Patterns re2js accepts though re2j 1.8, with its Unicode 6.0 tables,
// doesn't. Go accepts them.
const acceptedByGoOnly = new Set(['\\p{Cn}', '\\p{LC}']);

// Patterns that match differently in expr's Go implementation; see the README.
const goMatchDivergences = [
  'a|ab',
  'a??',
  '(|a)',
  '(?U)a+',
  '(?U)(a+)(a*)',
  'a*?',
];

describe('patterns agree with Go and re2j', () => {
  for (const c of cases) {
    // Where the implementations disagree, re2js takes the stricter view.
    const valid =
      (c.go.valid && c.re2j.valid) || acceptedByGoOnly.has(c.pattern);
    it(`${JSON.stringify(c.pattern)} is ${valid ? 'valid' : 'invalid'}`, () => {
      let pattern: Pattern;
      try {
        pattern = new Pattern(c.pattern);
      } catch (e) {
        if (!(e instanceof RE2JSSyntaxException)) {
          throw e;
        }
        assert.ok(!valid, `rejected valid pattern: ${e.message}`);
        return;
      }
      assert.ok(
        valid,
        `accepted invalid pattern: ${c.go.error ?? c.re2j.error}`,
      );
      assert.deepEqual(
        (c.inputs ?? []).map((input) => pattern.matches(input)),
        c.re2j.matches ?? [],
      );
    });
  }

  it('match differently in Go only where known', () => {
    const divergent = cases
      .filter(
        (c) =>
          c.go.valid &&
          c.re2j.valid &&
          JSON.stringify(c.go.matches) !== JSON.stringify(c.re2j.matches),
      )
      .map((c) => c.pattern);
    assert.deepEqual(divergent, goMatchDivergences);
  });
});

describe('toValue', () => {
  it('truncates numbers to integers', () => {
    assert.equal(toValue(1.9), 1n);
    assert.equal(toValue(-1.9), -1n);
  });

  it('accepts the 64-bit integer range', () => {
    assert.equal(toValue(2n ** 63n - 1n), 2n ** 63n - 1n);
    assert.equal(toValue(-(2n ** 63n)), -(2n ** 63n));
  });

  it('rejects values with no expr equivalent', () => {
    for (const v of [
      2n ** 63n,
      -(2n ** 63n) - 1n,
      1e300,
      NaN,
      Infinity,
      [],
      {},
    ]) {
      assert.throws(() => toValue(v), TypeError, String(v));
    }
  });

  it('treats null as undefined', () => {
    assert.equal(toValue(null), undefined);
  });
});
