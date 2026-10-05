import type { Variable } from './editor.ts';
import { pipelineValueFields } from './pipeline-values.generated.ts';

export { pipelineValueFields };

/** A field in the pipeline values registry. */
export interface PipelineValueField {
  /**
   * The name of the pipeline value. A name ending `.*` stands for any name
   * with that prefix.
   */
  readonly name: string;
  /** The scalar type of the value, e.g. string, boolean, uint or uuid. */
  readonly type: string;
  /** Constraints on, and interpretations of, the value. */
  readonly domain?: string;
  readonly description?: string;
  /** The field is not publicly documented. */
  readonly private?: true;
  /** The date, YYYY-MM-DD, the field is deprecated from. */
  readonly deprecatedAt?: string;
  /** The field that replaces this deprecated one. */
  readonly replacedBy?: string;
}

/**
 * The pipeline values as editor variables. Private fields are hidden from
 * completion. Fields deprecated on or before `now` are deprecated variables,
 * which produce warnings; fields to be deprecated later produce info
 * diagnostics.
 */
export function pipelineValueVariables({
  now = new Date(),
}: { now?: Date } = {}): Variable[] {
  // Dates in the registry are days, YYYY-MM-DD, compared in UTC.
  const today = now.toISOString().slice(0, 10);
  return pipelineValueFields.map((f) => {
    const variable: Variable = {
      name: f.name,
      detail: f.type,
      ...(f.description === undefined ? {} : { info: f.description }),
      ...(f.private === true ? { hidden: true } : {}),
    };
    if (f.deprecatedAt === undefined) {
      return variable;
    }
    const replacement =
      f.replacedBy === undefined ? '' : `, use "${f.replacedBy}" instead`;
    return f.deprecatedAt <= today
      ? {
          ...variable,
          deprecated: `"${f.name}" is deprecated since ${f.deprecatedAt}${replacement}`,
        }
      : {
          ...variable,
          pendingDeprecation: `"${f.name}" will be deprecated on ${f.deprecatedAt}${replacement}`,
        };
  });
}
