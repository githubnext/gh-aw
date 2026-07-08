# ADR-44051: Split logs_orchestrator.go into Focused Modules

**Date**: 2026-07-07
**Status**: Draft
**Deciders**: Unknown (generated from PR #44051 diff)

---

### Context

`pkg/cli/logs_orchestrator.go` had grown to 1284 lines, mixing orchestration, filter logic, type definitions, output rendering, and stdin processing in a single file. Approximately 150 lines of run-filter logic and `ProcessedRun` construction were duplicated verbatim between `DownloadWorkflowLogs` and `DownloadWorkflowLogsFromStdin`. Any bug fix or behavioral change to the filter pipeline had to be applied in two places, creating an active maintenance risk. The codebase uses the `pkg/cli` package for all command-line surface area; Go does not require splitting a package across multiple files, but readability and testability suffer when a single file grows past ~300–400 lines.

### Decision

We will split `logs_orchestrator.go` into five focused files within the same `cli` package: `logs_orchestrator_types.go` (option structs and internal types), `logs_orchestrator_filters.go` (filter helpers `applyRunFilters` and `buildProcessedRun`), `logs_orchestrator_render.go` (output rendering), `logs_orchestrator_stdin.go` (`DownloadWorkflowLogsFromStdin`), and a trimmed `logs_orchestrator.go` retaining only the pagination loop and `DownloadWorkflowLogs`. The public API (`DownloadWorkflowLogs`, `DownloadWorkflowLogsFromStdin`, `LogsDownloadOptions`, `StdinLogsOptions`) is unchanged. Unit tests for the newly extracted helpers are added in `logs_orchestrator_filters_test.go`.

### Alternatives Considered

#### Alternative 1: Deduplicate helpers without file splitting

Introduce `applyRunFilters` and `buildProcessedRun` as private helpers inside `logs_orchestrator.go` without moving any code to new files. This eliminates the duplication and is the minimal change, but the file would remain ~900 lines with type definitions, rendering, stdin processing, and the core pagination loop still entangled. Navigation and isolated testing of each concern would still be difficult.

#### Alternative 2: Move each concern into a separate Go package

Create sub-packages such as `pkg/cli/logsfilters` and `pkg/cli/logsrender`. This enforces hard encapsulation boundaries at the package level and prevents accidental coupling. However, it requires exporting all shared types (e.g., `DownloadResult`, `ProcessedRun`) that are currently package-private, widens the public API surface, and increases the cost of future changes that cross the new package boundaries. Given the single-package nature of the CLI layer, the encapsulation benefit does not justify the added complexity.

### Consequences

#### Positive
- Eliminates ~150 lines of duplicated filter and `ProcessedRun` construction code, so future filter changes are applied once and tested once.
- Smaller, single-concern files (87–252 lines each) are faster to navigate, review, and modify independently.
- The new `applyRunFilters` and `buildProcessedRun` helpers are directly unit-testable without exercising the full download pipeline.
- No callers outside the package need to change; the public API is stable.

#### Negative
- Tracing the complete logic for a single download path (e.g., `DownloadWorkflowLogs`) now requires opening multiple files rather than scrolling within one.
- If the filter semantics for the stdin path and the standard path ever need to diverge significantly, the shared `applyRunFilters` function becomes a constraint that must be refactored.

#### Neutral
- All five files remain in `package cli`; there is no package boundary or import-cycle risk introduced.
- The `logs_orchestrator_filters_test.go` file uses `package cli` (not `package cli_test`), so it retains access to package-private identifiers needed to construct test fixtures.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
