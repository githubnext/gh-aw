# ADR-45700: Share nolint and generated-file indexes across custom Go analyzers

**Date**: 2026-07-15
**Status**: Accepted
**Deciders**: Unknown (copilot-swe-agent, pelikhan)

---

### Context

The custom Go linter suite in `pkg/linters/` contained duplicated logic in every analyzer: each independently scanned all AST comments to build a per-run nolint directive index, and each independently checked `IsTestFile` to skip test files. Generated Go source files were not skipped at all, making them eligible for lint diagnostics. With ~20+ analyzers each performing the same comment scan per package, the analysis framework was repeating this work N times per package.

### Decision

We will extract the nolint-directive scanning and generated-file detection into two dedicated prerequisite analyzers (`nolintindex` in `pkg/linters/internal/nolint` and `generatedfileindex` in `pkg/linters/internal/filecheck`). Every custom analyzer declares both as `Requires` dependencies and fetches their results via `pass.ResultOf`, so the index is built exactly once per package. We will replace per-analyzer `BuildLineIndex(pass, linterName)` calls with `nolint.Index(pass)` and replace `IsTestFile` guards with `ShouldSkipFilename(filename, generatedFiles)`, which skips both test files and files marked with the standard Go generated-file marker.

### Alternatives Considered

#### Alternative 1: Keep per-analyzer comment scanning, add generated-file detection to each

Each analyzer continues to call `BuildLineIndex` locally but also calls a new per-analyzer `BuildGeneratedIndex`. This eliminates the generated-file gap without architectural change. Rejected because it multiplies the redundant comment and AST comment work proportionally to the number of analyzers and misses the opportunity to fix both problems with a single design change.

#### Alternative 2: Package-level caching with `sync.Once` or a global map

A shared utility function could cache results per `*analysis.Pass` using a global keyed map or `sync.Once`. This avoids changing the `Requires` dependency list. Rejected because it would be incompatible with how the Go analysis framework parallelises passes and could introduce data races; the `pass.ResultOf` prerequisite mechanism is the framework's canonical way to share computed data between analyzers in the same pass.

### Consequences

#### Positive
- Eliminates N-fold redundant comment scanning: each package's nolint index and generated-file index is built exactly once regardless of how many analyzers run against it.
- Makes file-skipping behavior consistent across all linters: generated files (detected via `ast.IsGenerated`) are now uniformly excluded in addition to test files.
- Unit tests for the shared index helpers (`BuildDirectiveIndex`, `BuildGeneratedIndex`, `ShouldSkipFilename`) provide regression coverage that was previously absent.

#### Negative
- Each custom analyzer's `Requires` slice now includes two additional entries (`nolint.Analyzer`, `filecheck.Analyzer`), slightly increasing per-analyzer dependency declaration verbosity and making the dependency graph wider.
- The `DirectiveIndex` now stores all linter names across all `nolint:` directives in a package rather than only those for the requesting linter; this trades a wider map at index-build time for elimination of repeated scans.

#### Neutral
- `HasDirective` and `BuildLineIndex` are preserved for backward compatibility but marked deprecated in favor of shared-index APIs (`HasDirectiveForLinter` and `Index`).
- The `nolintindex` and `generatedfileindex` analyzers set `RunDespiteErrors: true`, so analysis proceeds even when the package has type errors.

---

*ADR created by [adr-writer agent] and finalized in this PR.*
