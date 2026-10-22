# ADR-71164: Detect Approval Expiry via Attempt-Specific Run Page Annotations in Health

**Date**: 2026-10-22
**Status**: Proposed
**Deciders**: Repository maintainers through review of [#71164](https://github.com/github/gh-aw/pull/71164)

---

### Context

GitHub records workflow runs whose approval request expired as `completed`/`failure` runs, even though no jobs executed. `gh aw health` already excluded jobless runs from execution metrics (after the Jobs API confirms the attempt has zero jobs), but it still surfaced them in `failure_candidates` and `potential_issues`, producing false failure signals. Run `39766415060` had zero jobs and check runs, and its GitHub run page displayed the annotation `This workflow run required approval but was not approved before it expired.` The inspected REST run, check-suite, and approval-history payloads omitted this annotation, so they did not distinguish approval expiry from other jobless failures.

### Decision

Detect approval expiry for completed, failed runs by fetching the attempt-specific GitHub run page (`https://{host}/{owner}/{repo}/actions/runs/{id}/attempts/{n}`) as HTML, only after the Jobs API confirms the attempt has zero jobs. Parse elements whose `data-target` contains the token `annotation-message.annotationContainer`. Classification requires at least one recognized annotation, and every recognized annotation must exactly match the expiry message after HTML text extraction and whitespace normalization. Missing or inaccessible annotations, mixed messages, and other annotations do not establish expiry.

Confirmed expiries have evidence status `ignored` and reason `approval_expired`. Remove their counts from `failure_candidates` and `potential_issues`; remove an empty issue, or promote the next non-ignored inventory candidate when its representative was removed. Backfill excluded candidates, reusing overflow from fetched listing pages before requesting older pages, so recent expiries do not consume the retained-candidate allowance. The per-workflow cap remains five retained candidates, within the original ten-listing-page limit and shared request budget. The failure pass continues its existing job and revision checks for replacement candidates. Filtering does not turn a jobless run into a runtime success.

Page requests use the collection's configured host and repository identity, never a run-provided URL. They share the health request budget and are capped at 20 per collection. Evidence reuse across failure-first and chronological collection requires matching run ID, attempt, and update time; a changed attempt or update time requires fresh evidence. Candidate count updates also require matching attempt and update-time identity.

Annotation lookup errors that stop collection, including quota or shared-budget exhaustion, propagate immediately through both collection paths. Ordinary inaccessible-page errors leave evidence unknown and candidates retained. Request-failure reasons are bounded before reporting or logging.

Retain evidence in JSON `collection.approval_expiry_evidence` and text `[health_approval_expiry]` output. This includes unknown or retained classifications when expiry cannot be established.

### Alternatives Considered

#### Alternative 1: Use REST API payloads (runs, check suites, approval history)

The preferred source would be a structured API field. The inspected payloads omit the expiry annotation, so they cannot establish approval expiry. A documented structured field could replace HTML parsing if GitHub exposes equivalent attempt-specific evidence.

#### Alternative 2: Treat all jobless failures as non-failures

Removing every jobless failure from the failure inventory would require no annotation requests. Rejected because jobless failures can have other causes, such as workflow file errors; silently dropping them would hide genuine problems. The decision retains unconfirmed jobless failures as potential issues, while preserving their existing exclusion from execution metrics.

#### Alternative 3: Keep reporting expired approvals as potential failures

No new code or requests. Rejected because it produces recurring false-positive failure candidates that mislead health triage.

### Consequences

#### Positive
- Approval-expired runs no longer appear as failure candidates, reducing false positives in health reports.
- Fail-closed classification: missing, inaccessible, or mixed annotations never establish expiry, so genuine failures stay visible.
- JSON retains the attempt, update time, classification, reason, URL when available, and exact message for confirmed expiry. Text output identifies the run, attempt, classification, reason, and URL for auditability.

#### Negative
- Depends on an undocumented GitHub web UI selector (`annotation-message.annotationContainer`) and exact message text. DOM or wording changes can disable detection without a command failure; affected runs remain potential issues. Exact matching deliberately favors false positives in triage over hiding unrelated failures; broader selectors or fuzzy matching require fresh evidence and review.
- Adds extra HTTP requests (up to 20 per collection) that consume the shared health request budget; runs beyond the cap remain unclassified.
- Adds HTML parsing with `golang.org/x/net/html` to the health path.

#### Neutral
- Adds the `approval_expiry_filtering` capability and `approval_requests` counter to the health report schema.
- Expired runs are excluded from failures but are not counted as successes; the workflow may still report `no_recent_runs`.

---

This proposal records the implemented decision; maintainer review and merge determine acceptance. It does not claim human sign-off or a passing Design Decision Gate, and it does not authorize workflow dispatch.
