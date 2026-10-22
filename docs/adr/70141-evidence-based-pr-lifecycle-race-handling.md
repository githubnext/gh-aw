# ADR-70141: Evidence-Based Handling of Queued and In-Flight PR Lifecycle Races

**Date**: 2026-10-18

**Status**: Proposed (implemented in PR #70141; maintainer acceptance pending)

## Context

A pull request can merge or close while a workflow is queued or running, and its head branch can then be deleted. GitHub can finalize a queued run as failed or cancelled without starting any jobs, or an in-flight checkout or push can lose its target ref. Continuing after a failed PR checkout risks running the agent on the previous base checkout; a subsequent push risks recreating a deleted branch.

The user requested that "AW needs to deal with PR being merge half-way through and the branch deleted" and "definitely detect queued runs for PRs that got cancelled and weed them out". This requires distinguishing affirmative lifecycle evidence from ordinary failures, not treating closure or missing artifacts as a default explanation for every failure. Workflows intentionally triggered on closed or merged PRs must remain supported.

## Decision

### Require attempt-specific evidence for obsolete queued runs

[`logs_pr_lifecycle.go`](../../pkg/cli/logs_pr_lifecycle.go) classifies a run with `exclusion_reason: obsolete_pr` only when all of the following hold:

- It is a completed `pull_request` run with raw conclusion `failure` or `cancelled`, a known positive attempt number, and an exact head commit.
- The Jobs API affirmatively reports zero jobs for that exact attempt, and no observed failure signals contradict the classification. Missing, malformed, or inaccessible API data is unknown evidence, not an empty jobs list.
- The commit-to-PR API yields one eligible association after filtering affirmative head-branch mismatches. Missing PR branch refs remain candidates rather than being discarded to manufacture uniqueness. If the run has a known head branch, the selected PR must positively match it. This is not a requirement that the commit map to exactly one PR globally; the PR's current head SHA need not still equal the run's commit.
- That PR is closed, with closure strictly after the attempt's `started_at` (falling back to `created_at`) and no later than the run's `updated_at`.

The record retains the PR number, closure time, and attempt. Qualified runs remain in raw output and summary counts but are excluded from actionable failure clusters and investigations. Raw GitHub conclusions are not rewritten. Startup-validation conclusions, job-backed failures, observed failure signals, ambiguous associations, and unavailable evidence remain visible rather than being excused as lifecycle races.

Executed-run success metrics count only completed runs with confirmed jobs. Confirmed jobless runs and unknown execution evidence are not successes. This reports evidence of lifecycle invalidation without claiming to prove GitHub's internal cause or cancelling native queued runs.

### Stop failed PR checkouts instead of falling back to the base

[`checkout_pr_branch.cjs`](../../actions/setup/js/checkout_pr_branch.cjs) fails the checkout step when the required PR head cannot be checked out, preventing subsequent agent execution on the previous checkout. A missing remote PR ref plus a live state check confirming closure of an originally open PR records lifecycle invalidation separately, allowing failure handling to avoid a redundant lifecycle failure notification. Authentication, permission, network, and unknown checkout errors remain failures.

Failure handling also recognizes disappearance of the default `refs/pull/N/merge` checkout only with compiler-confirmed trigger-ref use, a failed default checkout completed at or after PR closure, and job logs identifying that exact missing remote ref. Closure alone does not suppress a checkout failure notification.

An intentionally closed-event workflow is not invalidated merely because its PR is closed. Where PR-head checkout is enabled, it uses the retained `refs/pull/N/head` path and still requires checkout success.

### Recheck live state at effect boundaries and fence branch writes

[`pr_lifecycle.cjs`](../../actions/setup/js/pr_lifecycle.cjs) checks live PR state for work triggered while the PR was open, including PR issue-comment context. The safe-output manager checks before handlers, deferred retries, temporary-ID follow-up updates, and buffered review submission. Confirmed closure skips obsolete effects; lookup failures are errors, not evidence of closure. Intentionally closed-event workflows and staged previews are not suppressed by this lifecycle check.

PR-branch pushes recheck state before push attempts and fallback PR creation, and before activation comments or extra CI-trigger commits. [`push_signed_commits.cjs`](../../actions/setup/js/push_signed_commits.cjs) requires the target PR branch to exist: signed API writes use `expectedHeadOid` without recreating a missing branch; direct Git pushes check ancestry and use an explicit `--force-with-lease=refs/heads/<branch>:<expected-oid>`. The extra CI-trigger commit likewise requires an existing branch, using an expected OID for API writes or an explicit lease for Git pushes. These branch-update conditions prevent deletion between a read and a write from recreating the ref; live PR state checks alone cannot provide that protection.

## Alternatives Considered

These comparisons explain the implemented choice. The available history does not establish that named human deciders evaluated these alternatives, including concurrency groups.

### Ignore every failed or cancelled run whose PR is now closed

This would hide real authentication, validation, runtime, or push failures and incorrectly suppress intentional closed-event workflows. Requiring attempt-specific zero-job evidence, a unique eligible association, a bounded closure time, and no contradictory failure signals preserves actionable failures and unknowns.

### Rely only on concurrency groups, cancellation, or a job-level guard

Concurrency policies can reduce overlapping work but do not establish why an attempt failed, guarantee deletion-safe writes, or protect a run already performing effects. A job guard cannot execute before any jobs start or change GitHub's pre-job conclusion. Evidence-based reporting and runtime checks therefore address different stages rather than relying on cancellation alone.

### Continue on the base checkout or retry a push without an existing-ref condition

Continuing after checkout failure can apply work to the wrong base. Checking PR state or branch existence once before an unconditional push still leaves a deletion race. Failed checkouts stop execution, while expected-OID conditions and explicit leases fence writes against branch changes or deletion.

## Consequences

Known obsolete queued attempts no longer dominate actionable failure analysis, while raw conclusions, lifecycle evidence, confirmed non-execution, and unknowns remain available for diagnosis. In-flight work stops on a failed PR checkout, and guarded PR pushes refuse to resurrect deleted branches. Intentional closed-event workflows remain usable.

The classifier is deliberately conservative: unavailable Jobs or PR APIs, missing association data, ambiguous commit associations, or closure outside the attempt window can leave a genuinely obsolete run unclassified. Live checks add API calls and require read access to the triggering PR; failures to obtain that evidence remain errors. Regression coverage must preserve these distinctions across fresh and cached log reports and across checkout, safe-output, signed-push, direct-push, and extra-commit paths.

PR state checks and later effects are not atomic. A PR can close after a successful check, and already-applied effects are not rolled back. Existing-ref conditions fence branch writes, not every possible side effect or the PR's open state. No in-job change repairs a platform-side pre-job conclusion; no native run cancellation or unconditional success conversion is introduced.

Implementation: [PR #70141](https://github.com/github/gh-aw/pull/70141). Operational references: [Logs](../src/content/docs/setup/cli.md#logs) and [Checkout](../src/content/docs/reference/checkout.md).
