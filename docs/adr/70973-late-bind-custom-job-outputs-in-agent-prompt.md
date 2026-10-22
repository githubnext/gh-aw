# ADR-70973: Late-Bind Custom Job Outputs in the Agent Prompt and Lint Unavailable Prompt References

**Date**: 2026-10-21
**Status**: Accepted
**Deciders**: pelikhan, Copilot (PR #70973)

---

### Context

The agent prompt is rendered and placeholder-substituted in the `activation` job. It is then uploaded as an artifact for the `agent` job. Workflow markdown can reference `${{ needs.<job>.outputs.<name> }}` or `${{ needs.<job>.result }}` for custom jobs. Some of those jobs run after `activation` but before `agent`; for example, `release.md` referenced `needs.release.outputs.release_id`. Activation cannot read those outputs. The expression remained unresolved in the prompt, and [release run 39690606538](https://github.com/github/gh-aw/actions/runs/39690606538) produced an unresolved-expression warning. The existing `needs-consistency` lint rule checked job-level `needs` graphs, but it did not check whether prompt interpolations referenced jobs that are unavailable to the job that renders the prompt.

### Decision

The compiler defers direct references to declared custom-job outputs (`needs.<job>.outputs.<declared>`) and `needs.<job>.result` when the producer runs between activation and the agent job. Eligible producers are not in `on.needs`, are not built-in jobs, and must not depend directly or transitively on `agent` or a post-agent job. This constraint prevents adding an agent dependency that would create a cycle. In activation, each reference becomes a deterministic placeholder `__GH_AW_DEFERRED_<sha256-prefix>__`. The agent job substitutes the real values in a "Resolve deferred prompt expressions" step, using the existing `substitute_placeholders.cjs` script. `validate_prompt_placeholders.sh` accepts only the deferred placeholders the compiler declares in `GH_AW_DEFERRED_PLACEHOLDERS`.

References inside compound expressions or `{{#if}}`/`{{else if}}` template conditionals are not rewritten, because that would change rendering semantics. The `needs-consistency` AST rule reports references unavailable during prompt rendering, including unsupported post-activation references, and undeclared outputs. In compiled lock files, the rule checks generated late bindings against the agent job's dependencies rather than activation's.

### Alternatives Considered

#### Alternative 1: Lint-only — forbid prompt references to post-activation jobs

Report every prompt reference to a job that runs after activation, and require authors to read the data from files staged for the agent. This PR does that for `release.md`. This alternative keeps the compiler simpler and avoids a second substitution pass. It was not chosen as the only mechanism because direct output references are a natural authoring pattern, and the compiler can resolve them safely. Several existing workflows (purelock, windows-grower, daily-windows-defender-scan, daily-astrostylelite-markdown-spellcheck) already depend on this pattern.

#### Alternative 2: Move prompt rendering entirely into the agent job

Render and substitute the whole prompt in the agent job, where all upstream outputs are available. This removes the timing problem for every expression. It was rejected because activation currently owns prompt assembly, validation, and artifact upload. Moving all of it would be a much larger change to the compiled job graph and its security boundaries, and it would delay validation failures until after agent setup.

#### Alternative 3: Require producers to move before activation via `on.needs`

Require authors to make producer jobs run before activation. This remains the guidance the lint diagnostic gives for independent producers. It cannot be applied in general, because many producers need activation outputs or must run after activation-level gating.

### Consequences

#### Positive
- Direct references to eligible custom-job outputs and results in prompts resolve in the agent job, without authors changing their workflows.
- Prompt references to unavailable jobs or undeclared outputs become lint diagnostics instead of silent empty strings or runtime warnings.
- Placeholder names are deterministic (SHA-256 of the expression), so lock files are stable across recompiles.

#### Negative
- Prompt substitution now happens in two places (activation and agent). This adds a compiler code path, generated steps, and an allowlist exception in placeholder validation.
- Compound expressions and template conditionals that reference post-activation outputs are still unsupported. Authors must restructure them after a lint diagnostic.
- The lint rule identifies late bindings by matching the generated step name ("Resolve deferred prompt expressions"), script content, and `GH_AW_DEFERRED_` environment bindings. This couples the linter to the compiler's output format; changes to that format must update the detection logic together.

#### Neutral
- Generated workflow lock files reflect the additional substitution step where needed. `release.md` was changed to read release metadata from staged JSON instead of interpolating the release job output.
- The workflow-linters reference documentation was updated to describe the extended `needs-consistency` rule.
