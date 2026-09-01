# ADR-56270: Add Per-Target Permission Controls to remove-labels

**Date**: 2026-08-27
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`safe-outputs.remove-labels` unconditionally requested both `issues:write` and `pull-requests:write` in the compiled workflow, regardless of whether the workflow needed to remove labels from issues, pull requests, or both. This violated the principle of least privilege and blocked GitHub Apps installed with issue-only scope from using the `remove-labels` safe output. The `add-comment` and `add-labels` tools had already solved this problem through per-target boolean flags (`issues` and `pull-requests`) introduced in #48877 and #49477. Consistency across the safe-outputs toolset and least-privilege security are both non-negotiable requirements for this system.

### Decision

We will add `issues` and `pull-requests` boolean fields to `RemoveLabelsConfig`, mirroring the pattern already established by `add-comment` and `add-labels`. Both default to `true` when omitted, preserving all existing behavior. Setting either to `false` omits the corresponding write permission from the compiled workflow and minted GitHub App token. Setting both to `false` is rejected at compile time. Permission computation is centralized in a new `buildRemoveLabelsPermissions` function, and a new `validateRemoveLabelsPermissions` validator is wired into the compiler's validation chain.

### Alternatives Considered

#### Alternative 1: Keep hardcoded dual permissions

Continue requesting both `issues:write` and `pull-requests:write` unconditionally. This requires zero new configuration surface and no validation logic. It was rejected because it actively breaks GitHub App installations scoped to issues only, and it grants more privilege than necessary to pure issue-labeling or pure PR-labeling workflows. The security cost outweighs the simplicity benefit.

#### Alternative 2: Split into separate `remove-issue-labels` and `remove-pr-labels` tools

Introduce two new safe-output tools with clear separation of concerns and no shared configuration. This would be maximally explicit about intent. It was rejected because it is a breaking change: all existing consumers that rely on `remove-labels` for both target types would need to migrate. The opt-out boolean approach achieves the same granularity without a migration burden, and it follows the pattern already established by the other safe-output tools.

### Consequences

#### Positive
- Workflows that only remove labels on issues no longer need `pull-requests:write`, and vice versa — least-privilege is achievable without changing the tool name.
- GitHub Apps installed with issue-only scope can now use `remove-labels` by setting `pull-requests: false`.
- The safe-outputs toolset is now internally consistent: `add-comment`, `add-labels`, and `remove-labels` all use the same per-target permission pattern.

#### Negative
- Two new boolean fields on `RemoveLabelsConfig` add configuration surface that must be kept in sync with the JSON schema and documentation.
- A new validation step must run at compile time to catch configurations that disable both permissions; missing this guard would silently produce an invalid permission set.

#### Neutral
- Existing workflows that omit both fields get identical behavior to today — nil fields are treated as `true` in `buildRemoveLabelsPermissions`.
- Tests for the new code follow the same table-driven pattern used across the package, so no new testing infrastructure is required.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
