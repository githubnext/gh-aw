# ADR-42538: Add sprintfint Linter — Flag fmt.Sprintf("%d", int) in Favor of strconv.Itoa

**Date**: 2026-06-30
**Status**: Draft
**Deciders**: pelikhan, linter-miner automation

---

### Context

Two production code sites in the repository (`pkg/stringutil/stringutil.go:59` and `pkg/console/console_wasm.go:41`) use `fmt.Sprintf("%d", x)` where `x` is a plain `int`. This idiom is correct but suboptimal: it incurs format-string parsing at runtime, requires importing the `fmt` package unnecessarily, and obscures intent compared to `strconv.Itoa(x)`. The project already encodes code-quality preferences as custom Go static-analysis linters in `pkg/linters/` (e.g., `sprintferrdot`, `sprintferrorsnew`), making a new linter the natural enforcement mechanism. Automated enforcement prevents this pattern from re-entering the codebase after the two existing instances are fixed.

### Decision

We will add a new custom Go analysis pass, `sprintfint`, that flags every call of the form `fmt.Sprintf("%d", x)` where `x` has the exact type `int` and reports `use strconv.Itoa(x) instead`. The analyzer includes a `SuggestedFix` that rewrites the call in place, skips `_test.go` files, and respects `//nolint:sprintfint` suppressions. It is registered in the project-wide `cmd/linters` multichecker following the same conventions as existing linters.

### Alternatives Considered

#### Alternative 1: Rely on code review alone

Do not add any automated rule; trust that code authors and reviewers catch `fmt.Sprintf("%d", int)` during review. This was the implicit status quo and did not prevent the two existing instances from entering production code. Without automation the pattern will continue to recur as the codebase grows.

#### Alternative 2: Use an existing external linter (e.g., perfsprint from golangci-lint)

The `perfsprint` linter (available in `golangci-lint`) covers similar patterns. However, it operates outside the project's custom-linter infrastructure, requires an additional external dependency and configuration layer, and its scope is broader than the narrow `fmt.Sprintf("%d", int)` → `strconv.Itoa` substitution the team wants to enforce. Adopting it would also break the convention of keeping all custom lint rules within `pkg/linters/` where they share internal helpers (`astutil`, `filecheck`, `nolint`).

#### Alternative 3: Extend the linter to cover all integer types (int8, int32, int64, uint, etc.)

A broader rule would flag `fmt.Sprintf("%d", n64)` (int64) and suggest `strconv.FormatInt(n64, 10)`, and similarly for other integer widths. This was not chosen because the safe mechanical replacement (`strconv.Itoa`) is only available for the plain `int` type; wider coverage would require type-specific suggestions that complicate the implementation and increase false-positive risk for less common types. The narrow scope can be expanded in a follow-up linter if needed.

### Consequences

#### Positive
- Eliminates a class of redundant format-string calls in production code, reducing minor runtime overhead and improving readability.
- Provides a `SuggestedFix` so editors and `go tool` can auto-apply the rewrite, lowering friction for contributors.
- Consistent with the project's established pattern of encoding style decisions as independently testable `golang.org/x/tools/go/analysis` passes.
- Prevents regression: the two existing instances and any future occurrences will be caught in CI.

#### Negative
- Scope is restricted to the exact type `int`; the same smell with `int64`, `uint`, `int32`, etc. remains undetected and must be addressed separately.
- Adds another analyzer to the multichecker that must be maintained as Go's standard library and `x/tools/go/analysis` API evolve.

#### Neutral
- Test fixtures follow the project's `analysistest` convention (`testdata/src/<pkg>/<pkg>.go` with `// want` annotations), adding a small amount of checked-in test data.
- The `//nolint:sprintfint` suppression mechanism follows the same pattern as other linters in the suite, so team norms around nolint comments carry over without new conventions.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
