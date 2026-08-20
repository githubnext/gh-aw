# ADR-54110: Surface Continuation Cursor When MaxIterations Is Hit During Explicit Date-Range Scans

**Date**: 2026-08-20
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `collectProcessedWorkflowRuns` pagination loop in `pkg/cli/logs_orchestrator_download.go` caps at `MaxIterations` (20). When an explicit `start_date`/`end_date` window is requested (`fetchAllInRange = true`) and many non-matching runs fill each batch, the loop can exhaust all iterations without reaching the requested `count` or triggering the timeout. Previously, the loop exited normally in this case with both `timeoutReached` and `countLimitReached` set to `false`, so no `continuation` cursor was emitted. Callers requesting wide windows (e.g., 90 days) silently received a narrow, potentially stale slice of data, which downstream trend-analysis consumers treated as a complete representative scan of the full range (see github/gh-aw#53995).

### Decision

We decided to set `countLimitReached = true` whenever `collectProcessedWorkflowRuns` exits due to the `MaxIterations` cap during an explicit date-range scan (`fetchAllInRange && !timeoutReached && iteration >= MaxIterations`). This reuses the existing `countLimitReached` path to guarantee a `continuation` cursor and `"partial": true` are always emitted for incomplete date-range scans. We additionally added `dateRangeCoverageWarning` — a function that, when the result is partial and the returned runs span less than 20% of the requested window, emits an explicit warning via the existing `stale_warning` mechanism so callers know the result is a narrow slice rather than a representative multi-day sample.

### Alternatives Considered

#### Alternative 1: Increase or Remove MaxIterations

Raising or removing the `MaxIterations` cap would allow the loop to scan deeper into a wide date range before stopping. This was rejected because the cap exists to bound API cost and wall-clock time per call; removing it could cause runaway pagination that exhausts rate limits or blocks other callers. Raising it only defers the problem without fixing the signaling gap.

#### Alternative 2: Add a Top-Level Post-Scan Incomplete-Range Check

An alternative was to detect incompleteness after the loop by comparing the oldest fetched run's date against the requested `start_date`, and emit a warning only at the render layer. This would avoid changing loop semantics. It was rejected because the root cause is a missing signal (`countLimitReached`), not just a missing warning — consumers that rely on the continuation cursor to decide whether to page would still receive no cursor and would have no way to resume the scan, even if the render layer warned them.

### Consequences

#### Positive
- Callers always receive a `continuation` cursor when pagination stops due to the iteration cap on explicit date-range scans, enabling them to resume the scan rather than silently consuming incomplete data.
- The `dateRangeCoverageWarning` gives users an actionable message when results are a narrow slice of the requested window, preventing silent multi-day trend analysis on unrepresentative data.

#### Negative
- The 20% coverage threshold (`dateRangeCoverageMinFraction = 0.2`) is a heuristic; edge cases exist where legitimate results cluster in a short sub-window without this being a data quality problem, causing false-positive warnings.
- Changing `countLimitReached` semantics to include iteration-cap exits means the continuation signal now appears in cases where it was previously absent — consumers must handle this cursor correctly or may perform unnecessary extra fetches.

#### Neutral
- Regression tests were added for both the iteration-cap continuation behavior and the coverage-warning threshold logic, establishing a baseline for future changes to the pagination semantics.
- The fix applies only when `fetchAllInRange` is `true` (i.e., an explicit date range was requested); the no-date-range code path is unaffected.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
