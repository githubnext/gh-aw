# ADR-54387: Consolidate Duplicate PR and Tool Usage Models

**Date**: 2026-08-21
**Status**: Draft
**Deciders**: Unknown

---

### Context

`pkg/cli` maintained two separate struct definitions for the same pull-request concept: `PRInfo` (in `pr_command.go`, used by `gh aw pr` commands) and `PullRequest` (in `pr_automerge.go`, used by automerge logic). The two structs shared a subset of fields (`Number`, `Title`) but diverged on others, allowing the schemas to drift independently over time.

Similarly, `ToolUsageSummary` (generic tool reporting) and `MCPToolSummary` (MCP tool reporting) each declared their own `ToolName`, `CallCount`, `MaxOutputSize`, and `MaxDuration` fields. With no shared definition, any field addition or rename had to be applied twice and synchronized manually.

### Decision

We decided to introduce a shared `ToolUsageStatsBase` embedded struct containing the common tool identity and metrics fields (`ToolName`, `CallCount`, `MaxOutputSize`, `MaxDuration`), embed it in both `ToolUsageSummary` and `MCPToolSummary`, and merge `PRInfo` and `PullRequest` into a single superset `PullRequest` struct in `pr_command.go` with `PRInfo` retained as a type alias (`type PRInfo = PullRequest`) for backward compatibility. The primary driver is eliminating schema drift between conceptually identical model fields and reducing the risk of callers diverging in future.

### Alternatives Considered

#### Alternative 1: Field Synchronization Tests

Add reflection-based tests asserting that both tool summary structs declare identical common fields, leaving the dual-struct layout intact. This prevents undetected drift but does not eliminate the duplication itself. It was rejected because it adds test maintenance burden without removing the root cause.

#### Alternative 2: Interface-Based Polymorphism

Define a `ToolStatsProvider` interface that each type satisfies independently, and write functions against the interface rather than a concrete base type. This avoids embedding but requires an interface method set for what are essentially plain data fields, which is idiomatic only when behavior varies. Rejected because the shared content is pure data with no behavioral differences between the two types.

### Consequences

#### Positive
- Single source of truth for shared tool metrics fields — additions or renames are applied once in `ToolUsageStatsBase`
- `PRInfo = PullRequest` type alias is a zero-cost compatibility shim requiring no call-site changes
- Reflection tests (`TestToolUsageSummariesShareStatsBase`) can now assert the embedding contract rather than field-by-field matching

#### Negative
- `ToolUsageSummary` requires custom `MarshalJSON`/`UnmarshalJSON` to preserve the existing `name`/`total_calls` JSON schema, adding serialization complexity that must be maintained when new fields are added to `ToolUsageStatsBase`
- Embedded struct initialization syntax (`ToolUsageStatsBase{ToolName: ..., CallCount: ...}`) is more verbose than flat struct literal initialization, increasing the size of test fixtures

#### Neutral
- `MCPToolSummary` JSON schema is preserved unchanged because the promoted `ToolUsageStatsBase` fields already carry the correct JSON tags (`tool_name`, `call_count`)
- The `PullRequest` superset struct now carries fields from both former structs; callers that only populate a subset of fields continue to compile and behave correctly

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
