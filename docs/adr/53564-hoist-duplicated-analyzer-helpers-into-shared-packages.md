# ADR-53564: Hoist Duplicated Analyzer Helpers into Shared Packages

**Date**: 2026-08-18
**Status**: Accepted
**Deciders**: pelikhan (PR author), copilot-swe-agent

---

### Context

The `pkg/linters` directory houses 63 custom Go static-analysis analyzers. A semantic clustering analysis revealed 8 clusters of helper logic that were independently re-implemented in each analyzer package, despite `internal/astutil` and `internal/analyzerutil` already existing for shared analyzer utilities. One of these duplicates had silently diverged: `regexpcompileinfunction` only matched `Compile`/`MustCompile`, while `regexpdynamicpattern` matched four names including the POSIX variants, leaving the package-level-hoisting check silently incomplete for POSIX regexp calls. Without a shared implementation, each new analyzer copy risks its own independent drift.

### Decision

We will extract the 8 clusters of duplicated logic into new exported functions in `internal/astutil` (`IsRegexpCompileCall`, `HasConstantStringArg`, `UniverseErrorInterface`, `StringLitValue`, `IsInInitFunction`, `NormalizeComparisonOperands`, `SwapPkgImportEdits`) and one new function in `internal/analyzerutil` (`Indexes`). Each of the 63 affected analyzer `run` functions will be updated to call these shared helpers. Caller-specific logic that genuinely differs between callers (e.g., the orphan-`fmt` determination in `SwapPkgImportEdits`) is left at each call site.

### Alternatives Considered

#### Alternative 1: Code Generation

Use `go generate` to stamp identical boilerplate into each analyzer package from a shared template, keeping per-package copies while enforcing consistency via CI. This avoids a shared runtime dependency but does not fix the divergence problem — generated code can still drift if templates are edited inconsistently — and adds tooling complexity (template maintenance, generation step in CI). The silent POSIX regexp gap would persist until the template was updated, rather than being fixed structurally.

#### Alternative 2: New Dedicated `linterutil` Package

Extract helpers into a new top-level `pkg/linters/internal/linterutil` package rather than extending `astutil` and `analyzerutil`. This isolates the new helpers but fragments the shared utility landscape: `astutil`, `analyzerutil`, and `linterutil` would all be separate places to look for analyzer helpers. The existing packages were created for exactly this purpose, so adding a third package without a clear boundary would confuse future contributors.

### Consequences

#### Positive
- Eliminates 8 independent re-implementations across 63 analyzer `run` functions, reducing the chance of future silent divergence.
- Fixes the behavioral gap in `regexpcompileinfunction` (POSIX variants `CompilePOSIX`/`MustCompilePOSIX` are now covered), matching the coverage already present in `regexpdynamicpattern`.
- Future improvements to shared helpers (e.g., adding a new regexp variant) take effect across all analyzers automatically with a single change.

#### Negative
- All 63 analyzers now depend on the same shared helper implementations — a behavioral change to `NormalizeComparisonOperands` or `IsRegexpCompileCall` affects every caller simultaneously, increasing the blast radius of future edits.
- `SwapPkgImportEdits` introduces a new convention: caller-specific logic that truly differs between callers stays at the call site rather than being absorbed into the helper, which contributors must understand to avoid over-extracting in the future.

#### Neutral
- `analyzerutil.Indexes` is semantically guaranteed by the `Requires` list already declared in `New`/`NewAtPath`; the refactor consolidates retrieval boilerplate without changing the data availability contract.
- All existing `pkg/linters` testdata suites pass unchanged; the behavioral changes to `regexpcompileinfunction` and `NormalizeComparisonOperands` are tested with new testdata added in this PR.
