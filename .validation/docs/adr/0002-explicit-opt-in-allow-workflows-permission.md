# ADR-0002: Explicit Opt-In for GitHub App workflows:write Permission via allow-workflows Field

**Date**: 2026-04-11
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw safe-outputs system mints scoped GitHub App tokens to push changes to pull request branches and create pull requests. When the `allowed-files` configuration targets `.github/workflows/` paths, GitHub requires the `workflows:write` permission on the minted token — a permission only available to GitHub Apps, not to `GITHUB_TOKEN`. The compiler's `ComputePermissionsForSafeOutputs` function previously had no knowledge of this requirement, leaving users unable to push workflow files through safe-outputs without resorting to fragile post-compile `sed` injection as a workaround. The question was: should the compiler auto-infer `workflows:write` from `allowed-files` patterns, or require an explicit opt-in field?

### Decision

We will add an explicit boolean field `allow-workflows` on both `create-pull-request` and `push-to-pull-request-branch` safe-outputs configurations. When set to `true`, the compiler adds `workflows: write` to the GitHub App token permissions computed by `ComputePermissionsForSafeOutputs`. The field is intentionally not auto-inferred from `allowed-files` patterns, keeping the elevated permission visible and auditable in the workflow source. Compile-time validation enforces that `safe-outputs.github-app` is configured (with non-empty `app-id` and `private-key`) whenever `allow-workflows: true` is present, because `workflows:write` cannot be granted via `GITHUB_TOKEN`.

### Alternatives Considered

#### Alternative 1: Auto-infer workflows:write from allowed-files patterns

Detect at compile time whether any `allowed-files` pattern matches a `.github/workflows/` path (e.g., `strings.HasPrefix(pattern, ".github/workflows/")`), and automatically add `workflows: write` to the token permissions when such a pattern is found. This eliminates a configuration step for the user. It was rejected because it makes an elevated, GitHub App-only permission appear silently: a reviewer reading a workflow file would have no indication that `workflows:write` is being requested unless they also inspected the `allowed-files` list and understood the compiler's inference rules. Explicit opt-in makes the elevated permission a first-class, auditable fact in the workflow source.

#### Alternative 2: Grant workflows:write globally for all safe-outputs operations

Always include `workflows: write` in the safe-outputs GitHub App token, regardless of whether any workflow files are being pushed. This simplifies the permission model by eliminating per-handler configuration. It was rejected because it violates the principle of least privilege: the vast majority of safe-outputs deployments do not push workflow files, and granting `workflows:write` to those tokens unnecessarily expands the blast radius of a token compromise. Scoped permissions per handler are a deliberate security property of the safe-outputs system.

#### Alternative 3: Keep the post-compile sed-injection workaround as the documented path

Document the existing workaround (injecting `permission-workflows: write` via `sed` in a post-compile step) as the official solution for users who need to push workflow files. This requires no compiler changes. It was rejected because sed injection is inherently fragile, version-sensitive, and bypasses compile-time safety checks. It also produces compiled output that diverges from what the compiler would generate from the source, breaking the compiler's reproducibility guarantee.

### Consequences

#### Positive
- The elevated `workflows:write` permission is explicit and visible in the workflow source — security reviewers can see it at a glance without needing to understand compiler inference rules.
- Compile-time validation prevents misconfiguration: a workflow with `allow-workflows: true` but no GitHub App configured fails at compile time with a clear error message, rather than silently generating a broken workflow.
- The implementation is consistent with the existing staged-mode pattern: `allow-workflows: true` has no effect when the handler is in staged mode, since staged mode does not mint real tokens.
- Eliminates the fragile post-compile `sed` workaround that was the only previous path for pushing workflow files.

#### Negative
- Users who need to push workflow files must add an extra field (`allow-workflows: true`) to their configuration, even when the need is obvious from their `allowed-files` patterns. This is a deliberate UX trade-off for auditability.
- The `validateSafeOutputsAllowWorkflows` validation function must be called explicitly from `validateWorkflowData` in `compiler.go`. If future validation functions are not wired up in the same way, they will be silently skipped.

#### Neutral
- The `allow-workflows` field is defined separately on each handler (`create-pull-request` and `push-to-pull-request-branch`) rather than as a single top-level safe-outputs flag. This allows per-handler control but means users targeting both handlers must set the field in both places.
- The JSON schema is updated for both handler schemas, keeping schema-based tooling (IDE validation, linting) in sync with the Go implementation.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Permission Computation

1. Implementations **MUST** add `workflows: write` to the computed GitHub App token permissions for a safe-outputs handler if and only if that handler's `allow-workflows` field is `true` and the handler is not in staged mode.
2. Implementations **MUST NOT** add `workflows: write` to token permissions when `allow-workflows` is absent or `false`.
3. Implementations **MUST NOT** add `workflows: write` to token permissions when the handler is in staged mode (i.e., `Staged: true`), even if `allow-workflows: true` is set.
4. Implementations **MUST NOT** infer the need for `workflows: write` from `allowed-files` patterns or any other implicit signal — the only authoritative source is the explicit `allow-workflows` field.

### Compile-Time Validation

1. Implementations **MUST** validate at compile time that `safe-outputs.github-app` is configured with a non-empty `app-id` and non-empty `private-key` whenever any safe-outputs handler has `allow-workflows: true`.
2. Implementations **MUST** produce a compile error if `allow-workflows: true` is present without a valid GitHub App configuration.
3. Implementations **SHOULD** include the handler name(s) in the compile error message to help users identify which handler(s) triggered the validation failure.
4. Implementations **SHOULD** include a configuration example in the compile error message showing how to add a GitHub App configuration.

### Schema and Documentation

1. Implementations **MUST** declare `allow-workflows` as an optional boolean field (default: `false`) in the JSON schema for both the `create-pull-request` and `push-to-pull-request-branch` handler objects.
2. Implementations **SHOULD** document that `allow-workflows` requires a GitHub App and cannot be used with `GITHUB_TOKEN`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically: the `workflows: write` permission is added to GitHub App token permissions when and only when an active (non-staged) safe-outputs handler has `allow-workflows: true`, and compilation fails with a clear error when `allow-workflows: true` is set without a valid GitHub App configuration. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24280835716) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
