# ADR-33610: Collector-Time `allowed-branches` Enforcement for `create-pull-request` Safe Output

**Date**: 2026-05-20
**Status**: Draft
**Deciders**: Unknown (draft generated from PR #33610 diff)

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `create-pull-request` safe output already supports `allowed-base-branches` so authors can restrict which base branches an agent may target at runtime, but there is no symmetric control over the **source** branch the PR is opened from. Some repositories enforce source-branch naming conventions (e.g. `feature/*`, `release/*`) and want those conventions to be enforced even when an agent generates the PR. Branch policy violations should also surface as soon as possible — ideally when the agent records the safe output through the MCP tool — so the agent gets actionable, immediate feedback rather than a late failure during the apply step.

### Decision

We will add an `allowed-branches` field to `safe-outputs.create-pull-request` that accepts an array of glob patterns (or a GitHub Actions expression resolving to a comma-separated list, consistent with `allowed-base-branches`). We will enforce the policy **at safe-output collection time** inside `safe_outputs_handlers.cjs`: when the agent emits a `create_pull_request` intent, the effective branch (agent-provided `branch`, or the resolved current checkout branch when omitted) MUST match one of the configured patterns, otherwise the handler returns an MCP error and the safe output is not recorded. The compiler (`pkg/workflow`) is extended to plumb the field as `allowed_branches` into the runtime handler config and to treat it as a templatable expression-array field alongside `labels`, `allowed-repos`, and `allowed-base-branches`.

### Alternatives Considered

#### Alternative 1: Enforce only at apply-time in the PR-creation workflow step

The branch policy could be checked only in the post-agent step that actually opens the PR. This was rejected because the agent would not see the rejection until much later in the pipeline (after the entire turn has completed), which wastes work and obscures the cause. Collector-time enforcement returns a structured MCP error the agent can react to inside the same turn.

#### Alternative 2: Reuse `allowed-base-branches` for source branches

We considered overloading the existing `allowed-base-branches` field with a separate boolean to also constrain source branches. This was rejected because base and source branches have different semantics (target vs origin) and conflating them would make the config harder to reason about and hide what is being constrained.

#### Alternative 3: Use regex patterns instead of glob patterns

A regex field would be more expressive. This was rejected for consistency: every other branch/path constraint in safe-outputs (`allowed-base-branches`, `allowed-files`, `excluded-files`, `protected-files`) uses glob syntax via the existing `globPatternToRegex` helper. A regex outlier would force workflow authors to learn two pattern dialects.

### Consequences

#### Positive
- Repository-level source-branch naming conventions can be enforced by workflow configuration rather than by relying on the agent's prompt obedience.
- Symmetric with `allowed-base-branches`, so the mental model is "two parallel branch policies (source + base)" and authors do not have to learn a new shape.
- Errors are surfaced inside the agent's turn (as an MCP error containing the configured patterns), enabling self-correction without a full pipeline failure.
- Expression-array support means the allow-list can be parameterized by `workflow_call` inputs (e.g. `${{ inputs['allowed-branches'] }}`).

#### Negative
- Workflow authors now have two distinct branch policy fields (`allowed-branches` and `allowed-base-branches`) and must understand the difference. Confusing the two would silently fail to constrain the intended branch.
- The runtime copy of `safe_outputs_handlers.cjs` and the compiler config plumbing must stay in sync; drift between the two would result in collector-time enforcement silently disappearing. This is partially mitigated by the new `TestSafeOutputsToolsJSONInSync` test, but the handler code itself is not similarly guarded.
- Collector-time enforcement runs in the agent's environment using its inferred branch; an agent that omits `branch` and runs from an unexpected checkout could still pass the policy if that checkout happens to match — the policy is only as strong as the branch resolution it sees.

#### Neutral
- Two copies of `safe_outputs_tools.json` (`actions/setup/js/` and `pkg/workflow/js/`) must be updated together when the tool description changes; a new test enforces tool-name parity between them.
- The `branch` tool-input description now references the workflow configuration, so generated MCP tool schemas vary slightly depending on which safe-output features are enabled.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Configuration Surface

1. The `safe-outputs.create-pull-request` config **MUST** accept an optional `allowed-branches` field.
2. The `allowed-branches` field **MUST** accept either (a) an array of strings, or (b) a single GitHub Actions expression string of the form `${{ ... }}` that resolves to a comma-separated list.
3. Implementations **MUST** treat each entry as a glob pattern; bare strings without `*` **MUST** be matched as exact branch names.
4. Implementations **MUST NOT** reject configurations where `allowed-branches` is absent or empty — in that case no branch policy is applied.

### Collector-Time Enforcement

1. When `allowed-branches` is non-empty, the `create_pull_request` MCP handler **MUST** resolve the effective branch from (in order): the agent-supplied `branch` field if present and not equal to the configured base, otherwise the branch detected from the current checkout (`GITHUB_HEAD_REF` / `GITHUB_REF_NAME`).
2. The handler **MUST** match the resolved branch against the configured patterns using the shared `globPatternToRegex` helper in `pathMode: true, caseSensitive: true` mode.
3. If the resolved branch does not match any configured pattern, the handler **MUST** return a structured MCP error with `isError: true`, `result: "error"`, and an `error` message that includes the rejected branch and the configured patterns.
4. On rejection, the handler **MUST NOT** call `appendSafeOutput` — the PR intent **MUST NOT** be recorded.
5. The handler **MUST** apply `allowed-branches` enforcement before any intent-probe validation (`validateCreatePullRequestIntent`), so policy violations short-circuit cheaper-to-detect errors.

### Compiler Plumbing

1. `CreatePullRequestsConfig` **MUST** expose `AllowedBranches []string` with YAML tag `allowed-branches,omitempty`.
2. The list of templatable expression-array fields for `create-pull-request` **MUST** include `allowed-branches` alongside `labels`, `allowed-repos`, and `allowed-base-branches`.
3. The handler config builder **MUST** emit `allowed_branches` via `AddTemplatableStringSlice` when `AllowedBranches` is non-empty, and **MUST NOT** emit the key when it is empty.
4. The JSON-schema in `pkg/parser/schemas/main_workflow_schema.json` **MUST** describe `allowed-branches` with a `oneOf` of either an array of strings or an expression string matching `^\$\{\{.*\}\}$`.

### Tooling Description Parity

1. The `branch` parameter description in both `actions/setup/js/safe_outputs_tools.json` and `pkg/workflow/js/safe_outputs_tools.json` **MUST** state that the branch must be provided and match `allowed-branches` when the workflow configures that field.
2. The compiler-embedded and runtime copies of `safe_outputs_tools.json` **MUST** declare the same tool names in the same order, as verified by `TestSafeOutputsToolsJSONInSync`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies every **MUST** and **MUST NOT** requirement above. In particular, an implementation that accepts the `allowed-branches` field but does not reject non-matching branches at MCP collection time is **non-conformant**.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26187066394) workflow. The PR author must review, complete (especially the Deciders field), and finalize this document before the PR can merge.*
