# ADR-71164: Detect Approval Expiry via Attempt-Specific Run Page Annotations in Health

**Date**: 2026-10-22
**Status**: Draft
**Deciders**: pelikhan, [TODO: verify]

---

### Context

GitHub records workflow runs whose approval request expired as `completed`/`failure` runs, even though no jobs ever executed. `gh aw health` already excluded jobless runs from execution metrics (after the Jobs API confirms the attempt has zero jobs), but it still surfaced them in `failure_candidates` and `potential_issues`, producing false failure signals. The only explicit evidence distinguishing an approval expiry from other jobless failures is the annotation `This workflow run required approval but was not approved before it expired.`, which is shown on the GitHub run page. The REST run, check-suite, and approval-history payloads do not include this annotation.

### Decision

We will detect approval expiry by fetching the attempt-specific GitHub run page (`https://{host}/{owner}/{repo}/actions/runs/{id}/attempts/{n}`) as HTML, only after the Jobs API confirms the attempt has zero jobs, and parsing `annotation-message.annotationContainer` elements. A run is classified as `approval_expired` only when annotations are present and all of them exactly match the expiry message. Confirmed expiries are removed from `failure_candidates` and `potential_issues` (with an older retained candidate promoted as representative), while evidence is retained in JSON `collection.approval_expiry_evidence` and text `[health_approval_expiry]` output. Page requests use the configured host and repository (never a run-provided URL), share the health request budget, are capped at 20 per collection, and are cached by run ID, attempt, and update time.

### Alternatives Considered

#### Alternative 1: Use REST API payloads (runs, check suites, approval history)

The preferred source would be a structured API field. It was rejected because, per the PR, these payloads omit the expiry annotation, so they cannot distinguish an approval expiry from other jobless failures.

#### Alternative 2: Treat all jobless failures as non-failures

Simpler and requires no extra requests. Rejected because jobless failures can have other causes (e.g., workflow file errors); silently dropping them would hide genuine problems. The decision keeps unconfirmed jobless failures as approximate potential issues.

#### Alternative 3: Keep reporting expired approvals as potential failures

No new code or requests. Rejected because it produces recurring false-positive failure candidates that mislead health triage.

### Consequences

#### Positive
- Approval-expired runs no longer appear as failure candidates, reducing false positives in health reports.
- Fail-closed classification: missing, inaccessible, or mixed annotations never establish expiry, so genuine failures stay visible.
- Explicit evidence (URL, attempt, reason, message) is retained in JSON and text output for auditability.

#### Negative
- Depends on undocumented GitHub web UI HTML structure (`data-target="annotation-message.annotationContainer"`) and exact message text, which may change without notice and silently disable detection.
- Adds extra HTTP requests (up to 20 per collection) that consume the shared health request budget; runs beyond the cap remain unclassified.
- Introduces a new dependency on `golang.org/x/net/html` parsing in the health path.

#### Neutral
- Adds the `approval_expiry_filtering` capability and `approval_requests` counter to the health report schema.
- Expired runs are excluded from failures but are not counted as successes; the workflow may still report `no_recent_runs`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
