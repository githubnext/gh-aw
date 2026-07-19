# ADR-46360: Fix SafeItemsCount Population in run_summary.json

**Date**: 2026-07-18
**Status**: Draft
**Deciders**: Unknown

---

### Context

Daily audit workflows (API Consumption, Safe Output Health) read `SafeItemsCount` from `run_summary.json` to measure how many safe-output items an agent run actually wrote to GitHub. Three independent bugs caused this field to always be `0`, forcing audits to fall back to `usage/activity/summary.json → safe_outputs.total_items`. This fallback masked the root cause and made audit data less reliable. The bugs were: (1) `WorkflowRun.SafeItemsCount` lacked a `json:"safe_items_count"` tag so it marshaled as PascalCase while readers expected snake_case; (2) the `safe-outputs-items` artifact was skipped by `flattenSingleFileArtifacts` because it contains two files, so `extractCreatedItemsFromManifest` never found either; (3) `backfillCacheHitIfNeeded` healed `SafeItemsCount` in memory but never called `saveRunSummary`, leaving the on-disk file stale for cached runs.

### Decision

We will fix all three root causes simultaneously: add the missing JSON struct tag to `WorkflowRun.SafeItemsCount`; introduce `flattenSafeOutputsItemsArtifact()` following the existing `flattenActivationArtifact` / `flattenAgentOutputsArtifact` pattern to move both files to the run root; and detect when `backfillCacheHitIfNeeded` changes `SafeItemsCount` (via snapshot-and-compare) and call `saveRunSummary` to persist the healed value. All three fixes are required together because any single fix leaves the other failure paths open.

### Alternatives Considered

#### Alternative 1: Treat usage/activity/summary.json as the authoritative source

Rather than fixing `run_summary.json` population, audit tools could always read `SafeItemsCount` from `usage/activity/summary.json`. This eliminates the need to flatten artifacts or persist the backfill value.

This was not chosen because it centralises audit logic on a different data path, requires every downstream consumer to know about the fallback, and obscures the signal that `run_summary.json` is incomplete. The existing fallback was already a workaround; making it permanent would entrench technical debt.

#### Alternative 2: Compute SafeItemsCount from raw JSONL at query time

Each report could count lines in `safe-output-items.jsonl` directly instead of relying on pre-computed fields in `run_summary.json`. This avoids JSON tagging and caching problems.

This was not chosen because it breaks the existing contract where `run_summary.json` is a self-contained, fully resolved snapshot of a run's metrics. Recomputing at query time duplicates parsing logic across consumers and increases I/O for each report run. The bugs are well-understood and fixable at the source.

#### Alternative 3: Add a dedicated artifact-search step instead of flattening

Instead of flattening the `safe-outputs-items` subdirectory, `extractCreatedItemsFromManifest` could be updated to search one level deep for the relevant files.

This was not chosen because the rest of the codebase consistently uses the flatten-to-root pattern for all multi-file artifacts (`activation`, `agent_outputs`). Changing `extractCreatedItemsFromManifest`'s search semantics would be a broader, riskier change and would deviate from the established convention.

### Consequences

#### Positive
- `run_summary.json` now contains accurate `SafeItemsCount` values, eliminating the need for the audit fallback path.
- `flattenSafeOutputsItemsArtifact()` follows the established pattern for artifact flattening, making the codebase consistent.
- Cache hits that previously left `SafeItemsCount = 0` on disk now produce a correct, persisted value for downstream readers.
- Three targeted tests lock in the correct behavior and prevent regression.

#### Negative
- The `omitempty` option on the JSON tag means runs with zero safe outputs emit no `safe_items_count` key; readers that do not handle the absent-key case gracefully may interpret absence as unknown rather than zero.
- The snapshot-and-compare approach in `tryLoadCachedRunResult` triggers an additional `saveRunSummary` disk write for every cache hit where `SafeItemsCount` changes; this is a one-time write per affected run but introduces a new I/O path in the hot cache-load code path.

#### Neutral
- The `flattenSafeOutputsItemsArtifact` function mirrors `flattenActivationArtifact` and `flattenAgentOutputsArtifact` — consistent pattern, but the number of special-cased flatten helpers continues to grow; a future refactor may want to unify them.
- Existing audit code that uses the `usage/activity/summary.json` fallback remains in place as a safety net; it is now dead code for correctly processed runs but still executes for older cached runs that have not been re-processed.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
