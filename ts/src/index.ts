export { gatherVariables } from './analyser.ts';
export type * from './expr.ts';
export { Builtin, builtinForLexeme } from './builtin.ts';
export { compile } from './compile.ts';
export {
  type CompleteOptions,
  type Completion,
  type CompletionResult,
  type DiagnoseOptions,
  type Diagnostic,
  type Variable,
  complete,
  diagnose,
} from './editor.ts';
export { ExprError, errorMessage } from './errors.ts';
export {
  InterpreterError,
  InterpreterErrorType,
  evaluate,
  interpret,
} from './interpreter.ts';
export { ParseError, ParseErrorType, parse } from './parser.ts';
export {
  MAX_PATTERN_LENGTH,
  ScanError,
  ScanErrorType,
  type ScanResult,
  scan,
} from './scanner.ts';
export { type Literal, type Token, TokenType } from './token.ts';
export {
  type Environment,
  Pattern,
  type Value,
  isTruthy,
  toValue,
} from './val.ts';
