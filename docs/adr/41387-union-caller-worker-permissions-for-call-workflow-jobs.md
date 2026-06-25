# ADR-41387: Union Caller and Worker Permissions for Call-Workflow Jobs

**Date**: 2026-06-25
**Status**: Draft
**Deciders**: pelikhan (via copilot-swe-agent)

---

### Context

The gh-aw compiler generates `call-<worker>` jobs in GitHub Actions lockfiles for `workflow_call` patterns. Previously, these jobs carried only the caller workflow's declared permissions; the compiler emitted a warning when the worker required more scope but never modified the caller's envelope. GitHub validates reusable workflow calls at startup and rejects the run entirely if any worker job requests a permission level greater than the calling job grants — a hard platform constraint. This mismatch caused 100% `startup_failure` on `Smoke Call Workflow` starting 2026-06-23, because the caller declared `contents: read` and `pull-requests: read` while the worker required `issues: write` and `pull-requests: write`.

### Decision

We will compute the effective permission envelope for every `call-<worker>` job as the **union of the caller's declared permissions and the worker's job-level permissions**. Worker permissions are extracted from the worker's lockfile (or same-batch compilation target), cloned from the caller base, and merged in. The compiled `call-<worker>` job always holds a sufficient grant for the worker without requiring the caller's markdown to enumerate every permission the worker needs.

### Alternatives Considered

#### Alternative 1: Caller-only permissions with validation warnings (previous behavior)

The call-workflow job was assigned exactly the caller's declared permission envelope. The compiler used `findUncoveredWorkerPermissions` to detect mismatches and emitted a `warning` to stderr when the caller was insufficient, leaving it to the workflow author to manually widen their `permissions:` block. This was the original design rationale — the comment stated "callers control their own permission surface." It broke entirely when GitHub began enforcing the startup check strictly, as the caller was never updated to match the worker's needs.

#### Alternative 2: Worker permissions only (discard caller's declared envelope)

Replace the caller's permissions entirely with the worker's extracted permissions. This would always give the worker enough scope, but would silently expand the calling job's permission surface beyond what the caller intentionally declared, removing any caller-level constraint. It would also be incorrect for callers that legitimately need additional permissions not required by the worker (e.g. `statuses: write` for a separate step in the calling workflow itself).

### Consequences

#### Positive
- `call-<worker>` jobs no longer cause `startup_failure` when the worker requires broader permissions than the caller explicitly declared.
- The fix is backward-compatible: when the caller's declared permissions already cover the worker's needs, the effective permissions equal the caller's declared permissions (no change in output).
- Eliminates a whole class of manual annotation burden — workflow authors do not need to know every permission each worker requires.

#### Negative
- The compiled `call-<worker>` job's permission envelope is now wider than what the caller's markdown explicitly declares, which may surprise reviewers auditing lockfiles for least-privilege.
- Compiler warnings about permission mismatches are removed; the previous warning nudged authors to explicitly acknowledge scope requirements. That signal is gone.

#### Neutral
- The `findUncoveredWorkerPermissions` function remains in the codebase but is no longer used for the warning path — it could be repurposed for future audit tooling or removed.
- The `extractCallWorkflowPermissions` docstring and the `compiler_safe_output_jobs.go` comments are updated to reflect the new merge semantics; any tooling that reads those comments will need updating.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
