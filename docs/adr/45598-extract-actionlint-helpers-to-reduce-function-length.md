# ADR-45598: Extract actionlint helper functions to reduce large-function lint findings

**Date**: 2026-07-15
**Status**: Draft
**Deciders**: Unknown

---

### Context

`pkg/cli/actionlint.go` contained two functions — `runActionlintOnFilesWithOptions` and `parseAndDisplayActionlintOutput` — that exceeded the project's `largefunc` lint threshold. Each function mixed several distinct responsibilities: version printing, path resolution, Docker argument construction, command execution, timeout/cancellation handling, output parsing, error-type mapping, and snippet extraction. This made them difficult to read at a glance and impossible to test individual sub-behaviours in isolation. The PR targets the explicit `largefunc` backlog in `pkg/cli`.

### Decision

We decided to decompose `runActionlintOnFilesWithOptions` and `parseAndDisplayActionlintOutput` into a set of focused, single-responsibility helper functions (`maybePrintActionlintVersion`, `resolveActionlintPaths`, `runActionlintCommand`, `buildActionlintDockerArgs`, `printActionlintRunMessage`, `actionlintContextError`, `handleActionlintExecutionError`, `handleActionlintFindings`, `actionlintFileDescription`, `buildActionlintCompilerError`, `actionlintErrorType`, `actionlintErrorMessage`, `actionlintErrorContext`), while retaining the original orchestration entry points and leaving the actionlint-facing API and output model unchanged.

### Alternatives Considered

#### Alternative 1: Suppress `largefunc` lint findings with ignore comments

Add per-function lint-suppress annotations (e.g., `//nolint:cyclop` or project-specific inline directives) to silence the findings without restructuring the code. This avoids refactoring cost entirely. Rejected because suppression does nothing to address the underlying complexity — the functions remain hard to read and impossible to unit-test at the sub-behaviour level, and the backlog of `largefunc` findings would continue to grow.

#### Alternative 2: Introduce a struct-based runner (`actionlintRunner`)

Encapsulate shared state (git root, relative paths, run options, command result) in an `actionlintRunner` struct with methods for each logical step, following a builder/runner pattern. This would provide natural grouping via a named type and clean method dispatch. Rejected for this change because it requires a larger structural refactor (new exported or unexported type, constructor, method set) that goes beyond the scope of the targeted lint cleanup; it can be revisited if further state accumulates around the runner in the future.

### Consequences

#### Positive
- Each extracted helper is independently unit-testable; the PR adds `TestBuildActionlintDockerArgs` and `TestBuildActionlintCompilerError` to demonstrate this directly.
- The orchestrating function `runActionlintOnFilesWithOptions` now reads as a high-level sequence of named steps, making the execution flow immediately apparent.
- The `largefunc` lint findings for these two functions are resolved, reducing the overall backlog in `pkg/cli`.

#### Negative
- The package's internal namespace gains roughly a dozen new unexported functions, increasing the number of symbols a reader must hold in mind when navigating the file.
- The new `actionlintCommandResult` struct is an additional type to track; its fields must be kept consistent with what `runActionlintCommand` populates and what `actionlintContextError`/`handleActionlintExecutionError` consume.

#### Neutral
- The public API (`runActionlintOnFiles` signature and all caller-visible behaviour) is unchanged; no callers require modification.
- Test coverage for the existing `parseAndDisplayActionlintOutput` orchestration path is not added in this PR; the new tests only cover the extracted sub-behaviours.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
