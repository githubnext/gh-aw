# ADR-53719: Warn Callers When `logs` MCP Tool Returns Stale Data Without a Date Range

**Date**: 2026-08-18
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `logs` MCP tool paginates workflow runs backwards by creation date until it has collected the requested count. In high-activity repositories, this walk can silently land on a window that is days or weeks old when recent non-agentic runs dominate the page. Callers that omit `start_date`/`end_date` receive stale data with no signal that more recent runs exist, causing deep-report and audit workflows to draw incorrect conclusions about current fleet health (see issue #53683).

### Decision

We will add a post-query staleness check (`staleLogsWarning`) that fires only when no date range was requested and the newest run in the result set is older than 48 hours. The warning is folded into the output's `message` field by `renderLogsOutput`, then extracted and surfaced verbatim in the MCP tool's top-level response text by `buildLogsFileResponse` so callers see it immediately without opening the cached file.

### Alternatives Considered

#### Alternative 1: Require explicit date bounds (hard rejection)

Reject any `logs` call that omits both `start_date` and `end_date`, returning an error that forces the caller to specify a range.

This eliminates the ambiguity entirely but is a breaking change: all existing callers that rely on the count-only invocation pattern would need to be updated simultaneously, and some legitimate use-cases (e.g., "give me the last N runs regardless of when they ran") become impossible to express.

#### Alternative 2: Transparent auto-retry with a default date window

Detect staleness and automatically re-issue the query with a sensible default `start_date` (e.g., `-1d`) without telling the caller.

This silently fixes the common case but hides the ambiguity rather than exposing it. Callers lose visibility into the scope of their query; if the auto-selected window is wrong for a given repo's cadence, results are still wrong—and now there is no warning to prompt investigation.

### Consequences

#### Positive
- Callers receive an actionable warning in the MCP response's top-level `message` field immediately, without reading the cached file.
- No breaking change: callers that already supply explicit date bounds see no difference in behavior.
- The 48-hour threshold is documented as a named constant (`staleLogsWarningThreshold`), making it easy to tune.

#### Negative
- The staleness heuristic (48 hours) is repo-cadence-agnostic; low-activity repositories may never trigger the warning even when results are genuinely stale, while repos with infrequent runs could trigger false positives.
- The warning travels through two encoding layers (render → JSON `message` field → guardrail JSON extraction), creating coupling between `renderLogsOutput` and `buildLogsFileResponse` that can silently break if the output schema changes.

#### Neutral
- Unit tests cover all four guard conditions: explicit start date, explicit end date, empty result set, and recent-data threshold, providing a regression baseline for future threshold changes.
- The `renderLogsOutputOptions` struct gains two new fields (`startDate`, `endDate`) that are threaded from `DownloadWorkflowLogs`, a minor expansion of the internal API surface.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
