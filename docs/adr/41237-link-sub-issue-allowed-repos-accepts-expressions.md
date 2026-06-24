# ADR-41237: Accept GitHub Actions Expressions for `link-sub-issue` `allowed-repos`

**Date**: 2026-06-24
**Status**: Draft
**Deciders**: Unknown (AI-drafted from PR #41237)

---

### Context

The `safe-outputs` configuration supports cross-repository targeting through a `target-repo` slug and an `allowed-repos` allowlist. Other cross-repo safe outputs — `dispatch-workflow` and `push-to-pull-request-branch` — already let `allowed-repos` be a GitHub Actions expression (e.g. `${{ inputs['allowed-repos'] }}`) so the allowlist can be resolved dynamically at runtime from workflow inputs or secrets. `link-sub-issue` was the lone exception: its `allowed-repos` field was parsed with `ParseStringArrayFromConfig`, which only accepts a literal YAML array, so dynamic cross-repo targeting was impossible for sub-issue linking.

### Decision

We will parse `link-sub-issue`'s `allowed-repos` with the expression-aware `ParseStringArrayOrExprFromConfig` instead of `ParseStringArrayFromConfig`, and widen the JSON schema for that field from `type: array` to a `oneOf` accepting either an array of `owner/repo` slugs or a single GitHub Actions expression string. This makes `link-sub-issue` consistent with the other cross-repo safe outputs and unblocks dynamic allowlists driven by workflow inputs.

### Alternatives Considered

#### Alternative 1: Keep array-only, require hardcoded repositories

Leave `allowed-repos` as a literal array and require authors to enumerate every allowed repository statically. Rejected because it prevents parameterized, reusable workflows (e.g. a shared workflow whose target set varies per caller) and leaves `link-sub-issue` inconsistent with `dispatch-workflow` and `push-to-pull-request-branch`.

#### Alternative 2: Introduce a separate expression-only field

Add a distinct config key (e.g. `allowed-repos-expr`) alongside the array form. Rejected because it duplicates surface area, diverges from the established pattern used by sibling safe outputs, and pushes the array-vs-expression branching onto every workflow author rather than handling it in one shared parser.

### Consequences

#### Positive
- `link-sub-issue` now matches the cross-repo expression behavior of `dispatch-workflow` and `push-to-pull-request-branch`, removing a special case.
- Allowlists can be resolved dynamically from `inputs`/`secrets`, enabling parameterized reusable workflows.
- Reuses the existing shared parser (`ParseStringArrayOrExprFromConfig`), so no new bespoke parsing logic is introduced.

#### Negative
- An expression value cannot be statically validated as `owner/repo` slugs at compile time; correctness of the resolved allowlist now depends on runtime input.
- The schema is more complex (`oneOf` array/string) and the pattern `^\$\{\{.*\}\}$` only checks expression shape, not the resolved contents.

#### Neutral
- Tool descriptions surfaced to the agent now include an "allowed repositories" constraint line when `allowed-repos` is set (`tool_description_enhancer.go`).
- A `repo` field was added to the `link_sub_issue` safe-output tool schema for cross-repo selection, consistent with the configured allowlist.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
