# ADR-47987: Extract Audit Diff Helper Functions to Satisfy Function-Length Linter

**Date**: 2026-07-25
**Status**: Draft
**Deciders**: copilot-swe-agent, pelikhan

---

### Context

`pkg/cli/audit_diff.go` contains two functions — `computeFirewallDiff` (144 lines) and `computeMCPToolsDiff` (80+ lines) — that exceeded the project's `golint-custom` function-length limit of 60 lines. The linter was reporting 662 total violations across `pkg/workflow` and `pkg/cli`. These functions are core to the audit-comparison workflow, computing diff entries for firewall domain statistics and MCP tool usage across two workflow runs. The project's lint discipline requires violations to be resolved by structural refactoring rather than suppression.

### Decision

We will extract focused helper functions from `computeFirewallDiff` and `computeMCPToolsDiff`, decomposing each into small, single-responsibility helpers (`firewallStatsByRun`, `firewallSortedDomains`, `appendFirewallDomainDiff`, `appendFirewallExistingDomainDiff`, `buildFirewallDiffSummary`, `mcpSummaryMap`, `mcpSortedKeys`, `appendMCPToolDiff`). The extraction preserves all existing behavior and anomaly-detection logic while bringing each orchestrating function within the 60-line limit.

### Alternatives Considered

#### Alternative 1: Suppress the Linter with nolint Annotations

Add `//nolint:function-length` comments to the affected functions to silence violations without structural change. Rejected because it permanently exempts the functions from the linter, accumulates technical debt, and undermines the project's lint discipline for all future contributors working in these files.

#### Alternative 2: Rewrite Using a Table-Driven or Strategy Pattern

Refactor the diff logic with a table-driven dispatch or a `DiffClassifier` interface per domain-event type. Rejected as disproportionate to the goal: the existing if-else/switch logic is well-understood, and introducing abstraction would increase indirection without meaningfully improving testability at this scale.

#### Alternative 3: Split audit_diff.go into Separate Files per Diff Type

Move each `compute*Diff` function and its helpers into its own file (e.g., `audit_diff_firewall.go`, `audit_diff_mcp.go`). Not chosen because the functions share internal types, constants, and the `auditDiffLog` package-level logger — co-locating them in one file remains more cohesive for the current codebase structure.

### Consequences

#### Positive
- Both `computeFirewallDiff` and `computeMCPToolsDiff` are no longer flagged by the function-length linter, reducing the shared backlog from 662 to 660 findings (a net reduction of 2).
- Each extracted helper has a single, well-named responsibility, making the logic easier to read and independently testable.
- The extraction pattern establishes a clear template for follow-up refactors of remaining violators in the same file.

#### Negative
- The file gains eight new package-private functions, increasing total function count and making the file longer by net line count even after deletions.
- Helpers such as `firewallStatsByRun` and `mcpSummaryMap` are trivially small — their extraction adds naming and call-site overhead that some reviewers may consider excessive relative to the complexity they encapsulate.

#### Neutral
- External callers of `computeFirewallDiff` and `computeMCPToolsDiff` are unaffected — the public API of `audit_diff.go` is unchanged.
- The three remaining violators in the same file (`computeRunMetricsDiff`, `computeToolCallsDiff`, `computeTokenUsageDiff`) are candidates for follow-up refactoring using the same extraction pattern established here.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
