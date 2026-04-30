# ADR-29230: Parameterize Safe-Output Policy Fields for `workflow_call` Reuse

**Date**: 2026-04-30
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `safe-outputs` configuration block for `create-pull-request` and `push-to-pull-request-branch` exposes two behavior-controlling fields: `protected-files` (policy: `blocked`, `allowed`, or `fallback-to-issue`) and `patch-format` (`am` or `bundle`). Both fields were previously restricted to compile-time literal enum values. This meant that reusable `workflow_call` workflows could not expose these as caller-controlled inputs — any team needing a different policy had to either fork the workflow file or maintain one copy per policy combination, creating duplicated YAML that diverged over time. The prior work in PR #29212 established the pattern of accepting GitHub Actions expression strings for list-valued constraints; this PR extends that pattern to enum-valued policy fields.

### Decision

We will allow `protected-files` (both string form and the `policy` key of its object form) and `patch-format` to accept GitHub Actions expression strings (matching `${{...}}`) in addition to their existing literal enum values. Literal values continue to be validated at compile time and rejected if unrecognized. Expression strings are passed through the compiler unchanged and emitted verbatim into the runtime handler configuration, where GitHub Actions evaluates them before the handler executes. The runtime handlers validate the resolved value and fail closed — `patch-format` returns an explicit error for unrecognized values; `protected-files` treats unrecognized values as `blocked` (most restrictive default).

### Alternatives Considered

#### Alternative 1: Duplicate Workflow Files per Policy Variant

Callers could maintain separate copies of the reusable workflow for each combination of `protected-files` and `patch-format` values. This was the status quo before this PR. It was rejected because it creates maintainability debt: every structural change to the base workflow must be propagated to all variants, and teams routinely drift their copies. The pattern does not scale as the number of callers grows.

#### Alternative 2: Accept Free-Form Strings with No Compile-Time Detection

The schema could be relaxed to accept any string (dropping enum enforcement entirely) and leave all validation to the runtime handler. This was not chosen because it would silently pass obviously invalid literal values (e.g., typos like `"blocekd"`) through compilation and only fail at job execution time, degrading the developer experience and breaking the "fail fast at compile time" guarantee that the rest of the system provides. Detecting expression strings by the `${{...}}` pattern preserves compile-time strictness for literals while enabling dynamic resolution for expressions.

#### Alternative 3: Introduce a New Dedicated `policy-expression` Field

A new field (e.g., `protected-files-expression`) could accept only expressions while the original field remains enum-only. This avoids changing the type of existing fields but doubles the surface area and requires callers to know which field to use in which context. It was rejected as unnecessarily complex when the pattern of "enum literal OR expression string" is both clear and consistent with how other expression-accepting fields work in GitHub Actions schemas.

### Consequences

#### Positive
- A single reusable `workflow_call` workflow can serve callers with different `protected-files` policies and `patch-format` requirements without duplicating files.
- Literal enum values retain compile-time validation and early error reporting; nothing regresses for existing non-expression usage.
- The fail-closed runtime behavior (`blocked` for unknown policy, explicit error for unknown patch format) ensures that expression misconfiguration cannot silently weaken security posture.
- The pattern is consistent with the expression-acceptance approach introduced in PR #29212 for list-valued constraints.

#### Negative
- Expression values are only validated at runtime, after GitHub Actions evaluates them. A typo in an input default (e.g., `default: blocekd`) will not be caught until the workflow runs.
- Two-stage validation logic (compile-time for literals, runtime for expressions) adds complexity to both the Go `validateStringEnumField` helper and the JavaScript handlers.
- The JSON schema now uses `oneOf` for fields that were previously a simple `enum`, which may complicate tooling (e.g., IDE autocomplete may not suggest enum values when the field contains an expression).

#### Neutral
- The `containsExpression` helper used to detect `${{...}}` patterns was already available in the codebase; this PR reuses it rather than introducing a new detection mechanism.
- Documentation in `docs/src/content/docs/reference/safe-outputs-pull-requests.md` was updated to show the `workflow_call` usage pattern with explicit examples.
- Schema changes apply symmetrically to both `create-pull-request` and `push-to-pull-request-branch` handler blocks.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Schema and Compile-Time Validation

1. The `protected-files` field and the `policy` sub-key of the `protected-files` object form **MUST** accept either a literal enum value (`blocked`, `allowed`, `fallback-to-issue`) or a GitHub Actions expression string matching the pattern `^\$\{\{.*\}\}$`.
2. The `patch-format` field **MUST** accept either a literal enum value (`am`, `bundle`) or a GitHub Actions expression string matching the pattern `^\$\{\{.*\}\}$`.
3. Literal enum values **MUST** be validated at compile time; unrecognized literal values **MUST** be rejected (removed from the config with a warning log) before the workflow is emitted.
4. GitHub Actions expression strings **MUST NOT** be enum-validated at compile time; they **MUST** be passed through to the runtime handler configuration verbatim.
5. The `validateStringEnumField` helper **MUST** use the `containsExpression` function to distinguish expressions from literal strings before applying enum validation.

### Runtime Handler Validation

1. Handlers **MUST** validate the resolved value of `patch_format` after GitHub Actions evaluates any expression; if the resolved value is not in `["am", "bundle"]`, the handler **MUST** return an error response and **MUST NOT** perform any git operations.
2. Handlers **MUST** validate the resolved value of `protected_files_policy` after evaluation; if the resolved value is not a recognized policy, the handler **MUST** treat it as `blocked` (fail closed, most restrictive).
3. Neither handler **MUST NOT** append any safe output to the output file when returning an error due to an invalid resolved field value.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement — in particular: allowing unrecognized literal values past compile time, performing git operations after a resolved-value validation failure, or treating an unrecognized `protected_files_policy` as anything other than `blocked` — constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25143053856) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
