import { RE2JS } from 're2js';

/** A compiled regular expression pattern literal. */
export class Pattern {
  /** The RE2 source of the pattern, without the surrounding slashes. */
  readonly source: string;
  readonly re2: RE2JS;

  /**
   * Compile an RE2 pattern.
   *
   * Throws RE2JSSyntaxException if the pattern is invalid.
   */
  constructor(source: string) {
    this.source = source;
    this.re2 = RE2JS.compile(source);
  }

  /**
   * Returns true if, and only if, the pattern matches the whole of s, as if
   * it had implicit ^ and $ anchors.
   */
  matches(s: string): boolean {
    return this.re2.matches(s);
  }

  toString(): string {
    return `/${this.source}/`;
  }
}

/**
 * A value in the expr language. `undefined` is the value of a variable that is
 * set in the environment without a value.
 */
export type Value = boolean | string | bigint | Pattern | undefined;

/**
 * The variables an expression is interpreted with. A variable that is present
 * with a null or undefined value is defined but has no value; referring to a
 * variable that is absent is an error.
 *
 * Numbers are truncated to integers, as expr only has 64-bit integer values.
 */
export type Environment =
  | ReadonlyMap<string, unknown>
  | Readonly<Record<string, unknown>>;

const MIN_INT64 = -(2n ** 63n);
const MAX_INT64 = 2n ** 63n - 1n;

/**
 * Convert a JS value to an expr Value.
 *
 * Throws a TypeError for values with no expr equivalent: objects other than
 * Patterns, and numbers that aren't finite or are outside the 64-bit integer
 * range.
 */
export function toValue(v: unknown): Value {
  switch (typeof v) {
    case 'undefined':
    case 'boolean':
    case 'string':
      return v;
    case 'bigint':
      if (v >= MIN_INT64 && v <= MAX_INT64) {
        return v;
      }
      break;
    case 'number':
      if (Number.isFinite(v)) {
        return toValue(BigInt(Math.trunc(v)));
      }
      break;
    case 'object':
      if (v === null) {
        return undefined;
      }
      if (v instanceof Pattern) {
        return v;
      }
      break;
  }
  throw new TypeError(`unable to convert ${String(v)} to an expr value`);
}

/** Undefined is falsey, booleans are their value, and all else is truthy. */
export function isTruthy(v: Value): boolean {
  return typeof v === 'boolean' ? v : v !== undefined;
}

export function equal(l: Value, r: Value): boolean {
  if (l instanceof Pattern && r instanceof Pattern) {
    return l.source === r.source;
  }
  return l === r;
}
