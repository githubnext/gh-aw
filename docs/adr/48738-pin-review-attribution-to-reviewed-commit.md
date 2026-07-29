# ADR-48738: Automatic PR Review Attribution via Compiler-Injected Head SHA

**Date**: 2026-07-29
**Status**: Accepted
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Under `workflow_run` triggers, the safe-outputs runtime submits a PR review after a parent workflow completes. A new commit can land on the PR branch while the agent is running. When that happens, `pulls.get()` returns the new HEAD SHA, and the review is attributed to a commit the agent never actually reviewed — making inline comments appear fabricated, outdated, or misaligned.

### Decision

The compiler automatically captures the PR head SHA at trigger time and injects it into the compiled workflow as the `GH_AW_HEAD_SHA` environment variable, without requiring any user YAML configuration. The JS safe-outputs runtime reads `process.env.GH_AW_HEAD_SHA` in `submitReview()` and uses it as the `commit_id` argument to `pulls.createReview()` when present, falling back to the live PR head SHA otherwise.

The compiler emits the correct expression based on the workflow trigger:

| Trigger | `GH_AW_HEAD_SHA` value |
|---|---|
| `workflow_run` | `${{ github.event.workflow_run.head_sha }}` |
| `pull_request` | `${{ github.event.pull_request.head.sha }}` |
| `pull_request_target` | `${{ github.event.pull_request.head.sha }}` |
| Other triggers (push, schedule, issues, …) | Not injected; live PR head SHA is used |

No user YAML configuration is needed. The feature is transparent and automatic for all affected trigger types.

### Alternatives Considered

#### Alternative 1: Optional user-configurable `commit-id` YAML field

Add a `commit-id` field to `submit-pull-request-review` and `create-pull-request-review-comment` so callers can explicitly pass the reviewed SHA.

Why not chosen: Requires users to explicitly wire the SHA through their workflow YAML (e.g., from an eligibility job output). Forgetting to set `commit-id` leaves the race condition unaddressed. Also introduces multi-target ambiguity when `target: "*"` is configured, since a single handler-level SHA cannot be correct for multiple distinct PRs. The compiler knows the trigger type at compile time and can inject the right expression automatically.

#### Alternative 2: Freeze the PR HEAD SHA at handler initialization

Capture and freeze the SHA once when the handler initializes, before the review accumulation phase begins.

Why not chosen: A commit pushed after handler initialization but during the agent run would still cause attribution drift. The correct SHA is the one that was in place when the triggering workflow ran, not when the safe-outputs job starts.

### Consequences

#### Positive
- Reviews are attributed to the commit the agent actually reviewed, eliminating false "outdated" or misaligned inline comments.
- GitHub correctly marks the review as outdated when HEAD moves, rather than incorrectly attaching stale comments to the new commit.
- Zero user configuration required — all existing workflows benefit automatically after recompile.
- Multi-target (`target: "*"`) scenarios are unaffected because the SHA is read from an environment variable at runtime, not passed as a single handler-level config value.

#### Negative
- For workflows with triggers other than `workflow_run`, `pull_request`, and `pull_request_target`, the env var is not injected and the live PR head SHA is used (existing behavior). If drift is a concern for those triggers, a separate mechanism would be needed.

#### Neutral
- The `GH_AW_HEAD_SHA` env var follows the existing pattern of compiler-injected context variables (e.g., `GH_AW_WORKFLOW_ID`, `GH_AW_CALLER_WORKFLOW_ID`, `GH_AW_AMBIENT_CONTEXT`).
- When `GH_AW_HEAD_SHA` is absent or empty, `submitReview()` falls back to `pullRequest.head.sha`, preserving existing runtime behavior for unsupported trigger types.
