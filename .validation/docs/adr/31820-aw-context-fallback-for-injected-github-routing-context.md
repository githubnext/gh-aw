## ADR-31820: `aw_context` Fallback for Injected GitHub Routing Context in Agent Prompt

**Date**: 2026-05-13
**Status**: Draft
**Deciders**: Unknown (auto-generated from PR diff)

---

### Part 1 — Narrative (Human-Friendly)

#### Context

The embedded `<github-context>` block of the agent prompt surfaces routing identifiers (`issue-number`, `discussion-number`, `pull-request-number`, `comment-id`) so the agent knows which entity it is operating on. Historically these values were bound to dedicated env vars sourced strictly from the GitHub Actions `github.event.*` payload (e.g. `GH_AW_GITHUB_EVENT_ISSUE_NUMBER: ${{ github.event.issue.number }}`). For runs triggered through centralized slash-command routing or `workflow_call` paths, the native `github.event.*` entity payload is often absent — the routing job passes the relevant identity via an `aw_context` workflow input instead. The net effect was that prompt placeholders rendered empty in exactly the entry points that depend on `aw_context`, even though the metadata was available, costing the agent its routing context.

#### Decision

We will inject GitHub routing context fields into the prompt through a virtual `github.aw.context.*` namespace that the compiler resolves to a fallback expression: `github.event.X || (fromJSON(github.event.inputs.aw_context || '{}').item_type == '<entity>' && fromJSON(...).item_number)`. The compiler emits each compound expression once as a hashed env var (`GH_AW_EXPR_<HASH>`) and substitutes it into the prompt template via the existing placeholder pipeline. An `item_type` guard is required on number fields so that an `aw_context` describing an issue does not leak its number into the pull-request-number slot.

#### Alternatives Considered

##### Alternative 1: Per-event-source workflow variants

Generate distinct workflow lock files (and prompt context blocks) for native-event invocations vs. slash/`workflow_call` invocations, so each variant could hard-code the appropriate source. Rejected because it doubles the compiled-workflow surface area, complicates `*.lock.yml` golden fixtures, and forks the routing logic that the unified compiler is explicitly designed to centralize.

##### Alternative 2: Resolve `aw_context` at runtime inside the prompt script

Drop the env-var substitution path entirely and have the activation step parse `aw_context` in JavaScript / shell, then write the resolved values into the prompt file in-place. Rejected because it leaves GitHub Actions' native expression engine (which already evaluates `github.event.*` for free) and shifts the resolution into agent-side code, increasing the chance of drift between what the prompt says and what other steps see.

##### Alternative 3: Unguarded `||` fallback without `item_type` checks

A simpler `github.event.issue.number || fromJSON(aw_context).item_number` form. Rejected because `item_number` is shared across entity kinds in `aw_context`; without an `item_type` guard, an issue-routed run would falsely populate the pull-request-number field (and vice versa), causing the agent to act on the wrong entity.

#### Consequences

##### Positive

- Slash-command and `workflow_call` invocations now produce fully-populated routing context in the agent prompt, restoring the entity-awareness the agent needs.
- Routing identity flows through a single virtual namespace (`github.aw.context.*`), so future fields (assignees, labels, etc.) extend the same mechanism without re-plumbing every workflow.
- Resolution stays inside GitHub Actions expressions, preserving the existing env-var-substitution placeholder model used elsewhere in the compiled workflow.

##### Negative

- Every compiled workflow now carries four extra `GH_AW_EXPR_<HASH>` env vars per agent step, each with a verbose double-`fromJSON` expression, increasing the size and visual noise of `*.lock.yml` files.
- The `GH_AW_EXPR_<HASH>` naming hides what each var represents; debugging a misbehaving prompt now requires cross-referencing the hash back to its source expression.
- A subtle correctness coupling is introduced between the agent prompt template and the `aw_context` schema fields (`item_type`, `item_number`, `comment_id`); a future rename of these keys must update both the routing emitter and the prompt-side fallback.

##### Neutral

- All existing `*.lock.yml` golden fixtures had to be regenerated; future PRs touching the prompt context will continue to see widespread golden-file churn.
- A regression test (`compiler_string_api_test.go`) now asserts the fallback expression remains present in the embedded context block, so accidental reverts will fail CI.

---

### Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

#### Prompt Context Injection

1. The embedded GitHub context prompt **MUST** source `issue-number`, `discussion-number`, `pull-request-number`, and `comment-id` from a fallback expression that prefers `github.event.*` and falls back to `aw_context`-derived values.
2. Entity number fields (`issue-number`, `discussion-number`, `pull-request-number`) **MUST** guard the `aw_context` branch with an `item_type` equality check for the matching entity kind.
3. The prompt template **MUST NOT** bind these routing fields to env vars that read only from `github.event.*` without an `aw_context` fallback.
4. The prompt template **SHOULD** route routing-context fields through the `github.aw.context.*` virtual namespace rather than emitting bespoke per-field expressions.

#### Compiler-Emitted Env Vars

1. The compiler **MUST** emit each distinct fallback expression as a single env var keyed by a deterministic hash (e.g. `GH_AW_EXPR_<HASH>`) and reuse the same key for identical expressions across steps within a workflow.
2. The compiler **MUST** include each emitted `GH_AW_EXPR_<HASH>` env var in the `substitutions` map passed to `substitutePlaceholders` so that the prompt placeholder resolves.
3. The compiler **SHOULD NOT** emit dedicated per-entity env vars (e.g. `GH_AW_GITHUB_EVENT_ISSUE_NUMBER`) for routing identity that the `github.aw.context.*` namespace already covers.

#### `aw_context` Schema Contract

1. The routing emitter writing `aw_context` **MUST** populate `item_type` with one of `issue`, `discussion`, or `pull_request` whenever it populates `item_number`.
2. The routing emitter **MUST** populate `comment_id` only when the originating event references a comment.
3. Callers reading `aw_context` from the prompt expression layer **MUST** treat a missing or malformed `aw_context` input as equivalent to the empty object `{}` (i.e. via `fromJSON(github.event.inputs.aw_context || '{}')`).

#### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25772016504) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
