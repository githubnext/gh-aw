# ADR-41693: Populate the `logs` JSON `message` Field with Actionable Guidance on Zero Results

**Date**: 2026-06-26
**Status**: Draft
**Deciders**: pelikhan

---

### Context

The `gh-aw logs` tool emits a JSON object whose top-level `message` field was always an empty string whenever zero workflow runs were returned. Callers (scripts, downstream tools, AI agents) had no way to distinguish a legitimately empty result set from a misconfigured query — for example, a future `--start-date`, a date older than GitHub Actions' 90-day log-retention window, or a download timeout. By contrast, an invalid workflow name already returned a descriptive error. The inconsistency degraded the usability of the JSON API surface for programmatic consumers.

### Decision

We will populate `logsData.Message` on all zero-result JSON output paths with a context-specific, human-readable explanation. A new helper `noRunsMessage(startDate, timeoutReached)` selects the most specific cause in priority order: (1) timeout, (2) future `start_date`, (3) `start_date` beyond the 90-day GitHub Actions retention window, (4) generic fallback. A second helper `parseFilterDate` normalises both `YYYY-MM-DD` and RFC 3339 date strings so the heuristic can be applied uniformly. The constant `GitHubActionsRetentionDays = 90` is extracted to make the threshold named and testable.

### Alternatives Considered

#### Alternative 1: Keep `message` Empty; Rely on `summary.total_runs: 0`

Leave the existing behaviour unchanged. Callers would be expected to detect an empty result via `total_runs: 0` and apply their own diagnostic logic. This was rejected because it shifts diagnostic burden to every caller, produces inconsistent UX compared with other error paths, and gives no guidance when the root cause is detectable server-side.

#### Alternative 2: Single Generic "No runs found." Message for All Zero-Result Paths

Emit a static `"No runs found."` string on all zero-result paths, without attempting to diagnose the specific cause. Simpler to implement and avoids heuristic reasoning. Rejected because it provides no more actionable information than an empty string for the most confusing cases (future date, beyond retention, timeout), which were the primary drivers of the change.

### Consequences

#### Positive
- Callers receive a machine-readable, human-readable explanation of why no runs were returned, enabling better error surfacing and automated diagnosis.
- The priority-ordered heuristic (`timeout > future date > retention > fallback`) mirrors the most common failure modes and is documented in code.
- `GitHubActionsRetentionDays` is now a named, exported constant, making it easy to update and reference in tests.

#### Negative
- The retention-period heuristic (90 days) is a best-effort estimate based on GitHub's default; it may produce a misleading message for organisations with custom retention settings or if GitHub changes the default.
- The `DownloadWorkflowLogsFromStdin` path uses a static string ("No valid runs could be loaded from the provided input.") rather than `noRunsMessage`, diverging slightly from the heuristic used in the main path — future contributors may need to reconcile this.

#### Neutral
- `parseFilterDate` accepts both `YYYY-MM-DD` and RFC 3339 formats to match the two date representations produced after flag resolution, which adds a small parse-cost on the zero-result hot path.
- The change does not affect non-JSON (human-readable) output paths.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
