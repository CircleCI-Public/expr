// Records how expr's Go implementation and re2j, used by the Java
// implementation, treat the patterns in src/testdata/re2-cases.json: whether they're
// valid, and whether they match the inputs.
//
// Usage: node scripts/re2-oracle/update.ts
//
// Needs Go, and Java with re2j 1.8 in the local Maven repository (run
// `lein deps` in clj/).

import { execFileSync } from 'node:child_process';
import { readFileSync, writeFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';

interface Verdict {
  valid: boolean;
  error?: string;
  matches?: boolean[];
}

interface Case {
  pattern: string;
  inputs?: string[];
  go?: Verdict;
  re2j?: Verdict;
}

const here = new URL('.', import.meta.url).pathname;
const casesFile = new URL('../../src/testdata/re2-cases.json', import.meta.url);
const re2j = join(
  homedir(),
  '.m2/repository/com/google/re2j/re2j/1.8/re2j-1.8.jar',
);

const cases = JSON.parse(readFileSync(casesFile, 'utf8')) as Case[];

const b64 = (s: string): string => Buffer.from(s, 'utf8').toString('base64');
const input = cases
  .flatMap((c) => [
    `P\t${b64(c.pattern)}`,
    ...(c.inputs ?? []).map((i) => `I\t${b64(i)}`),
  ])
  .join('\n');

function verdicts(command: string, args: string[]): Verdict[] {
  const lines = execFileSync(command, args, { cwd: here, input })
    .toString()
    .trimEnd()
    .split('\n');
  const result: Verdict[] = [];
  for (const line of lines) {
    const [kind, value, error] = line.split('\t');
    if (kind === 'V') {
      result.push(
        value === '1'
          ? { valid: true }
          : {
              valid: false,
              error: Buffer.from(error ?? '', 'base64').toString('utf8'),
            },
      );
    } else if (value !== '-') {
      const v = result.at(-1)!;
      v.matches = [...(v.matches ?? []), value === '1'];
    }
  }
  return result;
}

const go = verdicts('go', ['run', '.']);
const java = verdicts('java', ['-cp', re2j, 'Oracle.java']);

writeFileSync(
  casesFile,
  `${JSON.stringify(
    cases.map(({ pattern, inputs }, i) => ({
      pattern,
      ...(inputs === undefined ? {} : { inputs }),
      go: go[i],
      re2j: java[i],
    })),
    null,
    2,
  )}\n`,
);
