# ADR-26278: Split logs_report.go into Domain-Focused Files

**Date**: 2026-04-14
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

`pkg/cli/logs_report.go` had grown to 1,065 lines containing 15+ independent builder functions covering four distinct reporting domains (tool usage, MCP, firewall/access logs, and errors), with no shared state between them. This made the file difficult to navigate—contributors had to scroll through hundreds of lines to find the function relevant to the domain they were working in. The file was a classic "God file" anti-pattern within a single Go package.

### Decision

We will split `pkg/cli/logs_report.go` into five files within the same `cli` package, each owning one reporting domain: `logs_report_tools.go`, `logs_report_mcp.go`, `logs_report_firewall.go`, `logs_report_errors.go`, and a reduced `logs_report.go` that retains top-level orchestration and core data types. All files remain in the same Go package (`package cli`), so no import paths or public APIs change. No logic is modified; this is a pure structural reorganization to improve file-level navigability.

### Alternatives Considered

#### Alternative 1: Keep Everything in a Single File

The file could remain as-is. This is the simplest option—no merge conflicts, no navigation changes. It was rejected because 1,065 lines with no shared state between sections makes file navigation genuinely painful; finding a specific builder requires either text search or scrolling through unrelated sections.

#### Alternative 2: Extract into a Separate Sub-Package

The reporting domains could have been moved into a dedicated sub-package (e.g., `pkg/cli/logsreport/`). This would provide stronger compile-time boundaries and make the domain separation visible at the import level. It was not chosen because the builder functions reference unexported types in `cli` and moving them would require exporting those types or significantly restructuring the package boundary—a change well beyond the scope of the navigability problem being solved.

#### Alternative 3: Split by Concern Type (Types vs. Builders)

An alternative structure would group all `*Summary` types in one file and all `build*` functions in another, regardless of domain. This was rejected because it does not improve navigability for the primary use case: understanding or modifying all logic related to one domain (e.g., firewall log analysis). Domain grouping keeps related types and their builders collocated.

### Consequences

#### Positive
- Each new file is ≤200 lines, well within the team's navigability threshold.
- Contributors working on one reporting domain (e.g., firewall) can open a single focused file.
- `logs_report.go` is reduced from 1,065 to 417 lines, with the remaining bulk attributable to `buildLogsData` (the orchestration function kept intact per the issue specification).
- No API surface changes—callers outside the package are unaffected.

#### Negative
- The codebase now has more files, which can add overhead when searching for types or functions across the package for the first time.
- Cross-domain relationships (e.g., a type from `logs_report_tools.go` used in `logs_report.go`) are less immediately visible than when everything was in one file.

#### Neutral
- Go's intra-package visibility means all unexported identifiers remain accessible across the split files; no visibility changes are needed.
- The `buildLogsData` function in `logs_report.go` (~191 lines) remains the largest single unit and is a candidate for future decomposition if complexity grows.
- IDE tooling and `go build` are unaffected by intra-package file splits.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### File Organization

1. Implementations **MUST** keep all `logs_report_*.go` files in the same Go package (`package cli`).
2. Each domain-specific file **MUST** own all types and builder functions for that domain, and **MUST NOT** contain builder functions belonging to another domain.
3. Implementations **MUST NOT** introduce a new sub-package solely to house the split files; the existing `pkg/cli` package boundary **SHALL** be maintained.
4. Implementations **SHOULD** keep each `logs_report_*.go` file under 250 lines; if a file grows beyond this threshold, it **SHOULD** be further decomposed or the split reconsidered.

### Orchestration and Core Types

1. The top-level orchestration function `buildLogsData` and the core data types (`LogsData`, `LogsSummary`, `RunData`, `ContinuationData`) **MUST** remain in `logs_report.go`.
2. Implementations **MUST NOT** duplicate type definitions across split files; each type **SHALL** be defined in exactly one file.
3. Render functions (`renderLogsConsole`, `renderLogsJSON`) and `writeSummaryFile` **MUST** remain in `logs_report.go` alongside the orchestration layer.

### Domain-to-File Mapping

1. Tool-usage types and builders (`ToolUsageSummary`, `isValidToolName`, `buildToolUsageSummary`) **MUST** reside in `logs_report_tools.go`.
2. MCP builders (`buildMCPFailuresSummary`, `buildMCPToolUsageSummary`) **MUST** reside in `logs_report_mcp.go`.
3. Firewall and access-log types and builders (`AccessLogSummary`, `FirewallLogSummary`, `domainAggregation`, `aggregateDomainStats`, `convertDomainsToSortedSlices`, `buildAccessLogSummary`, `buildFirewallLogSummary`, `buildRedactedDomainsSummary`) **MUST** reside in `logs_report_firewall.go`.
4. Error-summary types and builders (`ErrorSummary`, `addUniqueWorkflow`, `aggregateSummaryItems`, `buildCombinedErrorsSummary`, `buildMissingToolsSummary`, `buildMissingDataSummary`) **MUST** reside in `logs_report_errors.go`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance. The primary conformance checks are: (a) all split files are in `package cli`, (b) no new sub-package is created, (c) each domain's builders and types are collocated in their designated file, and (d) orchestration and core types remain in `logs_report.go`.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24422316567) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
