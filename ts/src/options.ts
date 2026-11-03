/** Options for scanning and parsing. The scanner and parser must be given the same options. */
export interface Options {
  /** Enable list literals and the `in` builtin. Off by default. */
  readonly lists?: boolean;
}
