# ADR-48738: Pin PR Review Attribution to the Reviewed Commit via Optional `commit-id` Config

**Date**: 2026-07-28
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Under `workflow_run` triggers, the safe-outputs runtime submits a PR review after a parent workflow completes. A new commit can land on the PR branch while the agent is running. When that happens, `pulls.get()` returns the new HEAD SHA, and the review is attributed to a commit the agent never actually reviewed — making inline comments appear fabricated, outdated, or misaligned. There was no mechanism for callers to pass the specific SHA the agent reviewed to the review submission step. This race condition is most acute in `workflow_run` contexts where an eligibility job captures the PR HEAD at trigger time and passes it to downstream jobs.

### Decision

We will add an optional `commit-id` configuration field to both `submit-pull-request-review` and `create-pull-request-review-comment` safe-output handlers. When set, this SHA is used as the `commit_id` argument to `pulls.createReview()` instead of the live PR head SHA. The field is optional and backward-compatible: when omitted, behavior is unchanged (the live PR head SHA is used). Callers explicitly pass the reviewed SHA — typically captured by an upstream eligibility job — via `commit-id: ${{ needs.eligibility.outputs.head_sha }}` in their workflow YAML.

### Alternatives Considered

#### Alternative 1: Automatically capture and freeze the PR HEAD SHA at agent startup

Freeze the SHA once when the safe-outputs handler initialises, before the review accumulation phase begins, and always use that frozen value. This would require no caller-side configuration.

Why not chosen: The agent may run for an extended period before submitting the review. A commit pushed after handler initialisation but during the agent run would still cause attribution drift. More importantly, in `workflow_run` scenarios the correct SHA to attribute is the one the *triggering* workflow saw — which may already be older than what the safe-outputs job sees at startup. That SHA is only knowable by the caller via an upstream job output, not by the handler itself.

#### Alternative 2: Always use the triggering `workflow_run` payload SHA

For `workflow_run` events, automatically extract the SHA from the triggering workflow's context payload instead of calling `pulls.get()`.

Why not chosen: The triggering payload does not always carry the SHA of the PR commit the agent reviewed (it may contain the SHA of the triggering workflow's default branch commit). Callers need explicit control over which commit to attribute, particularly when the reviewed SHA is computed by a separate eligibility job and stored as a job output. This approach also does not address non-`workflow_run` triggers that may have similar drift in edge cases.

### Consequences

#### Positive
- Reviews are attributed to the commit the agent actually reviewed, eliminating false "outdated" or misaligned inline comments.
- GitHub correctly marks the review as outdated when HEAD moves, rather than incorrectly attaching stale comments to the new commit.
- The feature is fully opt-in and backward-compatible — existing callers are unaffected.

#### Negative
- Callers must explicitly wire the reviewed SHA through their workflow YAML (e.g., from an eligibility job output); forgetting to set `commit-id` leaves the race condition unaddressed in `workflow_run` contexts.
- The `commit-id` field is threaded through three layers (Go config struct → handler registry JSON → JS runtime), increasing the surface area of the config pipeline.

#### Neutral
- The Go config structs (`SubmitPullRequestReviewConfig`, `CreatePullRequestReviewCommentsConfig`) gain a new `CommitId string` field, which is zero-valued (empty) when not configured — no breaking change to existing parsed configs.
- The JS runtime falls back to `pullRequest.head.sha` when `commitId` is absent, preserving existing runtime behaviour.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
