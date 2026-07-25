# ADR-47912: Consolidate Import-Editing and File-Lookup Helpers into internal/astutil

**Date**: 2026-07-25
**Status**: Draft
**Deciders**: Unknown

---

### Context

Five import-manipulation helpers (`fileForPos`, `countPkgUsesInFile`, `addImportEdit`, `removeImportEdit`, `importSpecLineRange`) were copy-pasted nearly verbatim between `sprintfbool` and `sprintfint`, and the file-containing-position lookup pattern was re-implemented inline in at least five packages (`sprintfbool`, `sprintfint`, `writebytestring`, `bytescomparestring`, `uncheckedtypeassertion`). A latent drift was already visible: `addStrconvRemoveFmtEdits` in the two packages produced TextEdit slices in different order, and `sprintfint` used raw string comparison while `sprintfbool` used a helper — a behavioral gap that would silently grow over time. The shared helper layer `pkg/linters/internal/astutil` already owned `Inspector`, `ImportedAs`, and related AST utilities and was the natural consolidation point.

### Decision

We will promote the duplicated cluster — `FileForPos`, `CountPkgUsesInFile`, `ImportSpecLineRange`, `AddImportEdit`, `RemoveImportEdit`, and a generalized `SwapImportEdits` — into `internal/astutil` as exported, unit-tested functions, and update all call sites in `sprintfbool`, `sprintfint`, `writebytestring`, `bytescomparestring`, and `uncheckedtypeassertion` to use the shared versions. The generalized swap helper replaces the two bespoke `addStrconvRemoveFmtEdits` functions with a parameter-driven implementation, eliminating the ordering drift.

### Alternatives Considered

#### Alternative 1: Keep helpers local to each linter package (status quo)

Each linter independently owns its import-editing logic. No new shared surface area is introduced. Rejected because the near-verbatim duplication across five packages had already produced subtle behavioral drift (`addStrconvRemoveFmtEdits` edit ordering) and would continue to diverge silently as linters evolve independently.

#### Alternative 2: Extract into a dedicated internal/importutil package

Create a new package `pkg/linters/internal/importutil` rather than growing `internal/astutil`. Rejected because the helpers operate on the same `*ast.File`, `token.FileSet`, and `analysis.Pass` types that `astutil` already handles, and adding a second internal package would fragment the shared layer without a clear boundary benefit at this scale.

### Consequences

#### Positive
- Removes approximately 360 lines of duplicated code across five packages, shrinking per-linter maintenance surface.
- Eliminates the latent `addStrconvRemoveFmtEdits` ordering drift by replacing both copies with a single generic `SwapImportEdits` implementation.
- Adds unit tests for `FileForPos`, `CountPkgUsesInFile`, and `ImportSpecLineRange` in `astutil_test.go`, covering the shared logic for all consumers.
- Import-presence checks in `sprintfbool` switch to `astutil.ImportedAs`, which handles backtick-quoted paths correctly — a correctness improvement over the local `importSpecPathEquals`.

#### Negative
- `internal/astutil` grows in scope: it now owns import-editing concerns in addition to AST traversal and type predicates, making the package boundary less crisp.
- `bytescomparestring`'s `buildBytesImportTextEdit` was intentionally **not** migrated to `AddImportEdit` because it inserts the new import first (rather than appending), which differs from the shared helper's trailing-append behavior. This leaves a documented behavioural inconsistency between `bytescomparestring` and the other linters.

#### Neutral
- All existing linter golden-file tests continue to pass unchanged; the refactor is behaviour-preserving at the external test boundary.
- Future linters that need import-editing utilities now have a discoverable, tested API rather than needing to copy-paste from an existing linter.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
