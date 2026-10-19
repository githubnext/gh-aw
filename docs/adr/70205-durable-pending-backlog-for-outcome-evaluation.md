# ADR-70205: Durable Pending Backlog for Outcome Evaluation and Explicit Session Lineage

**Date**: 2026-05-14
**Status**: Draft
**Deciders**: pelikhan, Copilot coding agent [TODO: verify]

---

> **Superseded collector decision:** The pending-backlog design below was
> replaced before merge by CLI-only evaluation through `gh aw outcomes`.
> The scheduled collector, its cache lifecycle, and the duplicate JavaScript
> evaluator were removed. Explicit unified-session lineage, dispatch evidence,
> cross-repository PR attribution, and conservative episode grouping remain.

### Context

The outcome collector (`actions/setup/js/evaluate_outcomes.cjs`) discovers recent successful runs, downloads their `safe-outputs-items` manifest, evaluates each safe-output item, and records the run ID in a "seen" cache (a flat array capped at 500 entries). Every evaluated run was marked seen even when its outcomes were still `pending`, `unknown`, or `error`. Such runs were never revisited, so their eventual outcomes were permanently lost. In addition, read and listing failures were silently treated as "no data", dispatch run IDs and cross-repository PR attribution could be dropped, legacy episode lineage could be merged ambiguously, and unified sessions carried no structured parent identity from which to rebuild episodes.

### Decision

We will split collector state into two durable caches. The first is a versioned resolved-run cache (`seen-runs.json`, `{version: 2, ids}`) that holds only terminal runs. The second is a pending backlog (`pending-runs.json`) that holds runs with unresolved outcomes. Their manifests are copied to `run-<id>/safe-output-items.jsonl` so they can be revisited after the discovery window closes or the artifact expires. Any legacy array-format seen cache is discarded once, so its runs are rediscovered. Cache read errors other than ENOENT, run-listing failures, and malformed state now fail the collector rather than silently producing empty summaries. Alongside this change, we will project parent lineage (`parentRepo`, `parentRunId`, `parentRunAttempt`, `parentWorkflowId`, `parentWorkflowCallId`, `parentEventType`) from `aw_context` into unified session episode metadata. We will evaluate agent-linked PRs in the repository recorded in the issue's `repository_url`. We will order episode lineage resolution by run ID and never merge disconnected legacy `workflow_call` runs.

### Alternatives Considered

#### Alternative 1: Keep the single seen cache and skip unresolved runs

Keep the existing behavior, where every evaluated run is marked seen regardless of status. This is the simplest option and keeps state bounded. However, it permanently misreports runs whose outcomes resolve later, such as PRs merged after the collector cycle. That misreporting is the defect this PR addresses.

#### Alternative 2: Re-scan the full run history every cycle

Remove the seen cache and re-evaluate every run within a wider discovery window on each cycle. This avoids extra state, but API cost grows with history. It still cannot reach runs outside the window. It also fails once the `safe-outputs-items` artifact expires, because the manifest is no longer downloadable.

#### Alternative 3: Infer missing lineage from revision metadata

Rebuild episodes from matching repository, SHA, or ref heuristics instead of recording explicit parent identity. This needs no schema change. However, matching revision metadata does not identify a shared invocation (see the comment in `pkg/cli/logs_episode.go`), so unrelated runs could be merged. The PR explicitly chooses not to infer missing lineage.

### Consequences

#### Positive
- Outcomes that start as pending, unknown, or error are revisited until they become terminal, including runs outside the recent-run window or with expired artifacts.
- Failures in reading state or listing runs are now visible rather than appearing as "0 runs checked".
- Unified sessions carry structured parent identity, which lets episodes be reconstructed without heuristics. Older sessions without lineage remain valid.
- Cross-repository agent PRs are evaluated in the correct repository, and dispatch run IDs are preserved.

#### Negative
- The collector cache holds more state: a pending backlog plus a copied manifest for each unresolved run. The pending backlog has no explicit size cap or expiry. [TODO: verify retention policy]
- The collector now exits with an error on corrupt or unreadable cache state instead of degrading gracefully, so a bad cache needs manual intervention.
- The one-time legacy migration discards the old seen list, so previously seen runs that are still in the discovery window are evaluated again once.
- The unified session schema and both summary renderers gained new fields that must be kept in sync between the Go and JavaScript code.

#### Neutral
- Collector summaries remain per-cycle samples, now with `runs_revisited` and `unresolved_runs` counts, rather than cumulative fleet totals.
- AIC per accepted outcome is computed from observed full-run cost (`TotalAIC / Accepted`), not from per-item cost attribution.
- Shared Go/JavaScript conformance fixtures (`pkg/cli/testdata/outcome_conformance.json`) were extended to cover the new behavior.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
