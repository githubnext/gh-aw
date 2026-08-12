# Formal Notes: security-architecture-spec-validation.md

**Last formalized**: 2026-08-12-15-52-00
**Notation**: TLA+ / Z3-style guard conjunction / F*
**Issue**: created via safe-output (number assigned post-processing)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `WorkflowRunRepoSafetyCondition` | Compiled `if:` guard requires repo-id match and not-fork when event is workflow_run (T-PM-005) |
| P2 | `WorkflowRunRepoSafetyOnlyAppliesWhenTriggerPresent` | Guard only injected when `hasWorkflowRunTrigger` detects the trigger |
| P3 | `WorkflowRunRequiresNonEmptyWorkflowsField` | `on.workflow_run.workflows` must be non-empty (string/[]string/[]any forms) |
| P4 | `WorkflowRunBranchRestrictionModeSensitive` | Missing branches -> error in strict mode, warning otherwise (T-PM-003) |
| P5 | `WorkflowRunBranchRestrictionSatisfiedNoOp` | Present branches never triggers strict/warn path |
| P6 | `NoWorkflowRunTriggerIsNoOp` | Non-workflow_run triggers skip validation entirely |
| P7 | `DefaultGitHubTokenPrecedence` | Custom token wins; else 3-tier secret fallback (MCP_SERVER/GH_AW/GITHUB_TOKEN) (T-PM-007) |
| P8 | `SafeOutputGitHubTokenPrecedence` | Custom token wins; else 2-tier safe-output fallback chain (T-PM-007) |
| P9 | `TokenChainsAreDistinctByJobRole` | Tool-token chain includes MCP-server secret; safe-output chain excludes it (write isolation) |
| P10 | `StrictModeIsPerCompilerInstance` | `SetStrictMode` deterministically toggles `strictMode` field |
| P11 | `BashRestrictionWildcardSafe` | nil/wildcard/false/empty-list boundary matrix (carried from CTR notes) |

## Key Invariants

- workflow_run repository-safety `if:` guard is only injected when `on.workflow_run` is present in frontmatter (map or exact string form).
- Strict mode (`Compiler.strictMode`, set via `SetStrictMode`) converts missing workflow_run branch restrictions from a warning into a hard compile error — this closes the T-PM-003 evidence gap.
- Repository-ID + not-fork guard (`buildWorkflowRunRepoSafetyCondition`) closes T-PM-005.
- GitHub token precedence chains differ by job role: tool/agent tokens include `GH_AW_GITHUB_MCP_SERVER_TOKEN` first; safe-output tokens omit it, preserving write-scope isolation — closes T-PM-007.

## Edge Cases Identified

- `on: workflow_run` as a bare string (not map form) is NOT detected as a workflow_run trigger by `hasWorkflowRunTrigger`'s string-equality branch when combined with other trigger keys — documented as current, non-obvious behavior in the test suite.
- Empty vs. whitespace-only `workflow_run.workflows` values must both be rejected.
- `SetStrictMode` must be idempotent under repeated calls with the same value.

## Notes for Future Runs

- The generated test suite targets internal (unexported) functions (`hasWorkflowRunTrigger`, `buildWorkflowRunRepoSafetyCondition`, `validateWorkflowRunBranches`, `hasNonEmptyWorkflowRunWorkflows`, `getEffectiveGitHubToken`, `getEffectiveSafeOutputGitHubToken`). A follow-up PR should add small test-only exported wrappers (or move the suite into `package workflow`) before it can compile as-is.
- T-PM-001, T-PM-002, T-PM-004, T-PM-006 were already evidenced in the base spec (PM-01/PM-02 defaults, PM-08 fork protection, PM-10/PM-11 RBAC) — not re-derived this run.
- Sandbox Isolation (T-SI-001..007) and Threat Detection (T-TD-002..007) remain the next-highest-priority gaps per the compliance matrix; a future run should target those once exposed via testable Go APIs.
- T-GH-047 to T-GH-060 (companion MCP access-control spec) remain out of scope for this document — see `specs/github-mcp-access-control-compliance/README.md` notes instead.
