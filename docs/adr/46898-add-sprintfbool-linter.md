# ADR-46898: Add sprintfbool Linter to Flag fmt.Sprintf("%t", b) → strconv.FormatBool(b)

**Date**: 2026-07-20
**Status**: Draft
**Deciders**: Unknown

---

### Context

The repository maintains a suite of custom Go static analysis linters under `pkg/linters/`. A pre-existing `sprintfint` linter already flags `fmt.Sprintf("%d", n)` and recommends `strconv.Itoa(n)` for integer-to-string conversions, because `strconv` avoids the runtime reflection overhead of the `fmt` package. The same performance and clarity argument applies to boolean-to-string conversions: `fmt.Sprintf("%t", b)` uses reflection to format a `bool`, while `strconv.FormatBool(b)` is a direct, zero-allocation call. No existing linter in the suite catches this pattern, leaving a gap in consistency enforcement across the codebase.

### Decision

We will add a new `sprintfbool` analysis pass (`pkg/linters/sprintfbool/`) that reports `fmt.Sprintf("%t", b)` calls where `b` has the exact predeclared `bool` type, and provides a suggested fix that rewrites the call to `strconv.FormatBool(b)` while managing `fmt`/`strconv` import additions and removals automatically. The linter will be registered in `cmd/linters/main.go` alongside existing passes, will skip generated files, and will honour `//nolint:sprintfbool` suppression directives.

### Alternatives Considered

#### Alternative 1: Extend sprintfint to Cover Both Integer and Bool Patterns

The `sprintfint` linter could be generalised into a broader "prefer strconv over fmt.Sprintf" pass covering both `%d`/int and `%t`/bool (and potentially other format verbs). This would reduce the number of top-level packages. It was not chosen because combining unrelated verb/type pairs in a single analyzer increases code complexity, makes the diagnostic messages harder to scope precisely, and conflicts with the single-responsibility design pattern already established by the existing linter suite (each linter addresses one specific pattern).

#### Alternative 2: Rely on an External Linter (e.g., staticcheck S1039 or perfsprint)

Third-party linters such as `staticcheck` or `perfsprint` can already flag some `fmt.Sprintf` patterns. Delegating to an external tool would avoid adding a new in-house package. This was not chosen because the repository uses an internal linter framework (`pkg/linters/internal/astutil`, `nolint`, `filecheck`) that provides repo-specific affordances (generated-file skipping, `//nolint` directive support, suggested-fix generation with import management). Adding an external tool dependency would require integrating and configuring a separate toolchain and would not benefit from those shared utilities.

### Consequences

#### Positive
- Eliminates reflection overhead from `fmt.Sprintf("%t", b)` in flagged call sites, replacing it with the zero-allocation `strconv.FormatBool`.
- Makes bool-to-string conversion intent explicit, consistent with the `sprintfint` rule and the broader repo style of preferring `strconv` for primitive conversions.
- Suggested fixes automatically manage `fmt`/`strconv` imports, reducing friction when applying the rewrite.

#### Negative
- Adds a new custom package (`pkg/linters/sprintfbool/`) that must be maintained as the Go language, analysis framework, or internal utilities evolve.
- Developers unfamiliar with the linter suite will encounter a new diagnostic category and must learn the `strconv.FormatBool` idiom; this is a minor onboarding cost.

#### Neutral
- The linter intentionally excludes named bool types (e.g., `type myBool bool`) to avoid false positives, since `strconv.FormatBool` requires the exact predeclared `bool` type. Callers using named bool types will not see diagnostics.
- Calls using multiple arguments (e.g., `fmt.Sprintf("%t %t", a, b)`) or different format verbs (e.g., `%v`) are also excluded, preserving `fmt.Sprintf` where it provides genuine multi-arg formatting.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
