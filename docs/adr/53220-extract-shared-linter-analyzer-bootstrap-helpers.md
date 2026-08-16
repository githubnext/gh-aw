# ADR-53220: Extract Shared Linter Analyzer Bootstrap into analyzerutil Helpers

**Date**: 2026-08-16
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Every analyzer in `pkg/linters` opened its `run` function with an identical ~20-line bootstrap: call `nolint.Index(pass)`, call `filecheck.Index(pass)`, declare a `nodeFilter`, and wire a thin `analyzerutil.Preorder` closure forwarding to a package-specific analyze function. This pattern was duplicated across 60+ linter packages (see issue #53217). Any policy change to that setup — error handling, skip behavior, index construction — had to be replicated manually, creating drift risk and maintenance burden at scale.

### Decision

We will introduce two new helpers in `pkg/linters/internal/analyzerutil`:

- `Indexes(pass)` — builds both the `filecheck.GeneratedIndex` and the `nolint.DirectiveIndex` in a single call, returning `(GeneratedIndex, DirectiveIndex, error)`.
- `PreorderIndexed(pass, nodeFilter, visit)` — combines index construction with a preorder AST traversal, invoking `visit(pass, node, generatedFiles, noLintIndex)`. Because this signature matches most linters' existing analyze functions, those functions can be passed by reference rather than wrapped in a closure.

Linters that require extra state, package-path gating, or inline visitor bodies will continue to call `Indexes` and wire their own `Preorder` closure. This keeps the helpers composable rather than monolithic.

### Alternatives Considered

#### Alternative 1: Keep duplication as-is (do nothing)

Each package continues to own its full bootstrap verbatim. No new abstraction is needed; the code is already written and tested. Rejected because the duplication is already causing drift (policy changes must touch 60+ files) and the pattern is stable enough to warrant extraction.

#### Alternative 2: Options-struct or middleware pattern

Rather than `PreorderIndexed`, expose a builder or functional-options API that lets callers opt into index construction piece by piece. More flexible, but significantly more complex and harder to discover. The existing `Preorder` + `Indexes` split already provides the necessary escape hatch for non-standard linters, making a richer options API unnecessary overhead.

### Consequences

#### Positive
- Single point of change for bootstrap policy: adding a new skip condition, changing error propagation, or adding a third index only requires editing two functions instead of 60+.
- `run` functions that previously contained 20 lines of setup collapse to a single `PreorderIndexed` call, making the package-specific analysis logic immediately visible.
- Unit tests for `Indexes` and `PreorderIndexed` cover error propagation centrally, reducing the need for per-linter error-path tests.

#### Negative
- All linters that use `PreorderIndexed` now share the same visit-function signature `(pass, node, generatedFiles, noLintIndex)`. Future changes that require a different signature (e.g., adding a third index) must update both the helper and all callers simultaneously.
- The new `analyzerutil` helpers add a package-level dependency for all linters; removing or renaming them is now a large-scale refactor.

#### Neutral
- The 17 linters that adopt `PreorderIndexed` have their `run` bodies effectively opaque — the analysis logic lives in the named analyze function, not in the run body itself. This is consistent with the existing convention but may surprise readers expecting to see the logic inline.
- Linters with extra state or gating continue to use the explicit `Indexes` + `Preorder` pattern, so two idioms coexist in the package fleet.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
