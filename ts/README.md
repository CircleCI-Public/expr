# @circleci/expr

A TypeScript port of the expr scanner, parser and interpreter, with helpers for
validating and completing expressions in an editor such as CodeMirror. See the
[top-level README](../README.md) for the language.

It's pure ESM and runs in browsers and Node. Its one dependency is
[re2js](https://github.com/le0pard/re2js), a JavaScript port of re2j, for
regular expression patterns.

## Usage

```ts
import { compile, interpret } from '@circleci/expr';

const expr = compile('pipeline.git.branch matches /feat-.*/'); // throws ScanError or ParseError
interpret(expr, { 'pipeline.git.branch': 'feat-x' }); // true; throws InterpreterError
```

`evaluate` returns the value of an expression rather than its truthiness, and
`gatherVariables` returns the variables it refers to. Errors extend `ExprError`,
with `from`/`to` offsets and `asErrorMessage(expression)` for the same
multi-line messages as the Go and Clojure implementations.

`scan` never throws: on error it returns the tokens scanned so far with the
error. Parsing and evaluating recurse for each level of nesting, so `compile`,
`parse`, `interpret` and `evaluate` throw a `RangeError` for an expression
nested too deeply, from about 1,000 levels.

### In CodeMirror

`diagnose` and `complete` return the shapes CodeMirror's lint and autocomplete
sources use:

```ts
import { autocompletion } from '@codemirror/autocomplete';
import { linter } from '@codemirror/lint';
import { complete, diagnose } from '@circleci/expr';
import { pipelineValueVariables } from '@circleci/expr/pipeline-values';

const variables = pipelineValueVariables();

const extensions = [
  linter((view) => diagnose(view.state.doc.toString(), { variables })),
  autocompletion({
    override: [
      (context) =>
        complete(context.state.doc.toString(), context.pos, {
          variables,
          explicit: context.explicit,
        }),
    ],
  }),
];
```

With `variables`, `diagnose` reports references to unknown variables as errors,
deprecated ones as warnings, and ones that will be deprecated as info.
Deprecated variables are still offered as completions, ranked last and marked as
deprecated. A variable named like `pipeline.parameters.*` matches any name with
that prefix.

`diagnose` reports an empty expression as an error, as the interpreter does; if
empty input is allowed, check for it before calling `diagnose`. It never throws:
an expression too deeply nested to parse is reported as an error.
`@circleci/expr/pipeline-values` has the fields of the
[pipeline values registry](../domains/resources/com/circleci/expr/domains/pipeline-values.yml);
private fields are valid but not offered as completions, and fields are
deprecated from their deprecation date (`pipelineValueVariables({ now })`).

## Differences from the Go and Java implementations

- Offsets are UTF-16 code units, as JS strings and CodeMirror use, rather than
  Unicode code points. They only differ after characters outside the BMP.
- Patterns use re2js, to behave as in the Java implementation, which uses re2j.
  Where Go's regexp package and re2j differ, re2js mostly rejects the pattern,
  so the editor rejects some patterns Go accepts: loosely matched Unicode class
  names like `\p{greek}` and `\p{Letter}`, `\p{ASCII}`, `[[:]` and duplicate
  capture group names. It also rejects some re2j accepts: nested repeat counts
  that multiply past 1000, like `(x{100}){11}`, and escaped non-ASCII characters
  like `\é`. The exceptions are `\p{Cn}` and `\p{LC}`, which it accepts as Go
  does but re2j 1.8 doesn't.
- Matching is full-string, as in Java. Go instead checks whether the
  leftmost-first match spans the string, so for example `"ab" matches /a|ab/` is
  true here and in Java, but false in Go. `src/val.test.ts` lists the known
  cases.
- Numbers in the environment must be within the 64-bit integer range. Other
  values with no expr equivalent, like arrays, make `interpret` and `evaluate`
  throw a `TypeError` when they're referred to.

## Layout

Modules follow the Go implementation's files, e.g. `src/scanner.ts` is
`scanners/scanner.go` and `src/val.ts` is `interpreters/val.go`, with tests
alongside as `*.test.ts`. `src/index.ts` is the package entry point,
`src/compile.ts` scans and parses in one step, and `src/editor.ts` has the
editor helpers, which have no Go equivalent.

## Development

Use the Node version in `.nvmrc` and the pnpm version in `package.json`. Tests
run the TypeScript sources directly with Node's type stripping.

```sh
pnpm install
pnpm test        # node:test, including the shared corpus in ../clj/dev-resources
pnpm typecheck
pnpm lint        # oxlint
pnpm fmt         # oxfmt; fmt:check to check
pnpm build       # emit dist/
```

`pnpm generate` regenerates `src/pipeline-values.generated.ts` from the
registry; CI checks it's up to date.

`src/testdata/re2-cases.json` records how expr's Go implementation and re2j
treat a set of patterns, which re2js is tested against. To add cases, add a
`pattern` (and optionally `inputs` to match) and run `pnpm generate:re2-cases`,
which needs Go and Java with re2j in the local Maven repository (`lein deps` in
`clj/`).
