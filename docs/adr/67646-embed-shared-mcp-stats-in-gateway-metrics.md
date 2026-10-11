# ADR-67646: Embed Shared MCP Stats Base in Gateway Server Metrics

**Date**: 2026-10-10
**Status**: Draft
**Deciders**: pelikhan (PR author), gh-aw maintainers

---

### Context

`pkg/cli` has four report types that describe per-MCP-server usage: `MCPServerStats`, `MCPServerHealthDetail`, `MCPServerCrossRunHealth`, and `GatewayServerMetrics`. The first three already embed `MCPServerStatsBase` (`pkg/cli/logs_models.go`), a struct introduced specifically to give one Go spelling to `ServerName`, `ToolCallCount`, and `ErrorCount` after those concepts had drifted into four variants (`TotalCalls`/`ToolCalls`/`ToolCallCount`, `TotalErrors`/`ErrorCount`). `GatewayServerMetrics` (`pkg/cli/gateway_logs_types.go`) was the last straggler, re-declaring the same three fields by hand (tracked in issue #67199). The constraint is that gateway metrics are serialized to JSON with legacy capitalized keys and always-present zero values, and that existing persisted gateway log artifacts must keep decoding correctly.

### Decision

We will embed `MCPServerStatsBase` in `GatewayServerMetrics` and keep the gateway-specific fields (`RequestCount`, `TotalDuration`, `FilteredCount`, `GuardPolicyBlocked`, `Tools`) alongside it. Because the embedded struct's own JSON tags differ from the gateway's legacy wire format, the embedded field is tagged `json:"-" console:"-"` and the type gains explicit `MarshalJSON`/`UnmarshalJSON` methods that project `ServerName`, `ToolCallCount`, and `ErrorCount` back onto the historical top-level capitalized keys. The primary driver is type-name consistency across the MCP metrics surface with strictly zero change to the serialized schema.

### Alternatives Considered

#### Alternative 1: Leave `GatewayServerMetrics` with hand-declared fields

The status quo requires no custom marshalling and no risk to the wire format. It was rejected because it preserves the exact drift that `MCPServerStatsBase` was created to eliminate, leaving a fourth independent spelling of the same concepts and keeping the base struct's documented rationale only partially realized.

#### Alternative 2: Embed the base and adopt its JSON tags directly (no custom marshallers)

Embedding without overrides would be the simplest code, but Go would then promote the base struct's own tag names and omit-empty behaviour into the gateway payload. That changes key names and drops zero-valued counters, breaking existing gateway metric artifacts and the table renderer's round-trip. Rejected as a breaking serialization change for a pure naming cleanup. This was a close call: it would be preferable if the gateway format were not already published.

#### Alternative 3: Convert at the boundary with a separate DTO

A dedicated serialization DTO could keep both the domain struct and the wire format clean without method overrides on the domain type. Rejected as more code and an extra mapping layer for three fields, with the same maintenance burden the consolidation is trying to reduce.

### Consequences

#### Positive
- One Go spelling for `ServerName`/`ToolCallCount`/`ErrorCount` across all four MCP metrics types; the consolidation started in `logs_models.go` is complete.
- Shared statistics can now be read and updated generically wherever `MCPServerStatsBase` is accepted, including gateway data.
- Regression coverage added for legacy key JSON output, zero-value retention, partial/null decoding, duplicate-key decoding, and table rendering after a serialization round trip.

#### Negative
- `GatewayServerMetrics` now carries hand-written `MarshalJSON`/`UnmarshalJSON` methods; the wire format is no longer derivable from struct tags alone and must be kept in sync by hand when fields are added.
- The struct-embedding literal form (`MCPServerStatsBase{...}`) is more verbose at every construction site, including tests.

#### Neutral
- All gateway construction sites (`getOrCreateServer` and tests) were updated mechanically; no behavioural change is intended.
- The custom `UnmarshalJSON` preserves pre-existing field values when a key is absent or `null`, which matches the previous embedded-decode behaviour but is now explicit and test-pinned.
- `MCPServerStatsBase`'s doc comment was updated to list `GatewayServerMetrics` among its embedders.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
