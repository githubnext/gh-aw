# ADR-29089: Compile `on.workflow_run.conclusion` Filter into Job `if` Condition at Build Time

**Date**: 2026-04-29
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

GitHub Actions' `workflow_run` trigger fires for any conclusion value (`success`, `failure`, `cancelled`, etc.) when `types: [completed]` is specified — there is no native way to filter by conclusion in the `on:` block. Users who want to react only to failed runs must write a `if:` expression guard, and that guard requires an additional `github.event_name != 'workflow_run'` prefix so the condition remains transparent when the workflow fires from other events (e.g., `workflow_dispatch`). This two-part guard is subtle, error-prone to hand-write, and inconsistent with the simpler `on.deployment_status.state` pattern the compiler already handles.

### Decision

We will add `on.workflow_run.conclusion: string | string[]` as a recognized frontmatter field and compile it at build time into a guarded GitHub Actions expression that is AND-merged with any existing `if:` condition on the activation job. The generated guard takes the form `github.event_name != 'workflow_run' || (github.event.workflow_run.conclusion == '<value>')`, mirroring the pattern already established for `on.deployment_status.state`. The raw `conclusion:` field is commented out in the compiled YAML `on:` section to make clear it is not a native GitHub Actions key.

### Alternatives Considered

#### Alternative 1: Require Users to Write Manual `if:` Conditions

Users would write `if: github.event.workflow_run.conclusion == 'failure'` directly in frontmatter. This was rejected because it omits the event_name guard, which silently breaks workflows that also respond to `workflow_dispatch` or other non-`workflow_run` triggers — the conclusion expression evaluates to false for those events, preventing the workflow from running at all. The correct two-part guard is non-obvious and would require documentation warnings; the compiler can generate it correctly every time instead.

#### Alternative 2: Runtime Filtering in the Agent Context Script

The `aw_context.cjs` setup action could check `workflow_run.conclusion` at runtime and exit early if it doesn't match. This was rejected because it still consumes GitHub Actions runner minutes to start the job before aborting, and it moves filtering logic into the runtime layer rather than the declarative compilation layer where similar patterns already live (`deployment_status.state`).

### Consequences

#### Positive
- Users declare intent declaratively (`conclusion: failure`) without needing to understand GitHub Actions expression syntax or the event_name guard pattern.
- Consistent with the existing `on.deployment_status.state` compiler feature, keeping the frontmatter DSL coherent and the compilation logic concentrated in one place.
- The event_name guard is generated correctly and automatically, eliminating a class of silent bugs.
- `workflow_run_conclusion` is propagated to child workflows via `aw_context`, making the triggering conclusion visible to the agent without reading the raw event payload.

#### Negative
- Adds stateful flag complexity (`inWorkflowRun`, `inWorkflowRunConclusionArray`) to `commentOutProcessedFieldsInOnSection`, which already manages multiple section flags that can interact unexpectedly.
- The set of valid conclusion values (`success`, `failure`, `cancelled`, `skipped`, `timed_out`, `action_required`, `neutral`, `stale`) is documented only in the user-facing docs; if GitHub adds new conclusion values, the documentation must be updated manually.
- The compiler now silently transforms the `conclusion:` key — users inspecting compiled YAML must understand it has been moved to a job condition.

#### Neutral
- The `workflow_run_conclusion` field is also added to `awInfo` and the OTEL spans, which slightly increases the payload size for all `workflow_run`-triggered runs.
- Tests for the new logic are in a dedicated file (`workflow_run_conclusion_test.go`), consistent with how the deployment_status tests are organized.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Frontmatter Compilation

1. When `on.workflow_run.conclusion` is present in the workflow frontmatter, the compiler **MUST** convert it into a GitHub Actions job `if` expression of the form `github.event_name != 'workflow_run' || (<conclusion-expr>)`.
2. When `on.workflow_run.conclusion` is a string, `<conclusion-expr>` **MUST** be `github.event.workflow_run.conclusion == '<value>'`.
3. When `on.workflow_run.conclusion` is an array of strings, `<conclusion-expr>` **MUST** be the individual equality checks joined with ` || `.
4. The generated condition **MUST** be AND-merged with any existing `if:` expression from the frontmatter using the form `(<existing>) && (<conclusion-condition>)`.
5. The compiler **MUST** comment out the `conclusion:` key (and any array items beneath it) in the compiled YAML `on:` section, appending an explanatory comment.
6. The compiler **MUST NOT** comment out sibling keys of `workflow_run` (such as `workflows:` or `types:`) as a side effect of processing `conclusion:`.
7. If `on.workflow_run.conclusion` is absent or empty, the compiler **MUST NOT** add any conclusion condition to the job `if` expression.

### Agent Context Propagation

1. When a workflow is triggered by a `workflow_run` event, `buildAwContext()` **MUST** populate `workflow_run_conclusion` with the value of `context.payload.workflow_run.conclusion`.
2. For all other event types, `workflow_run_conclusion` **MUST** be set to an empty string.
3. Implementations **MUST** propagate `workflow_run_conclusion` to child workflows via `workflow_call` inputs, following the same pattern as `deployment_state`.
4. The `workflow_run_conclusion` field **SHOULD** be included in `awInfo` so agents can access the conclusion without reading the raw event payload.

### OTEL Instrumentation

1. When `workflow_run_conclusion` is non-empty, implementations **MUST** include a `gh-aw.workflow_run.conclusion` attribute on both the setup span and the conclusion span.
2. Implementations **MUST** read `workflow_run_conclusion` from `awInfo.workflow_run_conclusion` first, falling back to `awInfo.context.workflow_run_conclusion` for child workflows that received it via propagation.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25108336032) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
