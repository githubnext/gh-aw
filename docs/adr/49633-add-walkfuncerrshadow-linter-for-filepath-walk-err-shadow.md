# ADR-49633: Add walkfuncerrshadow Linter for filepath.Walk/WalkDir err Parameter Shadowing

**Date**: 2026-08-01
**Status**: Accepted
**Deciders**: Unknown

---

### Context

The `filepath.Walk` and `filepath.WalkDir` APIs take a callback whose third parameter is conventionally named `err`. When the outer variable receiving the walk's return value is also named `err`, the callback parameter silently shadows it. This is a narrow but recurring source of confusion during refactors — the inner `err` and the outer `err` are distinct variables, but their identical names make it easy to misread call sites and introduce bugs. The project already has a collection of custom `go/analysis` analyzers in `pkg/linters/` that target specific, high-signal patterns; this is another instance of the same category.

### Decision

We will add a new focused `go/analysis` analyzer (`walkfuncerrshadow`) in `pkg/linters/walkfuncerrshadow/` that flags exactly the case where a `filepath.Walk` or `filepath.WalkDir` call assigns its result to an outer variable named `err` and the callback's third parameter is also named `err`. The analyzer will be registered in the existing linter registry (`pkg/linters/registry.go`) so it runs through the project's standard linter driver. It will be narrowly scoped: it does not flag distinct outer names (e.g. `walkErr`), distinct callback parameter names, or non-`filepath` walkers.

### Alternatives Considered

#### Alternative 1: Rely on a Generic Shadow Checker (e.g., govet's `shadow` analyzer)

Generic shadow analyzers like `go vet -shadow` or golangci-lint's `govet` shadow pass would catch this pattern but also flag every other variable shadowing in the codebase. This produces a high volume of diagnostics that are not all bugs, making it impractical to enable globally. The decision was to write a focused analyzer to achieve zero false positives for the common non-problematic cases (distinct names, non-`filepath` walkers).

#### Alternative 2: Lint via a golangci-lint Plugin Wrapping an Existing Analyzer

The project could integrate a third-party golangci-lint plugin for shadow detection. This adds an external dependency that must be vendored and kept compatible, and the detection scope is still broader than needed. The in-tree custom analyzer approach avoids new external dependencies and keeps the detection logic within the project's standard analysis framework, consistent with all other custom analyzers in `pkg/linters/`.

### Consequences

#### Positive
- Provides precise, zero-false-positive detection of the specific `filepath.Walk`/`WalkDir` err-shadowing pattern without noise from unrelated variable shadowing.
- Reduces the risk of a class of subtle bugs introduced during refactors, where developers misread `err` in the callback as the outer walk-result `err`.
- Integrates cleanly into the existing linter driver and registry with minimal surface area.
- Provides an escape hatch (`//nolint:walkfuncerrshadow`) for intentional cases.

#### Negative
- Any existing code that uses the flagged pattern must be reviewed and potentially renamed; this is a one-time migration cost.
- The analyzer is intentionally narrow — it will not catch analogous patterns with `fs.WalkDir` or other walk-like APIs, so developers might not expect the asymmetry.

#### Neutral
- Increases the active analyzer count from 60 to 61; documentation and spec tests are updated accordingly.
- The analyzer follows the exact same structural pattern as all other analyzers in `pkg/linters/`, requiring no new conventions.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
