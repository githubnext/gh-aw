# ADR-44651: Split compiler_yaml_main_job.go into Phase-Focused Modules

**Date**: 2026-07-10
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`pkg/workflow/compiler_yaml_main_job.go` had grown to 1265 lines spanning 23 methods across 5 distinct compilation phases: checkout, runtime setup, AI execution, post-agent cleanup, and orchestration. The single-file layout made targeted edits expensive because readers had to mentally filter across phases to locate a specific method, and navigating to a phase-specific bug required scrolling past unrelated code. The file had become a maintenance liability as the compiler's feature set expanded.

### Decision

We will split the monolithic `compiler_yaml_main_job.go` into four phase-aligned files (`compiler_yaml_checkout.go`, `compiler_yaml_runtime_setup.go`, `compiler_yaml_ai_execution.go`, `compiler_yaml_post_agent.go`) within the same `package workflow`, reducing the orchestrator to a ~42-line phase dispatcher. No logic is changed and no imports are rewired — the split is purely mechanical to improve code navigability and reduce per-file cognitive load.

### Alternatives Considered

#### Alternative 1: Add region comments and IDE navigation hints within the single file

Add structured comments (e.g. `// --- Phase 1: Checkout ---`) and rely on IDE symbol navigation to improve discoverability within the existing file. This avoids any file system change and keeps the full compilation flow visible in one scroll. It was rejected because it doesn't reduce file size or parallel-edit conflicts, and region comments are not a Go convention — they would require team discipline to maintain consistently.

#### Alternative 2: Extract into sub-packages (e.g. `workflow/phases/checkout`)

Move each phase into its own sub-package to enforce encapsulation at the language level. This would grant each phase a private namespace and prevent accidental cross-phase coupling. It was rejected because the methods share many unexported helpers, types, and the `Compiler` receiver from `package workflow`; extracting them would require either a large interface refactor or circular imports, making this a significantly larger and riskier change than a pure file split.

### Consequences

#### Positive
- Each phase now lives in its own file, matching file name to responsibility and reducing per-file line count from ~1265 to ~250–500 lines.
- Parallel edits to different phases no longer require touching the same file, reducing merge conflicts in active development.
- Onboarding is faster: a developer investigating checkout behaviour can open `compiler_yaml_checkout.go` directly.

#### Negative
- The full end-to-end compilation flow is no longer readable in a single file; understanding the sequence now requires reading `compiler_yaml_main_job.go` plus the four phase files.
- All files remain in `package workflow`, so no encapsulation boundary is enforced — cross-phase coupling can still accumulate silently.

#### Neutral
- The `Compiler` method set grows across files, which is idiomatic Go but can be unexpected for developers coming from languages where a type's methods must be co-located.
- The `compiler_yaml_ai_execution.go` file absorbs two phases (3 and 4) and remains ~497 lines; a future split may be warranted if it grows further.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
