# ADR-43410: Logs Summary — Backfill Precedence Rule and Lazy Cache Healing

**Date**: 2026-07-05
**Status**: Draft
**Deciders**: Unknown (automated fix by copilot-swe-agent, pelikhan)

---

### Context

The `logs` tool builds a summary for each workflow run that includes `total_turns` and `total_safe_items`. These values can come from two sources:

1. **Log-derived metrics** (`result.Metrics.Turns`) — extracted from `events.jsonl` or `.log` files when full log artifacts are downloaded.
2. **Backfilled values** (`applyUsageActivitySummaryToResult`) — read from `usage/activity/summary.json`, which is always present even when full log artifacts are not downloaded (usage-only mode).

Two bugs caused these summary fields to always report `0`:

- **Bug 1 (orchestrator)**: The orchestrator unconditionally assigned `run.Turns = result.Metrics.Turns`. For usage-only downloads (no `events.jsonl`/`.log` files), `result.Metrics.Turns` is always `0`, discarding the backfilled value.
- **Bug 2 (cache-hit path)**: The cache-hit path returned cached `run_summary.json` as-is. Cache entries written before the backfill feature was introduced (no schema invalidation exists) kept `Turns` and `SafeItemsCount` at `0` indefinitely.

### Decision

We will apply two complementary rules:

1. **Backfill-wins-when-metrics-is-zero**: In the orchestrator, only overwrite `run.Turns` with `result.Metrics.Turns` when that value is greater than zero. This preserves any backfilled value for usage-only artifact downloads while still preferring the more precise log-derived count when full logs are available.

2. **Lazy cache healing on zero**: In the cache-hit path, if either `Turns` or `SafeItemsCount` is zero in the cached result, re-apply `applyUsageActivitySummaryToResult` from the on-disk `usage/activity/summary.json`. Because `applyUsageActivitySummaryToResult` is a no-op when values are already non-zero, this is safe to call unconditionally and self-heals stale cache entries without requiring a re-download or explicit cache invalidation.

### Alternatives Considered

#### Alternative 1: Always Prefer Backfilled Values

Never overwrite `run.Turns` with `result.Metrics.Turns`; treat log-derived metrics as supplementary only. This would fix Bug 1 but would lose precision: for runs with full log artifacts, `events.jsonl` counts turns at a finer granularity than `session.turns` in the activity summary.

Not chosen because it degrades accuracy for the common case where full log artifacts are present.

#### Alternative 2: Version-Based Cache Invalidation

Embed a backfill schema version in `run_summary.json` and invalidate the cache whenever the version changes. This would fix Bug 2 cleanly and avoid the I/O cost of conditionally re-reading the activity summary.

Not chosen because it requires a versioning/invalidation infrastructure that does not currently exist in the cache layer, and would force re-downloads rather than self-healing existing entries.

#### Alternative 3: Backfill in `extractLogMetrics` Fallback

When `extractLogMetrics` returns `Turns == 0`, fall back to reading the activity summary within the metrics-extraction step so that `result.Metrics.Turns` is always non-zero.

Not chosen because it would couple the log-metrics extraction path to the usage-activity backfill, blurring the separation of concerns between log parsing and usage-summary reading. It also does not address Bug 2.

### Consequences

#### Positive
- `total_turns` and `total_safe_items` now correctly reflect non-zero values from `usage/activity/summary.json` for usage-only artifact downloads.
- Stale cache entries self-heal on the next `logs` invocation without requiring cache deletion or a full re-download.
- Log-derived turn counts retain priority over backfilled values when full log artifacts are present, preserving accuracy.
- Five new unit tests codify the precedence rule and cache-healing behavior as executable contracts.

#### Negative
- The cache-hit path now incurs a conditional file read (`loadUsageActivitySummary`) for any cache entry where `Turns` or `SafeItemsCount` is zero, adding a small I/O cost per affected run.
- If a run genuinely has zero turns and zero safe items (e.g., an aborted run with no activity), the guard condition fires on every cache hit, reading a summary that changes nothing — a small wasted I/O.

#### Neutral
- The backfill function `applyUsageActivitySummaryToResult` must remain idempotent (no-op for non-zero values) for the lazy-healing approach to be safe; this invariant is now load-bearing.
- No cache format or schema version change is introduced; old cache entries are healed in place rather than invalidated.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
