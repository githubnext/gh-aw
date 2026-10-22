# ADR-70915: Ignore Evidence-Confirmed PR Lifecycle Interruptions in `gh aw health`

**Date**: 2026-10-21
**Status**: Draft
**Deciders**: pelikhan, [TODO: verify reviewers]

---

### Context

`gh aw health` samples recent workflow runs and turns cancelled and failed runs into failure metrics, revision/recovery status, failure clusters and investigation targets. Many `pull_request` runs end because the PR lifecycle interrupts them, not because the workflow is broken. A PR can be merged, closed or have its head branch deleted while a run is in progress, which cancels the run. A checkout can also fail on purpose with the diagnostic "PR #N closed before its head could be fetched". Counting these runs as failures produces false clusters and investigations, as seen with run 39618696466 after PR #70898 merged. The current state of a PR or branch (closed now, or a 404 now) does not show *when* it changed, so it cannot be trusted alone. The command must stay read-only and within a bounded API budget.

### Decision

We will classify every completed `pull_request` run with a `cancelled` or `failure` conclusion as `ignored`, `retained` or `unknown`. A run is excluded from failure inventory and chronological sampling **only** when it is `ignored`. Before metrics, clustering and investigation selection run, a run becomes `ignored` only if all of these hold:

- Exact-commit PR evidence (`/commits/{sha}/pulls`, matching head branch) or timestamped `head_ref_deleted` issue events (not reversed by a later `head_ref_restored`) place a merge, closure or head deletion during the attempt.
- The run's job inventory is complete.
- No job or step failed for a genuine reason.
- A cancelled run has a cancelled job that completed at or after the lifecycle event.
- A failed run has only an `agent` job whose single failed step is `Checkout PR branch`, and the job log contains the exact closed-PR diagnostic. The only other failure allowed is the subsequent `Handle agent failure` step in `conclusion`.

Ignored runs are not successes and do not count toward the sample minimum. Lifecycle API responses, including errors, are cached per collection. Lifecycle requests are capped at 40, and branch-event listings at two pages. Every classification and its reason are exposed per run in JSON and text output.

### Alternatives Considered

#### Alternative 1: Ignore all cancelled `pull_request` runs (or all runs whose PR is now closed)

This approach is simple and needs few or no extra API calls. It was not chosen because it hides genuine failures and unrelated cancellations. A PR's closed state today, or a branch's absence today, does not show that the closure caused the interruption.

#### Alternative 2: Keep counting lifecycle interruptions as failures and leave them to human triage

This approach adds no API cost or classification logic. It was not chosen because it causes false failure clusters, wrong revision/recovery status and wasted investigations. In the live example, a workflow whose only run was interrupted by a merge was reported as failing.

#### Alternative 3: Treat lifecycle-interrupted runs as successes

This approach keeps sample counts stable. It was not chosen because an interrupted run tells us nothing about whether the workflow works. Counting it as a success would inflate health. Ignored runs are dropped instead, and a window that contains only ignored runs reports `no_recent_runs`.

### Consequences

#### Positive
- PR merges, closures and branch deletions no longer create false failure clusters or investigation targets.
- The rules are conservative. Missing, incomplete or ambiguous evidence gives `unknown` or `retained`, so genuine failures are never hidden.
- Each run has an explicit classification and reason, which makes health decisions auditable.

#### Negative
- Each collection can make up to 40 extra API requests (commit→PR lookups, issue events and job logs), which adds latency and budget pressure.
- The checkout rule depends on an exact diagnostic string and the step names `Checkout PR branch` and `Handle agent failure`. If those texts change, the rule fails closed and the runs are retained.
- The classification logic adds complexity and must stay in step with `obsoletePRWindow` and the typed job-evidence pipeline from ADR-70885.

#### Neutral
- Filtering happens before sampling, so ignored runs do not count toward the minimum and more history may be scanned to reach it.
- Once the lifecycle request cap is reached, the remaining runs are classified `unknown` and retained.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
