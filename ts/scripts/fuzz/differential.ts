// Differential fuzzer: evaluates generated expressions with the Go, Java and
// TypeScript implementations and reports where they disagree, grouped by the
// kind of disagreement, with a minimised example of each.
//
// Usage: node scripts/fuzz/differential.ts [--cases N] [--seconds N] [--seed N]
//
// Needs Go, and Java with the Java implementation compiled (`lein javac` in
// clj/) and re2j 1.8 in the local Maven repository.
//
// The oracles read a line per case and write a line per result:
//
//   request:  b64(expression) ( TAB b64(name) "," kind "," b64(value) )*
//             kind: b(oolean) s(tring) n(umber) p(attern) u(ndefined)
//   response: "R" TAB kind TAB b64(value)
//           | "E" TAB errorType TAB tokenType TAB b64(lexeme) TAB pos TAB b64(message)
//           | "X" TAB b64(unexpected exception)

import { type ChildProcess, execFileSync, spawn } from 'node:child_process';
import { writeFileSync } from 'node:fs';
import { homedir, tmpdir } from 'node:os';
import { join } from 'node:path';
import { createInterface } from 'node:readline';
import { parseArgs } from 'node:util';

import fc from 'fast-check';

import {
  ExprError,
  InterpreterError,
  ParseError,
  Pattern,
  ScanError,
  compile,
  evaluate,
} from '../../src/index.ts';
import { type Case, testCase, toEnvironment } from './arbitraries.ts';

const { values: args } = parseArgs({
  options: {
    cases: { type: 'string', default: '100000' },
    seconds: { type: 'string' },
    seed: { type: 'string' },
    out: { type: 'string', default: join(tmpdir(), 'expr-differential.json') },
  },
});

const here = new URL('.', import.meta.url).pathname;
const repo = join(here, '../../..');
const re2j = join(
  homedir(),
  '.m2/repository/com/google/re2j/re2j/1.8/re2j-1.8.jar',
);

type Outcome =
  | { kind: 'R'; type: string; value: string }
  | {
      kind: 'E';
      errorType: string;
      tokenType: string;
      lexeme: string;
      pos: number;
      message: string;
    }
  | { kind: 'X'; error: string };

const b64 = (s: string): string => Buffer.from(s, 'utf8').toString('base64');
const unb64 = (s: string): string => Buffer.from(s, 'base64').toString('utf8');

function encode({ expression, environment }: Case): string {
  return [
    b64(expression),
    ...Object.entries(environment).map(([name, v]) => {
      const value = v.kind === 'undefined' ? '' : String(v.value);
      return `${b64(name)},${v.kind[0]},${b64(value)}`;
    }),
  ].join('\t');
}

function decode(line: string): Outcome {
  const f = line.split('\t');
  switch (f[0]) {
    case 'R':
      return { kind: 'R', type: f[1]!, value: unb64(f[2]!) };
    case 'E':
      return {
        kind: 'E',
        errorType: f[1]!,
        tokenType: f[2]!,
        lexeme: unb64(f[3]!),
        pos: Number(f[4]),
        message: unb64(f[5]!),
      };
    default:
      return { kind: 'X', error: unb64(f[1] ?? '') };
  }
}

/** A long-running oracle process, answering requests in order. */
class Oracle {
  private readonly pending: ((line: string) => void)[] = [];
  private readonly proc: ChildProcess;

  constructor(command: string, commandArgs: string[], cwd: string) {
    this.proc = spawn(command, commandArgs, {
      cwd,
      stdio: ['pipe', 'pipe', 'inherit'],
    });
    createInterface({ input: this.proc.stdout! }).on('line', (line) =>
      this.pending.shift()!(line),
    );
    this.proc.on('exit', (code) => {
      if (this.pending.length > 0) {
        console.error(`${command} exited with ${code}`);
        process.exit(1);
      }
    });
  }

  run(c: Case): Promise<Outcome> {
    return new Promise((resolve) => {
      this.pending.push((line) => resolve(decode(line)));
      this.proc.stdin!.write(`${encode(c)}\n`);
    });
  }

  close(): void {
    this.proc.stdin!.end();
  }
}

function runTS({ expression, environment }: Case): Outcome {
  try {
    const v = evaluate(compile(expression), toEnvironment(environment));
    if (v === undefined) return { kind: 'R', type: 'u', value: '' };
    if (typeof v === 'boolean')
      return { kind: 'R', type: 'b', value: String(v) };
    if (typeof v === 'string') return { kind: 'R', type: 's', value: v };
    if (typeof v === 'bigint')
      return { kind: 'R', type: 'n', value: String(v) };
    if (v instanceof Pattern) return { kind: 'R', type: 'p', value: v.source };
    return { kind: 'X', error: `unexpected value ${String(v)}` };
  } catch (e) {
    if (e instanceof ScanError) {
      return error(`Scanner/${e.type}`, '-', e.char, e.pos, e, expression);
    }
    if (e instanceof ParseError || e instanceof InterpreterError) {
      const subsystem = e instanceof ParseError ? 'Parser' : 'Interpreter';
      return error(
        `${subsystem}/${e.type}`,
        e.token.type,
        e.token.lexeme,
        e.token.charPos,
        e,
        expression,
      );
    }
    return { kind: 'X', error: String(e) };
  }
}

function error(
  errorType: string,
  tokenType: string,
  lexeme: string,
  pos: number,
  e: ExprError,
  expression: string,
): Outcome {
  return {
    kind: 'E',
    errorType,
    tokenType,
    lexeme,
    pos,
    message: e.asErrorMessage(expression),
  };
}

/**
 * Converts a UTF-16 offset to code points, which Go's offsets are in. After
 * a character outside the BMP, error messages mark errors in those different
 * units, so they aren't compared.
 */
function normalise(
  expression: string,
  outcome: Outcome,
  utf16: boolean,
): Outcome {
  if (outcome.kind !== 'E') {
    return outcome;
  }
  const pos = utf16
    ? Array.from(expression.slice(0, outcome.pos)).length
    : outcome.pos;
  const nonBMP = /[\u{10000}-\u{10ffff}]/u.test(expression);
  return { ...outcome, pos, message: nonBMP ? '' : outcome.message };
}

const impls = ['go', 'java', 'ts'] as const;
type Impl = (typeof impls)[number];
type Outcomes = Record<Impl, Outcome>;

/** The kind of the outcome, without the details that vary between cases. */
function shape(o: Outcome): string {
  switch (o.kind) {
    case 'R':
      return `result:${o.type}`;
    case 'E':
      return o.errorType;
    case 'X':
      return `CRASH:${o.error.split(/[:\n]/)[0]}`;
  }
}

/**
 * Describes how the implementations disagree, e.g.
 * "go≠java=ts [result:b | Scanner/INVALID_PATTERN]", or undefined if they
 * agree. Outcomes that differ only in their details are described by the
 * fields that differ.
 */
function signature(o: Outcomes): string | undefined {
  const same = (a: Outcome, b: Outcome): boolean =>
    JSON.stringify(a) === JSON.stringify(b);
  if (same(o.go, o.java) && same(o.java, o.ts)) {
    return undefined;
  }
  const groups: Impl[][] = [];
  for (const impl of impls) {
    const group = groups.find((g) => same(o[g[0]!], o[impl]));
    if (group) group.push(impl);
    else groups.push([impl]);
  }
  const shapes = groups.map((g) => shape(o[g[0]!]));
  let detail = shapes.join(' | ');
  if (new Set(shapes).size === 1) {
    const [a, b] = groups.map((g) => o[g[0]!]) as [Outcome, Outcome];
    const fields = Object.keys(a).filter(
      (k) =>
        JSON.stringify(a[k as keyof Outcome]) !==
        JSON.stringify(b[k as keyof Outcome]),
    );
    detail = `${shapes[0]} differs in ${fields.join(',')}`;
  }
  return `${groups.map((g) => g.join('=')).join('≠')} [${detail}]`;
}

/** The position of an invalid pattern error, if the outcome is one. */
function invalidPatternAt(o: Outcome): number | undefined {
  return o.kind === 'E' && o.errorType === 'Scanner/INVALID_PATTERN'
    ? o.pos
    : undefined;
}

/**
 * Known causes of disagreement, checked in order. Each recognises outcomes
 * that disagree in the way the cause explains; anything they don't recognise
 * is reported as unexplained.
 */
const causes: { label: string; matches: (c: Case, o: Outcomes) => boolean }[] =
  [
    {
      label:
        'java: the pattern length limit is in UTF-16 code units, not UTF-8 bytes',
      matches: (_, o) =>
        o.go.kind === 'E' &&
        o.go.errorType === 'Scanner/PATTERN_TOO_LONG' &&
        !(
          o.java.kind === 'E' &&
          o.java.errorType === 'Scanner/PATTERN_TOO_LONG' &&
          o.java.pos === o.go.pos
        ),
    },
    {
      label:
        'java: matches checks the left operand is a string before the right is a pattern',
      matches: (_, o) =>
        o.java.kind === 'E' &&
        o.java.errorType === 'Interpreter/EXPECTED_STRING_OPERAND' &&
        o.go.kind === 'E' &&
        o.go.errorType === 'Interpreter/EXPECTED_PATTERN_OPERAND',
    },
    {
      label:
        're2: Go accepts duplicate capture group names, re2j and re2js reject them',
      matches: (c, o) =>
        /\(\?P?<(\w+)>.*\(\?P?<\1>/s.test(c.expression) &&
        invalidPatternAt(o.java) !== undefined &&
        invalidPatternAt(o.go) !== invalidPatternAt(o.java),
    },
    {
      label:
        're2: Go matches Unicode class names loosely, re2j and re2js exactly',
      matches: (c, o) =>
        /\\[pP]\{/.test(c.expression) &&
        o.java.kind === 'E' &&
        o.java.errorType === 'Scanner/INVALID_PATTERN' &&
        JSON.stringify(o.java) === JSON.stringify(o.ts),
    },
    {
      label:
        're2: re2j accepts escaped non-ASCII characters, Go and re2js reject them',
      matches: (c, o) =>
        /\\[\u{80}-\u{10ffff}]/u.test(c.expression) &&
        o.go.kind === 'E' &&
        o.go.errorType === 'Scanner/INVALID_PATTERN' &&
        JSON.stringify(o.go) === JSON.stringify(o.ts),
    },
    {
      label:
        'go: matches checks the leftmost-first match spans the string, not a full match',
      matches: (c, o) =>
        /matches/i.test(c.expression) &&
        o.go.kind === 'R' &&
        o.java.kind === 'R' &&
        JSON.stringify(o.java) === JSON.stringify(o.ts),
    },
    {
      label:
        'java: scan errors hold a UTF-16 code unit, and format it unescaped',
      matches: (_, o) =>
        o.java.kind === 'E' &&
        o.go.kind === 'E' &&
        o.ts.kind === 'E' &&
        o.java.errorType.startsWith('Scanner/') &&
        o.java.errorType === o.go.errorType &&
        o.java.pos === o.go.pos &&
        (o.java.lexeme !== o.go.lexeme || o.go.message === o.ts.message),
    },
  ];

const UNEXPLAINED = 'UNEXPLAINED';

function explain(c: Case, o: Outcomes): string | undefined {
  const sig = signature(o);
  if (sig === undefined) return undefined;
  return `${causes.find((cause) => cause.matches(c, o))?.label ?? UNEXPLAINED}\n    ${sig}`;
}

const goBinary = join(tmpdir(), 'expr-differential-go-oracle');
execFileSync('go', ['build', '-o', goBinary, '.'], {
  cwd: join(here, 'oracle'),
  stdio: 'inherit',
});
const go = new Oracle(goBinary, [], here);
const java = new Oracle(
  'java',
  [
    '-cp',
    `${join(repo, 'clj/target/classes')}:${re2j}`,
    join(here, 'oracle/Oracle.java'),
  ],
  here,
);

async function outcomes(c: Case): Promise<Outcomes> {
  const [g, j] = await Promise.all([go.run(c), java.run(c)]);
  return {
    go: normalise(c.expression, g, false),
    java: normalise(c.expression, j, true),
    ts: normalise(c.expression, runTS(c), true),
  };
}

/** Greedily shrinks a case while it still disagrees in the same way. */
async function minimise(c: Case, sig: string): Promise<Case> {
  const keeps = async (candidate: Case): Promise<boolean> =>
    explain(candidate, await outcomes(candidate)) === sig;
  let best = c;
  for (const name of Object.keys(best.environment)) {
    const { [name]: _, ...environment } = best.environment;
    const candidate = { ...best, environment };
    if (await keeps(candidate)) best = candidate;
  }
  // Delete code points, not UTF-16 code units, so as not to split surrogate
  // pairs.
  let chars = Array.from(best.expression);
  for (let size = Math.max(1, chars.length >> 1); size >= 1; size >>= 1) {
    for (let i = 0; i + size <= chars.length;) {
      const shorter = [...chars.slice(0, i), ...chars.slice(i + size)];
      const candidate = { ...best, expression: shorter.join('') };
      if (await keeps(candidate)) {
        best = candidate;
        chars = shorter;
      } else {
        i += size;
      }
    }
  }
  return best;
}

interface Bucket {
  count: number;
  example: Case;
}

const buckets = new Map<string, Bucket>();
const maxCases = Number(args.cases);
const deadline =
  args.seconds === undefined
    ? Infinity
    : Date.now() + Number(args.seconds) * 1000;
const seed = args.seed === undefined ? Date.now() : Number(args.seed);
const batchSize = 1000;
let total = 0;

console.error(`seed ${seed}`);
for (let batch = 0; total < maxCases && Date.now() < deadline; batch++) {
  const cases = fc.sample(testCase, {
    numRuns: Math.min(batchSize, maxCases - total),
    seed: seed + batch,
  });
  const results = await Promise.all(cases.map(outcomes));
  cases.forEach((c, i) => {
    const sig = explain(c, results[i]!);
    if (sig === undefined) return;
    const bucket = buckets.get(sig);
    if (bucket === undefined) {
      buckets.set(sig, { count: 1, example: c });
    } else {
      bucket.count++;
      if (c.expression.length < bucket.example.expression.length)
        bucket.example = c;
    }
  });
  total += cases.length;
  process.stderr.write(
    `\r${total} cases, ${buckets.size} kinds of disagreement`,
  );
}
process.stderr.write('\n');

const report = [];
const unexplainedFirst = (sig: string): number =>
  sig.startsWith(UNEXPLAINED) ? 0 : 1;
const sorted = [...buckets].sort(
  ([sa, a], [sb, b]) =>
    unexplainedFirst(sa) - unexplainedFirst(sb) ||
    sa.localeCompare(sb) ||
    b.count - a.count,
);
for (const [sig, bucket] of sorted) {
  const example = await minimise(bucket.example, sig);
  const o = await outcomes(example);
  report.push({ signature: sig, count: bucket.count, example, outcomes: o });
  console.log(`\n### ${sig} (${bucket.count} cases)`);
  console.log(`expression:  ${JSON.stringify(example.expression)}`);
  if (Object.keys(example.environment).length > 0) {
    console.log(
      `environment: ${JSON.stringify(example.environment, (_, v) => (typeof v === 'bigint' ? String(v) : v))}`,
    );
  }
  for (const impl of impls) {
    console.log(`${impl.padEnd(5)} ${JSON.stringify(o[impl])}`);
  }
}
writeFileSync(
  args.out,
  JSON.stringify(report, (_, v) => (typeof v === 'bigint' ? String(v) : v), 2),
);
const unexplained = sorted.filter(([sig]) =>
  sig.startsWith(UNEXPLAINED),
).length;
console.error(
  `\n${total} cases, ${buckets.size} kinds of disagreement, ${unexplained} unexplained, report in ${args.out}`,
);

go.close();
java.close();
