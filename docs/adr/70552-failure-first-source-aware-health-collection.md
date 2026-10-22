# ADR-70552: Failure-First, Source-Aware Workflow Health Collection

**Date**: 2026-10-20
**Status**: Proposed — implementation reviewed; maintainer acceptance pending
**Deciders**: pelikhan (confirmed PR author); no additional deciders or approval confirmed

---

### Context

`gh aw health` samples recent runs for each registered agentic workflow under a shared GitHub API budget (about 1,000 requests). The earlier collector walked each workflow's history chronologically. It spent much of the budget on successful runs before reaching failures, so it often missed failures that maintainers needed to act on. It also compared revisions only through lock-file `gh-aw-metadata` hashes. If someone pushed a Markdown source edit without recompiling the lock file, old failures still counted as current. Health reports then sent investigators to failures that a later source edit had already superseded. Live checks over the same window showed that collection order strongly affected how many actionable failures appeared before the budget ran out.

### Decision

Collect health evidence in two budgeted passes and track workflow source revisions:

1. **Failure-first sniffing**: Inventory registered AW runs whose conclusion is explicitly `failure`, using repository-wide `actions/runs?status=failure` queries, or a workflow-scoped endpoint when a single workflow is selected. Limit listing to ten pages and five candidates per workflow under the shared API budget. Preserve these approximate, list-only signals in `potential_issues`; they are not confirmed current-revision failures. Verify jobs and revision evidence for at most one job-bearing representative per workflow/branch/event scope, skipping jobless candidates and superseded revisions.
2. **Chronological pass**: The existing per-workflow sampling runs next. It visits workflows with recent failures first (`prioritizeHealthFailureWorkflows`) and deduplicates runs it already collected (`mergeHealthRunSamples`).

For each supported run, compare the Markdown source's blob SHA at the run's head SHA with its blob SHA on the pushed branch. When they differ, collection stops for older history in that branch/event scope, and the superseded failures are excluded from current clusters and investigation targets. The report separates failure-listing completeness from representative verification with `collection.failure_listing_complete` and `collection.failure_pass_complete`, and exposes `source_sha` on revision cohorts.

When the local checkout's origin matches the target repository and host, reuse committed Git trees and blobs at immutable SHAs. Resolve pushed branch heads through the API rather than trusting local branches, and never use working-tree edits as pushed evidence. Local reads disable lazy fetching and prompts; missing objects fall back to GitHub API reads. Cache workflow directory snapshots per ref and lock metadata per blob to amortize remote lookups.

### Alternatives Considered

#### Alternative 1: Keep chronological per-workflow sampling and raise the API budget

The earlier collector used chronological per-workflow sampling. Raising its budget would retain that ordering, but cost grows with fleet size, rate limits still apply, and successful runs can consume requests before failures appear. Failure-first inventory is preferred; neither bounded approach guarantees an unbiased sample.

#### Alternative 2: Detect superseded revisions from lock-file metadata hashes only

The earlier revision check used `frontmatter_hash`/`body_hash` from the compiled `.lock.yml`. Keeping only that check avoids source lookups, but source edits can land before recompilation, leaving old failures apparently current. Source blob comparison catches these edits, with local reuse and cached directory snapshots reducing API cost.

#### Alternative 3: Stop collection on any repository commit after a run

This alternative, evaluated during this ADR review rather than documented as a prior implementation, would treat any newer branch commit as a revision boundary. Unrelated commits would stop sampling too early and hide current failures. Per-workflow source comparison ignores unrelated changes.

#### Alternative 4: Verify every candidate remotely before recording failure signals

The initial failure-first implementation checked candidate jobs and revisions during collection. The subsequent sniffing implementation instead records bounded list-only signals before representative verification and reuses local objects. This preserves visibility into potential issues without spending the shared API budget verifying every candidate; incomplete or unknown evidence remains explicit.

### Consequences

#### Positive
- Bounded failure inventory makes potential issues visible before expensive verification, while local Git reuse leaves more API budget for run evidence. Earlier live comparisons motivated failure-first ordering but are not performance guarantees for the current sniffing implementation.
- Investigators no longer chase failures from superseded sources. A pushed source with no matching runs is reported as `no_current_revision_runs` (awaiting evidence), not as healthy or fixed.
- Listing/verification completeness flags, representative status, and `source_sha` distinguish approximate signals from checked revision evidence.

#### Negative
- The sample is deliberately biased toward failures, so it is not a fleet-wide failure rate. The inventory caps can omit workflows or scopes, and limited verification can leave potential issues unconfirmed.
- Source lookups still consume API budget when local objects or cached snapshots are unavailable. Local reuse requires a matching checkout and cannot assume shallow or sparse checkouts contain all required objects.
- Collection now depends on a `.lock.yml` → `.md` path convention. Unavailable source revisions produce warnings and unknown revision evidence.

#### Neutral
- `HealthCollection` exposes inventory, completeness, and local/remote revision-read diagnostics, and `HealthRevisionCohort` adds `source_sha`. The text output of health revisions includes `source_sha`.
- The collector only treats a run's head SHA as its source revision for events that run from that SHA (push, schedule, dispatch, issues, discussions, and similar). For other events, revision evidence remains unknown.

---

*The generated draft has been checked against the current implementation. Proposed status records the decision without asserting maintainer approval; acceptance remains a maintainer decision.*
