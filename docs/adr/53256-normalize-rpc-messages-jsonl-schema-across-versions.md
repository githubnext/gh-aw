# ADR-53256: Normalize rpc-messages.jsonl Schema Across Versions via EffectiveType

**Date**: 2026-08-17
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `rpc-messages.jsonl` telemetry file that records MCP gateway traffic exists in two incompatible schemas. The legacy schema uses a top-level `type` field with values `REQUEST`, `RESPONSE`, and `DIFC_FILTERED`. Production telemetry written by the real Copilot CLI MCP gateway (schema `rpc-message/v2`) instead uses a top-level `event` field with values `rpc_request`, `rpc_response`, and `difc_filtered`, plus a `_schema` marker. Investigation of 20 real workflow runs confirmed that every run using the `rpc-messages.jsonl` fallback path (when `gateway.jsonl` is absent) emits schema `rpc-message/v2` exclusively — meaning 0 of 202 entries exposed the `type` field the parser expected, silently zeroing all tool-call counts and durations.

### Decision

We will introduce a thin normalization layer — `EffectiveType()` in Go and `getRpcMessageType()` in JavaScript — that transparently bridges the two schemas at every call site that previously compared `entry.Type` or `entry.type` directly. The legacy `type` field takes precedence when both fields are present; otherwise, `event` values are mapped to their `type` equivalents (`rpc_request` → `REQUEST`, `rpc_response` → `RESPONSE`, `difc_filtered` → `DIFC_FILTERED`). The `RPCMessageEntry` struct gains `Event` and `Schema` fields for JSON deserialization. All existing `entry.Type ==` comparisons across `gateway_logs_rpc.go`, `gateway_logs_timeline.go`, and `parse_mcp_gateway_log.cjs` are replaced with calls to the normalization helper.

### Alternatives Considered

#### Alternative 1: Migrate parsers to schema v2 exclusively

Rewrite all parsers to read `event`/`_schema` only and drop the legacy `type` field entirely. This would be the cleaner long-term path if the `type` field were truly dead. Rejected because synthetic test fixtures, CI-generated artifacts, and any older production files that still emit `type` would silently break — there is no authoritative signal that the legacy schema is gone from all producers.

#### Alternative 2: Normalize at ingestion by always mutating `entry.type`

Overwrite `entry.type` with the normalized value on every parsed entry regardless of whether `type` was already set. Simpler call sites — downstream code never needs to call a helper. Rejected because it risks clobbering an entry's own explicit `type` value in a mixed-schema edge case, and it mutates data that may be forwarded or logged downstream. The JS parser does back-fill `entry.type` when absent (for downstream consumers that still read the field directly), but only when the field is not already present.

### Consequences

#### Positive
- Tool-call counts and durations are now correctly computed for all production workflow runs that emit schema `rpc-message/v2`, recovering previously silent zeroes across 20/20 sampled runs.
- Both schema versions are supported transparently without branching at every call site — adding the `rpc-message/v2` schema fields to the struct is backward-compatible with all existing tests and consumers.
- Regression test coverage added in Go and JavaScript for all three entry types (`REQUEST`/`RESPONSE`/`DIFC_FILTERED`) under both schemas.

#### Negative
- Two parallel representations of the same concept (`type` and `event`) now coexist in the data model and codebase indefinitely, increasing conceptual overhead for future contributors.
- The precedence rule (`type` wins when both are set) is an implicit contract that future telemetry schema versions must respect; violating it would silently produce incorrect results.

#### Neutral
- The `entry.type` back-fill in the JS parser (`if (!entry.type) entry.type = messageType`) is a shim for downstream consumers that still read `entry.type` directly; removing it in the future would require auditing those consumers.
- Documentation in `daily-observability-report.md` (Phase 3.4) is updated to reflect the dual-schema reality and explicitly instructs against flagging runs unhealthy solely for missing `type`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
