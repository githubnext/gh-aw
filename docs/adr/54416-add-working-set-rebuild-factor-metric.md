# ADR-54416: Add Working-Set Rebuild Factor as a Run-Level Observability Metric

**Date**: 2026-08-21
**Status**: Draft
**Deciders**: pelikhan

---

### Context

The gh-aw system runs coding agents that make multiple sequential LLM invocations per run. Each invocation incurs a context reconstruction cost proportional to the total input tokens sent to the model. There is currently no metric that captures cumulative context reconstruction overhead relative to the peak single-invocation context size. Without such a signal, engineers cannot distinguish runs where context grew steadily (healthy progressive work) from runs where context was rebuilt from scratch many times (potentially inefficient). The metric is conceptually inspired by the "Working Set of a Coding Agent" paper (arXiv:2608.16630), which introduces "coherence debt" as a measure of context reconstruction overhead.

### Decision

We will compute a Working-Set Rebuild Factor (WSRF) defined as `sum(input_tokens) / max(input_tokens)` across all per-invocation records in the agent's `token_usage.jsonl` file. The metric will be written to the usage activity summary (`usage/activity/summary.json`) during the conclusion job, emitted as `gh-aw.working_set.*` OTLP span attributes on the built-in conclusion span, and surfaced in `gh aw logs` and `gh aw audit` CLI outputs. The metric will use the canonical `input_tokens` field only — cache-read and cache-write fields are intentionally excluded because the provider normalization layer already produces `input_tokens` as the logical input count. The metric is explicitly declared an efficiency/trajectory indicator, not a measurement of semantic coherence or task success.

### Alternatives Considered

#### Alternative 1: Include cache token fields in the WSRF computation

Add `input_tokens + cache_read_tokens + cache_write_tokens` as the base measurement per invocation. This would capture total traffic to the provider rather than just the logical input. Rejected because provider normalization already folds cache semantics into `input_tokens` as a canonical logical count; re-adding cache fields would double-count tokens and produce a metric with no well-defined interpretation across providers.

#### Alternative 2: Derive WSRF from firewall proxy logs instead of token_usage.jsonl

Compute the metric from per-request data captured by the Squid-based firewall proxy, which records all outbound LLM traffic. Rejected because proxy logs are only available when the agent routes through the firewall, creating measurement gaps for runs that bypass it or for providers with direct integrations. The `token_usage.jsonl` file is always written by the agent phase regardless of routing, making it a more reliable data source.

### Consequences

#### Positive
- Provides a quantitative efficiency signal (WSRF) enabling engineers to compare context-building patterns across runs and over time via `gh aw audit` diff output.
- Gracefully degrades: reports `measurement_state: unavailable` when data is missing, `partial` when some records are malformed — never fabricating a factor of zero or one.
- OTLP attribute emission enables dashboard aggregation and alerting without requiring CLI access.
- BigInt arithmetic in the JS computation prevents integer overflow for runs with very large cumulative token counts (above `Number.MAX_SAFE_INTEGER`).

#### Negative
- WSRF is not a proxy for task success or missing-context quality; equal factors can appear on both successful and failed runs. Misinterpreting it as a quality gate would be misleading.
- The metric is only computed in the conclusion job and depends on `token_usage.jsonl` being present in the agent artifact; runs where the artifact is unavailable or incomplete will show `unavailable` state.

#### Neutral
- The `working_set` field is added to the existing `usage-activity-summary/v1` JSON schema as an optional additive section, preserving backwards compatibility with consumers that do not yet read the field.
- The OTLP attributes are emitted only on the built-in conclusion span, not on per-invocation agent spans, keeping span cardinality unchanged.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
