// fast-check arbitraries for expr expressions and environments.

import fc from 'fast-check';

import { Pattern } from '../../src/index.ts';

/** A value in an environment, in a form every implementation can build. */
export type EnvValue =
  | { kind: 'bool'; value: boolean }
  | { kind: 'string'; value: string }
  | { kind: 'number'; value: bigint }
  | { kind: 'pattern'; value: string }
  | { kind: 'undefined' };

export interface Case {
  expression: string;
  environment: Record<string, EnvValue>;
}

const MAX_INT64 = 2n ** 63n - 1n;
const MIN_INT64 = -(2n ** 63n);

/** Names the expression generator refers to, which the environment may set. */
export const variableNames = [
  'b',
  'n',
  's',
  'p',
  'u',
  'pipeline.git.branch',
  'pipeline.parameters.dry-run?',
  'x_1',
] as const;

const keywords = [
  'and',
  'AND',
  'or',
  'OR',
  'not',
  'NOT',
  'true',
  'TRUE',
  'false',
  'FALSE',
  'True',
  'And',
];
const builtins = [
  'contains',
  'CONTAINS',
  'matches',
  'MATCHES',
  'starts-with',
  'STARTS-WITH',
  'Contains',
  'ends-with',
];
const operators = ['==', '!=', '>', '>=', '<', '<=', '!', '=', '(', ')'];

/** Characters that tend to sit on boundaries between implementations. */
const interestingChars = [
  'a',
  'Z',
  '0',
  ' ',
  '\t',
  '\n',
  '\r',
  '"',
  '\\',
  '/',
  '.',
  '-',
  '?',
  '_',
  '\0',
  '\x7f',
  '\x85',
  '\xa0',
  'é',
  'ÿ',
  'Ā',
  ' ',
  '​',
  '日',
  '😀',
  '\u{10ffff}',
  '﻿',
];

const wellFormedString = (maxLength: number): fc.Arbitrary<string> =>
  fc.oneof(
    fc.string({ unit: 'grapheme', maxLength }),
    fc.string({ unit: fc.constantFrom(...interestingChars), maxLength }),
    // A small alphabet, so that patterns sometimes match.
    fc.string({ unit: fc.constantFrom('a', 'b', 'é', 'A', '\n'), maxLength }),
  );

const whitespace = fc.oneof(
  { weight: 8, arbitrary: fc.constant(' ') },
  { weight: 1, arbitrary: fc.constantFrom('', '\n', '\t', '\r\n', '  ') },
);

const numberLiteral = fc.oneof(
  fc.bigInt({ min: 0n, max: MAX_INT64 }).map(String),
  fc.constantFrom(
    '0',
    '00',
    '007',
    '9223372036854775807',
    '9223372036854775808',
    '99999999999999999999',
  ),
);

/** A string literal, with escapes, as it appears in an expression. */
const stringLiteral = wellFormedString(12).map(
  (s) => `"${s.replaceAll('\\', '\\\\').replaceAll('"', '\\"')}"`,
);

const regexAtom = fc.oneof(
  fc.constantFrom(
    'a',
    'b',
    'é',
    'ÿ',
    '.',
    '\\d',
    '\\w',
    '\\s',
    '\\S',
    '\\.',
    '\\\\',
    '\\/',
    '[a-z]',
    '[^x]',
    '[[:alpha:]]',
    '\\pL',
    '\\p{Greek}',
    '\\p{L}',
    '\\PL',
    '\\Q*\\E',
    '\\b',
    '\\A',
    '\\z',
    '^',
    '$',
    '(?i)',
    '(?s)',
    '(?U)',
    '(?m)',
    '\\C',
    '\\x41',
    '\\101',
    '\\e',
  ),
  fc.constantFrom(...interestingChars).filter((c) => c !== '/'),
);

const regex: fc.Arbitrary<string> = fc.letrec<{ re: string }>((tie) => ({
  re: fc.oneof(
    { depthSize: 'small', withCrossShrink: true },
    regexAtom,
    fc
      .tuple(
        tie('re'),
        fc.constantFrom(
          '*',
          '+',
          '?',
          '{2}',
          '{1,3}',
          '*?',
          '{0,}',
          '{1001}',
          '{2,1}',
        ),
      )
      .map(([r, q]) => `(?:${r})${q}`),
    fc
      .array(tie('re'), { minLength: 1, maxLength: 4 })
      .map((rs) => rs.join('')),
    fc
      .array(tie('re'), { minLength: 2, maxLength: 3 })
      .map((rs) => rs.join('|')),
    tie('re').map((r) => `(${r})`),
    tie('re').map((r) => `(?P<g>${r})`),
  ),
})).re;

/** A pattern literal, as it appears in an expression. */
const patternLiteral = fc
  .oneof(
    { weight: 6, arbitrary: regex },
    // Near the length limit, which is in bytes of UTF-8 in Go.
    {
      weight: 1,
      arbitrary: fc
        .tuple(fc.integer({ min: 120, max: 260 }), fc.constantFrom('a', 'é'))
        .map(([n, c]) => c.repeat(n)),
    },
    { weight: 1, arbitrary: wellFormedString(10) },
  )
  .map((re) => `/${re}/`);

const identifier = fc.oneof(
  { weight: 4, arbitrary: fc.constantFrom(...variableNames) },
  {
    weight: 1,
    arbitrary: fc.stringMatching(/^[a-zA-Z][a-zA-Z0-9_?.-]{0,8}$/),
  },
);

const primary = fc.oneof(
  identifier,
  numberLiteral,
  stringLiteral,
  patternLiteral,
  fc.constantFrom(...keywords.filter((k) => /^(true|false)$/i.test(k))),
);

/** A syntactically plausible expression, as a list of tokens. */
const grammarTokens: fc.Arbitrary<string[]> = fc.letrec<{ expr: string[] }>(
  (tie) => ({
    expr: fc.oneof(
      { depthSize: 'small', withCrossShrink: true },
      primary.map((p) => [p]),
      fc
        .tuple(
          tie('expr'),
          fc.constantFrom(
            '==',
            '!=',
            '>',
            '>=',
            '<',
            '<=',
            ...keywords.slice(0, 4),
            ...builtins,
          ),
          tie('expr'),
        )
        .map(([l, op, r]) => [...l, op, ...r]),
      fc
        .tuple(fc.constantFrom('not', 'NOT', '!'), tie('expr'))
        .map(([op, e]) => [op, ...e]),
      tie('expr').map((e) => ['(', ...e, ')']),
    ),
  }),
).expr;

/** Arbitrary tokens in arbitrary order. */
const tokenSoup = fc.array(
  fc.oneof(
    primary,
    fc.constantFrom(...keywords, ...builtins, ...operators),
    fc.constantFrom(...interestingChars),
  ),
  { maxLength: 12 },
);

/** Join tokens with whitespace, then perhaps corrupt a character. */
const render = (tokens: fc.Arbitrary<string[]>): fc.Arbitrary<string> =>
  fc
    .tuple(tokens, fc.infiniteStream(whitespace))
    .map(([ts, ws]) => {
      let out = '';
      for (const t of ts) {
        out += t + ws.next().value;
      }
      return out;
    })
    .chain((s) => {
      // Edit code points, so as not to split surrogate pairs.
      const cs = Array.from(s);
      return fc.oneof(
        { weight: 6, arbitrary: fc.constant(s) },
        {
          weight: 1,
          arbitrary: fc
            .tuple(
              fc.nat(cs.length),
              fc.constantFrom(...interestingChars, '=', '<', '!', '('),
            )
            .map(([i, c]) => [...cs.slice(0, i), c, ...cs.slice(i)].join('')),
        },
        {
          weight: 1,
          arbitrary: fc
            .nat(Math.max(0, cs.length - 1))
            .map((i) => [...cs.slice(0, i), ...cs.slice(i + 1)].join('')),
        },
      );
    });

const smallAlphabet = ['a', 'b', 'é', 'A', '\n'];

/** A pattern over a small alphabet, which strings over it may match. */
const smallRegex: fc.Arbitrary<string> = fc.letrec<{ re: string }>((tie) => ({
  re: fc.oneof(
    { depthSize: 'small', withCrossShrink: true },
    fc.constantFrom(
      ...smallAlphabet.filter((c) => c !== '\n'),
      '.',
      '[ab]',
      '[^a]',
      '\\w',
      '(?i)a',
      '(?s).',
      '^',
      '$',
      '\\b',
    ),
    fc
      .tuple(
        tie('re'),
        fc.constantFrom(
          '*',
          '+',
          '?',
          '*?',
          '+?',
          '??',
          '{2}',
          '{1,2}',
          '{0,}?',
        ),
      )
      .map(([r, q]) => `(?:${r})${q}`),
    fc
      .array(tie('re'), { minLength: 2, maxLength: 3 })
      .map((rs) => rs.join('')),
    fc
      .array(tie('re'), { minLength: 2, maxLength: 3 })
      .map((rs) => `(?:${rs.join('|')})`),
  ),
})).re;

/** A string matched against a pattern, to compare the regex engines. */
const matching = fc
  .tuple(
    fc.string({ unit: fc.constantFrom(...smallAlphabet), maxLength: 6 }),
    smallRegex,
  )
  .map(([s, re]) => `"${s}" matches /${re}/`);

/** An expression to fuzz, which may well be invalid. */
export const expression: fc.Arbitrary<string> = fc.oneof(
  { weight: 6, arbitrary: render(grammarTokens) },
  { weight: 2, arbitrary: matching },
  { weight: 3, arbitrary: render(tokenSoup) },
  { weight: 1, arbitrary: wellFormedString(30) },
);

/** Patterns for the environment, which every implementation accepts. */
const envPattern = fc.constantFrom(
  'fo+',
  'a|ab',
  '.*',
  '',
  '(?i)FOO',
  '[0-9]+',
  'é+',
);

const envValue: fc.Arbitrary<EnvValue> = fc.oneof(
  fc.boolean().map((value) => ({ kind: 'bool' as const, value })),
  wellFormedString(8).map((value) => ({ kind: 'string' as const, value })),
  fc
    .oneof(
      fc.bigInt({ min: MIN_INT64, max: MAX_INT64 }),
      fc.constantFrom(0n, 1n, -1n, 42n, MAX_INT64, MIN_INT64),
    )
    .map((value) => ({ kind: 'number' as const, value })),
  envPattern.map((value) => ({ kind: 'pattern' as const, value })),
  fc.constant({ kind: 'undefined' as const }),
);

/** An environment setting some of the variables the expressions refer to. */
export const environment: fc.Arbitrary<Record<string, EnvValue>> =
  fc.dictionary(fc.constantFrom(...variableNames), envValue, {
    maxKeys: variableNames.length,
  });

export const testCase: fc.Arbitrary<Case> = fc.record({
  expression,
  environment,
});

/** Convert an environment to the values the TypeScript implementation takes. */
export function toEnvironment(
  env: Record<string, EnvValue>,
): Map<string, unknown> {
  return new Map(
    Object.entries(env).map(([name, v]) => [
      name,
      v.kind === 'undefined'
        ? undefined
        : v.kind === 'pattern'
          ? new Pattern(v.value)
          : v.value,
    ]),
  );
}
