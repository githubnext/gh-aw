# ADR-42323: Add errortypeassertion Custom Go Analyzer

**Date**: 2026-06-29
**Status**: Draft
**Deciders**: Unknown

---

### Context

Go 1.13 introduced error wrapping via `fmt.Errorf("...: %w", err)` and the `errors.As` / `errors.Is` traversal API. However, callsites that use direct type assertions — `err.(*os.PathError)` — silently fail when the error value is wrapped, because a type assertion checks the concrete dynamic type of the outermost value and does not unwrap the chain. This creates correctness bugs that are easy to miss in code review: the assertion compiles and runs without panic, but always produces a zero value or `ok == false` for wrapped errors. The `gh-aw` codebase already has a suite of custom `go/analysis` analyzers in `pkg/linters/` that enforce similar error-handling patterns (e.g., `errorfwrapv`, `errstringmatch`), and adding enforcement here follows the same pattern of static analysis at build time rather than relying solely on reviewer attention.

### Decision

We will add a new custom `go/analysis` analyzer, `errortypeassertion`, that flags any `TypeAssertExpr` where the asserted-from expression has the built-in `error` type and the asserted-to type is a concrete (non-interface) type, and emits a diagnostic recommending `errors.As`. Interface assertions (e.g., `err.(interface{ Timeout() bool })`) and type-switch guards are intentionally excluded because they represent valid behavior checks, not wrapped-error traversal. The analyzer is registered in `cmd/linters/main.go` and in the spec test, consistent with all other analyzers in this suite.

### Alternatives Considered

#### Alternative 1: Code Review and Documentation Only

Rely on PR reviewers and developer documentation to catch direct error type assertions. This costs nothing to implement but provides no automated enforcement, so violations persist whenever reviewers miss them. In a large codebase with many contributors, manual review is insufficient for systematic enforcement of a subtle correctness invariant.

#### Alternative 2: Use an Existing Third-Party Linter (e.g., `errorlint --errorlint-assertion`)

The `errorlint` linter from `golangci-lint` has an `--errorlint-assertion` flag that flags exactly this pattern. This avoids building and maintaining a custom analyzer. However, it introduces an external dependency outside the existing custom analyzer framework, would not integrate with the project's internal `nolint`, `filecheck`, and `astutil` helpers, and may flag patterns the team intentionally wants to allow — requiring either upstream configuration or wrapper logic that approaches the complexity of a custom analyzer.

### Consequences

#### Positive
- Correctness bugs caused by direct error type assertions bypassing wrapped error chains are caught at static analysis time, before runtime.
- The new analyzer reuses all existing internal infrastructure (`astutil.Inspector`, `nolint.BuildLineIndex`, `filecheck.IsTestFile`), keeping enforcement uniform and the implementation small (72 lines).
- Developers receive an actionable diagnostic pointing them to `errors.As`, reducing the learning curve for the error wrapping pattern.

#### Negative
- Adds a custom analyzer that must be maintained alongside the internal helper packages; if the shared helpers change their API, this analyzer must be updated.
- Any new legitimate error assertion pattern not covered by the current exclusion rules (e.g., future patterns that are not interface assertions) would produce false positives until the analyzer is updated.

#### Neutral
- The analyzer is suppressed by `//nolint:errortypeassertion` for cases where the caller knowingly uses direct assertion (e.g., in code that cannot use `errors.As` due to interface constraints). This matches the nolint convention used by all other analyzers in the suite.
- Test files are excluded from analysis by the `filecheck.IsTestFile` helper, consistent with the suite's test-exclusion policy.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
