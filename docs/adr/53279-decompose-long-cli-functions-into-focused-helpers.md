# ADR-53279: Decompose Long CLI Functions into Focused Helpers

**Date**: 2026-08-17
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `pkg/cli/` package contained several functions that exceeded the project-enforced 60-line limit checked by `make golint-custom` (largefunc). The worst offenders — `downloadRunArtifacts` and `buildLogsData` — each reached 321 lines. These monolithic functions combined artifact enumeration, cache-resolution, incremental planning, bulk/individual download dispatch, error recovery, and finalization in a single scope, making individual steps impossible to test in isolation and difficult to read at a glance.

### Decision

We will decompose each oversized function into focused, single-responsibility helpers named after the conceptual step they perform (enumerate, plan, execute, recover, finalize). `downloadRunArtifacts` is now an orchestrating pipeline that delegates to `resolveCachedArtifacts`, `planArtifactDownload`, `bulkDownloadArtifacts` / `downloadArtifactsIndividually`, and `finalizeArtifactDownload`. `buildLogsData` delegates accumulation to a `logsAggregate` struct with dedicated methods (`accumulateRunTotals`, `accumulateChainMetrics`, `classifyRunFailure`, `summary`). All new helpers are package-private; the external API and error semantics are unchanged.

### Alternatives Considered

#### Alternative 1: Raise or disable the lint limit

The `largefunc` rule could be configured with a higher threshold or disabled entirely. This would silence the lint failure without changing any code, preserving the monolithic structure. This was rejected because the 60-line limit exists to enforce readability and testability; raising it would erode the policy without solving the underlying problem.

#### Alternative 2: Introduce method receivers on a shared struct

Related operations could be grouped by moving them onto method receivers of a new struct (e.g., `artifactDownloader`). This would co-locate state and make the relationship between helpers explicit through the type system. It was rejected for this refactor because the functions are essentially a procedural pipeline with no persistent state between calls; adding a struct would add abstraction overhead without meaningful benefit over named free functions in the same package.

### Consequences

#### Positive
- Each extracted helper can be unit-tested in isolation, reducing the need to set up the full download pipeline to test a single decision (e.g., `planIncrementalDownload`).
- The top-level orchestrating functions now read as a sequential pipeline of named steps, making it easy to understand the high-level flow without reading implementation details.
- Repeated patterns (e.g., the "download logs for diagnostics then clean up empty dir" block) are de-duplicated into a single `downloadWorkflowRunLogsForDiagnostics` helper.

#### Negative
- Readers must navigate more functions to trace the full execution path; a call graph spans more files than before.
- Some state that was naturally scoped within the monolith (e.g., the spinner lifecycle, the `skippedNonZipArtifacts` flag) must now be explicitly threaded through parameters or return values.

#### Neutral
- The public API surface is unchanged; all new helpers are unexported (`package cli`).
- Output shape and error semantics are preserved exactly — this is a behavior-preserving refactor.
- The `buildSessionAnalysis` function (77 lines, same file as `buildMCPServerHealth`) was intentionally left out of scope for this PR.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
