# ADR-55891: Adopt testify/assert and Exhaustive Boundary Tests for pkg/timeutil Format Functions

**Date**: 2026-08-26
**Status**: Draft
**Deciders**: Unknown

---

### Context

`pkg/timeutil/format.go` exposes three duration-formatting functions: `FormatDuration`, `FormatDurationMs`, and `FormatDurationNs`. The existing `format_test.go` only covered `FormatDuration` and used raw `if`/`t.Errorf` assertions, inconsistent with the `testify/assert` style adopted elsewhere in the repository. `FormatDurationMs` and `FormatDurationNs` — both non-trivial due to rounding logic, zero/negative guards, and multi-unit composition — were tested only by the documentation-driven `spec_test.go`, which maps cases to README spec sections rather than exercising boundary conditions. This left critical paths (rounding at half-second, negative-input behavior, minute/hour composition) without direct, exhaustive coverage.

### Decision

We will add table-driven tests for `FormatDurationMs` and `FormatDurationNs` directly in `format_test.go`, using `testify/assert` for all assertions (including migrating the existing `TestFormatDuration` from `if`/`t.Errorf`). The `spec_test.go` file will remain untouched because it serves a distinct, documentation-traceability purpose and lives in a separate package (`timeutil_test`). This keeps exhaustive boundary coverage and documentation-driven contract tests as two complementary, intentionally separate layers.

### Alternatives Considered

#### Alternative 1: Keep Standard `if`/`t.Errorf` Assertions

The existing `TestFormatDuration` used Go's built-in `testing` package for assertions. Keeping this style avoids the `testify` import in this file. However, it produces less informative failure messages and diverges from the assertion style already adopted in the wider repository, increasing cognitive overhead when switching between test files.

#### Alternative 2: Consolidate All Tests Into `spec_test.go`

Adding `FormatDurationMs` and `FormatDurationNs` boundary cases to `spec_test.go` would centralize all tests in one file. However, `spec_test.go` is intentionally documentation-driven — each case cites a README spec section — and adding exhaustive boundary rows (e.g., `499_999_999 ns`, `500_000_000 ns`) would conflate two different testing goals. It would also require bringing `testify` into the `timeutil_test` external package, changing its character.

### Consequences

#### Positive
- `FormatDurationMs` and `FormatDurationNs` now have explicit coverage for boundary conditions: zero/negative guard, rounding at ±half-unit, and multi-unit composition.
- `testify/assert` provides richer failure messages (`expected X, got Y`) and a consistent style with the rest of the test codebase.

#### Negative
- `format_test.go` and `spec_test.go` now use different assertion libraries (`testify/assert` vs standard `testing`), which could cause confusion for contributors new to the package.
- Negative-input behavior for `FormatDurationMs` (e.g., `-500` → `"-500ms"`, not `"—"`) is asserted as-is without a guard, leaving a known behavioral inconsistency between `FormatDurationMs` and `FormatDurationNs` undocumented in production code.

#### Neutral
- The `testify` package is already a project-level test dependency; adding it to this specific file does not introduce a new external dependency.
- `spec_test.go` is preserved with no changes, maintaining its role as a documentation-contract layer.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
