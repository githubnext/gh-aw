# ADR-45635: Decompose `downloadRunArtifactsConcurrent` into Focused Helper Functions

**Date**: 2026-07-15
**Status**: Draft
**Deciders**: Unknown (automated refactor by Copilot SWE agent, pelikhan)

---

### Context

The CLI logs/download cluster enforces a `largefunc` linter rule capping functions at 60 lines. `downloadRunArtifactsConcurrent` in `pkg/cli/logs_run_processor.go` had grown to 430 lines, inlining the full per-run pipeline — context cancellation check, cache-hit path, artifact download, metrics extraction, security analysis, behavioral-signal extraction, usage-metrics collection, agentic analysis derivation, and `RunSummary` persistence — inside a single goroutine closure. This made the function the last remaining `largefunc` violation in the cluster and made each stage hard to locate, test in isolation, or reason about independently.

### Decision

We will decompose `downloadRunArtifactsConcurrent` into 14 named helper functions, each at or under 60 lines, within the same file and package (`pkg/cli`). The orchestration function is reduced to pure pool setup and result collection; per-run work is delegated to `processSingleRunDownload`, which in turn calls focused helpers (`tryLoadCachedRunResult`, `analyzeRunArtifacts`, `extractRunMetricsAndMetadata`, `applyRunSecurityAnalysis`, `applyRunBehavioralSignals`, `applyRunUsageMetrics`, `finalizeAndSaveRunSummary`, and small utilities). Shared goroutine parameters are bundled into a `concurrentRunDownloadParams` struct to avoid repetitive argument passing. Public interfaces and runtime behavior are unchanged.

### Alternatives Considered

#### Alternative 1: Suppress the linter for this function

Add a `//nolint:cyclop` or linter-exception comment to keep `downloadRunArtifactsConcurrent` as a single function. This avoids any structural change and removes the build noise with minimal effort.

Not chosen because it treats the symptom (lint failure) rather than the root cause (large function that mixes coordination and detail). Future contributors would inherit 430 lines of interleaved concerns with no linter guidance to keep it bounded.

#### Alternative 2: Extract the pipeline into a dedicated struct or sub-package

Move the download pipeline into a `runDownloadPipeline` struct with methods, or into a new `internal/rundownload` sub-package, giving each stage a well-typed receiver and hiding helpers behind a package boundary.

Not chosen because the per-run pipeline is tightly coupled to types and unexported helpers already in `pkg/cli` (e.g. `loadRunSummary`, `backfillCacheHitIfNeeded`, `deriveRunAgenticAnalysis`). A struct or sub-package refactor would require either exporting those helpers (expanding the API surface) or co-locating them in the new package (a larger change). The PR scope was limited to function-length compliance with no behavioral change.

### Consequences

#### Positive
- All 14 resulting functions are at or under the 60-line limit, clearing the `largefunc` backlog for this cluster.
- Each pipeline stage (cache lookup, security analysis, behavioral signals, usage metrics, summary persistence) is now a named function, making the code self-documenting and easier to navigate.
- The `concurrentRunDownloadParams` struct eliminates repetitive argument passing through the goroutine closure, reducing the chance of positional errors when signatures change.

#### Negative
- Readers who want to understand the full per-run pipeline must follow a call chain through 9 functions rather than reading one long function top-to-bottom; this trades vertical length for horizontal indirection.
- The helper functions are package-private but not enforced at a package boundary — future additions to the pipeline are not structurally guided toward any particular function, so the decomposition can decay over time without discipline.

#### Neutral
- The `run := run` loop-variable copy idiom was removed as Go 1.22+ `copyloopvar` semantics make it redundant; this is a correctness-neutral cleanup bundled into the same PR.
- The `concurrentRunDownloadParams` struct is a new named type in the package that future maintainers must be aware of when adding download parameters.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
