# ADR-53693: Share MCPServerStatsBase Across MCP Server Health/Stats Structs

**Date**: 2026-08-18
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Three structs in `pkg/cli` — `MCPServerStats` (`audit_report.go`), `MCPServerHealthDetail` (`audit_expanded.go`), and `MCPServerCrossRunHealth` (`audit_cross_run.go`) — all model the same "per-server identity and volume" shape: a server name, a call count, and an error count. Despite modeling the same concepts, they used four different field spellings (`TotalCalls` / `ToolCalls` / `ToolCallCount` and `TotalErrors` / `ErrorCount`), creating reader confusion and copy-paste drift risk. The codebase already solved this exact problem for `MissingToolSummary` / `MissingDataSummary` / `MCPFailureSummary` via `AggregatedSummaryBase` in `pkg/cli/logs_models.go`.

### Decision

We will introduce `MCPServerStatsBase` in `pkg/cli/logs_models.go`, adjacent to `AggregatedSummaryBase`, and embed it in `MCPServerStats`, `MCPServerHealthDetail`, and `MCPServerCrossRunHealth`. The base struct standardizes Go field names (`ServerName`, `ToolCallCount`, `ErrorCount`) and carries the canonical JSON/console tags. Types whose serialized wire format differs from these tags preserve their schema via a `MarshalJSON` override — the same technique already used by `MCPFailureSummary`.

### Alternatives Considered

#### Alternative 1: Keep Structs Independent

Leave each struct with its own field definitions and accept the inconsistency. This avoids any structural change and keeps struct literals flat.

Rejected because the same problem was already solved for the failure-summary types via `AggregatedSummaryBase`. Leaving the server stats types inconsistent grows the two-tier inconsistency and means any future shared stat (e.g., a new latency field) would again require four synchronized changes.

#### Alternative 2: Rename Fields Uniformly Without a Shared Base

Standardize field names across all structs individually (e.g., rename all to `ToolCallCount` / `ErrorCount`) without extracting a base struct.

Rejected because this still leaves four copies of the same fields to maintain. Future additions would need to touch all four structs, and the compiler cannot enforce that they stay in sync. The base struct approach is already an established idiom in this package.

### Consequences

#### Positive
- Eliminates four-way field name inconsistency; callers and readers only need to learn one name for each concept.
- Follows the existing `AggregatedSummaryBase` pattern, so the approach is validated and immediately recognizable to package maintainers.
- The compiler enforces that shared fields exist and have the correct types — future drift between the three embedding types is caught at compile time.

#### Negative
- `MCPServerHealthDetail` and `MCPServerCrossRunHealth` require explicit `MarshalJSON` overrides to emit their original wire-format keys (`tool_calls`, `total_calls`, `total_errors`), adding implementation surface and tests.
- Struct literals at call sites and in tests must use explicit `MCPServerStatsBase{…}` nesting, making initialization slightly more verbose than flat field assignment.

#### Neutral
- `MCPServerHealth` (a rollup over the `Servers` slice) is explicitly excluded: it has no server name field and its counts are aggregates rather than per-server stats, so the base does not apply.
- The `ErrorCount` field retains `omitempty` in the base because `MCPServerStats` (the only embedder that serializes base tags directly) requires it; the other embedders override `MarshalJSON` and are unaffected by the tag.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
