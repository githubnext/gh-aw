# ADR-26321: Track Time Between Turns (TBT) as an Agentic Performance Metric with Prompt Cache TTL Warning

**Date**: 2026-04-14
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

Agentic workflow runs issue multiple sequential LLM API calls. LLM inference providers implement server-side prompt caching to avoid re-processing unchanged context on each turn; the cache has a fixed time-to-live (TTL). Anthropic recently reduced their cache TTL from 1 hour to 5 minutes. When the elapsed time between consecutive LLM calls (Time Between Turns, TBT) exceeds this TTL, the cached prompt context expires and every subsequent turn pays full re-processing costs — significantly increasing token spend. The `gh aw audit` and `gh aw logs` commands already report turn counts and wall-clock time but gave users no visibility into whether their TBT was safe relative to provider cache TTLs.

### Decision

We will add `AvgTimeBetweenTurns` and `MaxTimeBetweenTurns` as first-class fields on `LogMetrics` and surface them through the entire `audit`/`logs` pipeline. For Copilot engine runs, TBT will be computed precisely from per-turn RFC3339 timestamps embedded in `user.message` events in `events.jsonl`. For other engines (or when timestamps are absent), TBT will be estimated as `Duration / Turns`. The audit report will emit a cache warning when the maximum observed TBT exceeds the hard-coded Anthropic 5-minute TTL threshold. We chose to hard-code the 5-minute constant rather than make it configurable because it reflects an externally-imposed provider constraint, not a user preference.

### Alternatives Considered

#### Alternative 1: Derive TBT solely from wall-clock time divided by turn count

TBT could be approximated using only already-available data (`Duration / Turns`). This requires no new parsing logic and works across all engine types. It was rejected as the primary approach because it averages out spikes — a single 10-minute pause surrounded by fast turns would be invisible — whereas the precise timestamp-based approach captures both average and maximum TBT and can correctly identify individual cache-busting intervals.

#### Alternative 2: Make the cache TTL threshold configurable via `.design-gate.yml` or a CLI flag

The warning threshold (5 minutes) could be exposed as a user-configurable value to accommodate providers other than Anthropic or future TTL changes. This was rejected because the threshold is a vendor-imposed fact rather than a tunable policy; hard-coding it keeps the feature self-contained and avoids configuration sprawl. If Anthropic changes their TTL again, a code change is appropriate since the PR author will need to update documentation anyway.

#### Alternative 3: Report TBT only in `gh aw audit` (not in `gh aw logs`)

TBT could have been scoped to the single-run `audit` command only. This was rejected because `gh aw logs --json` output is consumed by downstream tooling and dashboards that benefit from per-run TBT data at the aggregated level.

### Consequences

#### Positive
- Users can identify workflow designs where slow tool calls are busting the prompt cache and costing extra tokens.
- The `CacheWarning` field provides an immediately actionable signal in the audit report.
- JSON output from `gh aw logs` gains a new `avg_time_between_turns` field for programmatic consumption.
- Precise timestamp-based TBT is computed only from engine logs that carry per-turn timestamps; other engines gracefully degrade to an estimate.

#### Negative
- The Anthropic 5-minute TTL is hard-coded; if the threshold changes, a code change is required.
- Only `user.message` events in `events.jsonl` (Copilot engine format) carry per-turn timestamps; other engine log formats will always use the estimated fallback until they add timestamp support.
- The fallback estimate (`Duration / Turns`) can be misleading for highly variable workflows where some turns are fast and others are slow.

#### Neutral
- Two new fields (`AvgTimeBetweenTurns`, `MaxTimeBetweenTurns`) are added to `LogMetrics`, `WorkflowRun`, `SessionAnalysis`, and `RunData` — widening the data model across four structs.
- Tests are scoped to the Copilot engine JSONL parser; audit/logs integration tests are unaffected.
- The feature is additive and non-breaking: existing consumers of `gh aw logs --json` will see a new optional field (`avg_time_between_turns`) that is omitted when TBT cannot be computed.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### TBT Computation

1. Implementations **MUST** compute TBT as the elapsed time between consecutive LLM API call initiations (one interval per pair of adjacent turns).
2. Implementations **MUST** compute both average TBT (`AvgTimeBetweenTurns`) and maximum TBT (`MaxTimeBetweenTurns`) when two or more turn timestamps are available.
3. Implementations **MUST NOT** report a non-zero TBT value when fewer than two turns are recorded.
4. Implementations **MUST** use per-turn RFC3339 timestamps from `user.message` events in `events.jsonl` as the primary source for TBT computation when those timestamps are available.
5. Implementations **SHOULD** fall back to `Duration / Turns` as an estimated average TBT when per-turn timestamps are absent or when only one unique timestamp value is present; the estimated value **MUST** be labelled `(estimated)` in human-readable output.
6. Implementations **MUST** ignore zero-duration intervals (timestamps that are identical) when computing TBT averages and maximums to avoid artefacts from log files that reuse a single timestamp.

### Cache Warning

1. Implementations **MUST** emit a cache warning when the maximum observed TBT exceeds the Anthropic 5-minute prompt cache TTL (300 seconds).
2. Implementations **SHOULD** emit a cache warning when the average TBT exceeds 300 seconds and the maximum is unavailable.
3. Implementations **MUST NOT** emit a cache warning when both average and maximum TBT are zero or absent.
4. The cache warning **MUST** include the observed TBT value and state that the Anthropic 5-minute cache TTL is being exceeded.

### Data Model and Propagation

1. Implementations **MUST** store `AvgTimeBetweenTurns` and `MaxTimeBetweenTurns` as `time.Duration` fields on `LogMetrics`.
2. Implementations **MUST** propagate `AvgTimeBetweenTurns` from `LogMetrics` through `WorkflowRun` and into `RunData` for JSON output.
3. Implementations **MUST** expose `avg_time_between_turns` as an `omitempty` JSON field in `RunData` so that existing consumers are unaffected when TBT is unavailable.
4. Implementations **SHOULD** expose both average and maximum TBT in the `SessionAnalysis` struct used by the `audit` command's report renderer.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24427002630) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
