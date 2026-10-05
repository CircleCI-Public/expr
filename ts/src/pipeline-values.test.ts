import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import {
  pipelineValueFields,
  pipelineValueVariables,
} from './pipeline-values.ts';

describe('pipeline values', () => {
  it('are generated from the registry', () => {
    const branch = pipelineValueFields.find(
      (f) => f.name === 'pipeline.git.branch',
    );
    assert.equal(branch?.type, 'string');
    assert.equal(branch?.domain, 'git-branches');
  });

  it('hide private fields', () => {
    const vcsType = pipelineValueVariables().find(
      (v) => v.name === 'pipeline.vcs.type',
    );
    assert.equal(vcsType?.hidden, true);
  });

  it('are deprecated from their deprecation date', () => {
    const source = (now: string) =>
      pipelineValueVariables({ now: new Date(now) }).find(
        (v) => v.name === 'pipeline.trigger_source',
      );

    const before = source('2026-07-31T23:59:59Z');
    assert.equal(before?.deprecated, undefined);
    assert.equal(
      before?.pendingDeprecation,
      '"pipeline.trigger_source" will be deprecated on 2026-08-01, use "pipeline.trigger.type" instead',
    );

    const on = source('2026-08-01T00:00:00Z');
    assert.equal(
      on?.deprecated,
      '"pipeline.trigger_source" is deprecated since 2026-08-01, use "pipeline.trigger.type" instead',
    );
    assert.equal(on?.pendingDeprecation, undefined);
  });
});
