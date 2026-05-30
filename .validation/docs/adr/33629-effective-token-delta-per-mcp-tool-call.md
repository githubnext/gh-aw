# ADR-33629: Effective-Token Delta per MCP Tool Call via Timestamp Correlation

**Date**: 2026-05-20
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

Each MCP tool call result is appended to the LLM context window on the next turn, which directly increases per-turn token cost. The `gh aw audit` Go CLI and the GitHub Actions step-summary (`parse_mcp_gateway_log.cjs`) both surface MCP tool call timelines, but neither rendering exposes the cost of each tool call's payload in the next API call. The gateway log (`rpc-messages.jsonl`) records tool call timestamps, and `token-usage.jsonl` records every LLM API call with input/output/cache token counts and model identifiers from which an effective-token weighting is already computed elsewhere in the codebase. The two files are written independently and have no shared correlation key.

### Decision

We will compute an effective-token delta (`ΔET`) for each MCP tool call by **timestamp-bracketing** against the existing `token-usage.jsonl` stream. For each tool call at timestamp `T`, we locate `prev` (the last token-usage entry with `ts < T`) and `next` (the first token-usage entry with `ts > T`), then assign `ΔET = effectiveTokens(next) − effectiveTokens(prev)`. The delta is rendered as a new `ΔET` column in the JS step-summary REQUEST table and as a separate "Tool Call Timeline (Effective Token Δ)" table in the Go CLI MCP tool usage report. When `token-usage.jsonl` is missing, unreadable, or insufficient (fewer than two entries), the delta is silently omitted — the feature is strictly additive and non-fatal.

### Alternatives Considered

#### Alternative 1: Measure tool call result payload size directly (bytes or local tokenization)

We could instead size the raw tool call result payload from `rpc-messages.jsonl` and convert that to tokens via a local tokenizer. This was rejected because (a) it ignores model-specific effective-token weightings (cache reads, cache writes, output tokens) already centralized in `computeEffectiveTokens` / `computeModelEffectiveTokensWithWeights`, (b) it cannot account for cache eviction, system-prompt resends, or context pruning between turns, and (c) it would require introducing a tokenizer dependency to JS and Go for results the LLM provider's own token accounting already attributes precisely.

#### Alternative 2: Add a correlation ID linking each MCP request to the LLM API call that consumed it

The MCP gateway and the LLM client could be modified to share a correlation ID (e.g., a turn ID), and `token-usage.jsonl` could record which tool call results it consumed. This would yield exact attribution rather than timestamp-based brackets. It was rejected for this PR because it requires coordinated changes across the gateway, the LLM client, and every engine adapter; timestamp correlation is sufficient for the common case (sequential calls) and uses only existing log data. A correlation-ID approach remains a future option if timestamp bracketing proves inaccurate in practice.

#### Alternative 3: Compute the delta inside the audit pipeline only and skip the JS step summary

We could limit the new feature to the Go CLI audit/logs path. This was rejected because the GitHub Actions step summary is the most visible artifact during a workflow run and is where users first notice expensive tool calls — adding the column there has the highest impact-per-byte. Duplicating the correlation algorithm in Go and JS is intentional: each environment has its own log-reading pipeline, and the algorithm is small and well-tested in both.

### Consequences

#### Positive
- Users see, per tool call, how much effective-token cost the tool's result added to the next API call.
- Implementation reuses existing effective-token weight computation (`computeEffectiveTokens` in JS, `computeModelEffectiveTokensWithWeights` in Go), so caching, model multipliers, and output weighting are honored consistently.
- Feature is fully additive and degrades gracefully: when `token-usage.jsonl` is absent or has fewer than two entries, no `ΔET` column is rendered and no error is reported.
- Unit tests cover the bracketing algorithm in both Go (`TestCorrelateToolCallsWithTokenDelta`) and JS (`computeToolCallTokenDeltas`), including the no-prev, no-next, multi-call, and empty-input cases.

#### Negative
- Timestamp-bracketing is approximate: when multiple tool calls execute between two consecutive API calls, the entire ET delta is attributed to none of them (since both share the same `prev` and `next`) — only tool calls bracketed by distinct API call pairs receive a delta.
- The algorithm is duplicated across two languages (Go and JS); fixes to the bracketing logic must be made in two places.
- The delta conflates tool-result cost with any other context growth between the two API calls (e.g., system reminders, growing message history) — it is an upper bound on the tool's contribution, not an exact measurement.
- A clock-skew or out-of-order log scenario (timestamps in `rpc-messages.jsonl` not aligned with `token-usage.jsonl`) silently produces wrong attribution.

#### Neutral
- A new `EffectiveTokenDelta int` field is added to the `MCPToolCall` Go struct with `omitempty` JSON serialization, so JSON consumers see no breaking change.
- The JS REQUEST table grows a fourth column (`ΔET`) only when at least one delta is computed; the existing three-column form is preserved when no deltas exist.
- Two new exported symbols are added to the JS module (`computeToolCallTokenDeltas`) and the Go `pkg/cli` package (`readTokenUsageEntries`, `correlateToolCallsWithTokenDelta`).
- Deltas with value `≤ 0` are not rendered, so a tool call followed by a smaller API call (e.g., due to cache eviction recovery) is silently shown as a dash, not a negative value.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Delta Computation

1. Implementations **MUST** compute each MCP tool call's `ΔET` as `effectiveTokens(next) − effectiveTokens(prev)`, where `prev` is the last `token-usage.jsonl` entry with timestamp strictly less than the tool call timestamp and `next` is the first entry with timestamp strictly greater than the tool call timestamp.
2. Implementations **MUST** use the project's existing effective-token weighting routines (`computeEffectiveTokens` in JS, `computeModelEffectiveTokensWithWeights` in Go) to compute `effectiveTokens(entry)`; they **MUST NOT** introduce a divergent token weighting for this feature.
3. Implementations **MUST NOT** assign a `ΔET` value to a tool call that lacks either a `prev` or a `next` token-usage entry.
4. Implementations **MUST NOT** render a `ΔET` value that is less than or equal to zero; such values **MUST** be treated as "no delta available" for that tool call.
5. Implementations **MUST** be silent and non-fatal when `token-usage.jsonl` is missing, unreadable, malformed, or contains fewer than two parseable entries — they **MUST** continue rendering the tool call timeline without a delta column.

### Rendering

1. Go CLI implementations **MUST** render a separate "Tool Call Timeline (Effective Token Δ)" table containing only tool calls whose `EffectiveTokenDelta` is greater than zero.
2. JS step-summary implementations **MUST** render a `ΔET` column in the REQUEST table only when at least one tool call has a positive delta; otherwise the existing three-column REQUEST table layout **MUST** be preserved.
3. Tool calls in the JS REQUEST table that do not have a positive delta **MUST** display `-` in the `ΔET` column when the column is present.
4. Delta values **SHOULD** be formatted with a leading `+` sign and thousands separators (e.g., `+1,234`) in user-facing output.

### Data Model

1. The Go `MCPToolCall` struct **MUST** include an `EffectiveTokenDelta int` field with the JSON tag `effective_token_delta,omitempty`.
2. The Go correlation routine `correlateToolCallsWithTokenDelta` **MUST** return a new slice (not mutate the input) and **MUST** be invoked for both code paths in `extractMCPToolUsageData` (the `rpc-messages.jsonl` path and the `gateway.jsonl` path).
3. The JS correlation routine `computeToolCallTokenDeltas` **MUST** return a `Map` keyed by request index, and **MUST** omit entries whose computed delta is less than or equal to zero.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26193570188) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
