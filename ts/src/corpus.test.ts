// Runs the test corpora shared with the Go and Clojure implementations. See
// the .edn files in clj/dev-resources/ for each corpus's format.

import assert from 'node:assert/strict';
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { describe, it } from 'node:test';

import {
  type Environment,
  InterpreterError,
  ParseError,
  Pattern,
  ScanError,
  TypeCheckError,
  type Value,
  checkTypes,
  compile,
  evaluate,
  interpret,
} from './index.ts';

const resources = new URL('../../clj/dev-resources/', import.meta.url).pathname;

interface ExpectedError {
  tokenType?: string;
  errorType: string;
  lexeme: string;
  charPos: number;
}

interface Case<Input> {
  input: Input & { expression: string };
  expected: { error?: ExpectedError; result?: unknown };
}

interface EvaluationInput {
  environment: Record<string, unknown>;
}

const REGEXP_PREFIX = 'corpus/regexp:';

/**
 * Parse corpus JSON, reading integer values as bigints from their source text,
 * so values beyond Number's precision aren't rounded. Positions stay numbers.
 */
function parseCorpus(text: string): unknown {
  return JSON.parse(
    text,
    (key: string, value: unknown, context?: { source?: string }) => {
      const source = context?.source ?? '';
      return key !== 'charPos' &&
        typeof value === 'number' &&
        /^-?\d+$/.test(source)
        ? BigInt(source)
        : value;
    },
  );
}

/** Convert a corpus JSON value to the JS value it stands for. */
function fromJSON(v: unknown): unknown {
  if (typeof v === 'string' && v.startsWith(REGEXP_PREFIX)) {
    return new Pattern(v.slice(REGEXP_PREFIX.length));
  }
  return v;
}

function toEnvironment(json: Record<string, unknown>): Environment {
  return Object.fromEntries(
    Object.entries(json).map(([k, v]) => [k, fromJSON(v)]),
  );
}

/** A comparable representation of a result. */
function outcome(expression: string, run: () => Value): unknown {
  let error: unknown;
  try {
    const result = run();
    if (result instanceof Pattern) {
      return { result: `${REGEXP_PREFIX}${result.source}` };
    }
    return { result };
  } catch (e) {
    error = e;
  }
  if (error instanceof ScanError) {
    return {
      error: {
        errorType: `Scanner/${error.type}`,
        lexeme: error.char,
        charPos: error.pos,
      },
    };
  }
  if (error instanceof TypeCheckError) {
    return {
      error: {
        tokenType: error.token.type,
        errorType: `TypeChecker/${error.type}`,
        lexeme: error.token.lexeme,
        charPos: error.token.charPos,
        message: error.asErrorMessage(expression),
      },
    };
  }
  if (error instanceof ParseError || error instanceof InterpreterError) {
    const subsystem = error instanceof ParseError ? 'Parser' : 'Interpreter';
    return {
      error: {
        tokenType: error.token.type,
        errorType: `${subsystem}/${error.type}`,
        lexeme: error.token.lexeme,
        charPos: error.token.charPos,
      },
    };
  }
  throw error;
}

function expectedOutcome(expected: Case<unknown>['expected']): unknown {
  if (expected.error !== undefined) {
    return { error: expected.error };
  }
  return { result: expected.result ?? undefined };
}

function corpus<Input>(
  dir: string,
  run: (input: Case<Input>['input']) => Value,
): void {
  const root = join(resources, dir);
  const files = readdirSync(root, { recursive: true, encoding: 'utf8' })
    .filter((f) => f.endsWith('.json'))
    .sort();
  assert.ok(files.length > 0, `no corpus files in ${root}`);

  describe(dir, () => {
    for (const file of files) {
      const { input, expected } = parseCorpus(
        readFileSync(join(root, file), 'utf8'),
      ) as Case<Input>;
      it(`${file}: ${input.expression}`, () => {
        assert.deepEqual(
          outcome(input.expression, () => run(input)),
          expectedOutcome(expected),
        );
      });
    }
  });
}

corpus<EvaluationInput>('test-corpus', ({ expression, environment }) =>
  interpret(compile(expression), toEnvironment(environment)),
);
corpus<EvaluationInput>(
  'evaluator-test-corpus',
  ({ expression, environment }) =>
    evaluate(compile(expression), toEnvironment(environment)),
);
corpus<{ types: Record<string, string> }>(
  'type-check-test-corpus',
  ({ expression, types }) => {
    checkTypes(compile(expression), new Map(Object.entries(types)));
    return undefined;
  },
);
