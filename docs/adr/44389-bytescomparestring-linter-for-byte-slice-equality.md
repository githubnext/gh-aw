# ADR-44389: Add `bytescomparestring` Linter to Flag Allocating `[]byte` Equality Checks

**Date**: 2026-07-08
**Status**: Draft
**Deciders**: Unknown (automated PR by Linter Miner)

---

### Context

Comparing two `[]byte` values via `string(a) == string(b)` or `string(a) != string(b)` compiles successfully but causes two unnecessary heap allocations per comparison — one for each `string()` conversion. The allocation-free idiomatic alternative is `bytes.Equal(a, b)` and `!bytes.Equal(a, b)`. A code-pattern scan found 3 live instances of this pattern in the repo (`pkg/parser/import_bfs.go:566`, `pkg/parser/tools_merger.go:279`, `pkg/cli/packages.go:146`). The existing custom linter suite (`pkg/linters/`) is the established mechanism for enforcing repo-wide Go idioms at compile time with automatic suggested fixes.

### Decision

We will add a new `go/analysis` linter named `bytescomparestring` that detects `string(a) == string(b)` and `string(a) != string(b)` expressions where both `a` and `b` have underlying type `[]byte`, reports a diagnostic, and emits a `SuggestedFix` that rewrites the expression to `bytes.Equal(a, b)` or `!bytes.Equal(a, b)`. The linter is registered in `cmd/linters/main.go` and documented in `pkg/linters/doc.go`. False-positive rate is zero: the linter only fires when both sides are `string(x)` type-conversions with a `[]byte` argument.

### Alternatives Considered

#### Alternative 1: Rely on an External Tool (staticcheck / golangci-lint)

Staticcheck's `SA4010` and similar rules cover some allocation patterns, but none specifically target the `string([]byte) == string([]byte)` form with auto-fix support. Adopting an external tool for this single pattern would add a build-tool dependency and complicate CI configuration, while the project already maintains a bespoke linter framework with consistent nolint directives, file-skip logic, and suggested-fix plumbing. Not chosen because the marginal benefit of a third-party tool does not justify adding a dependency.

#### Alternative 2: Use `slices.Equal` or `reflect.DeepEqual`

`slices.Equal` (Go 1.21+) and `reflect.DeepEqual` are functionally correct alternatives to `bytes.Equal` for `[]byte` comparisons. However, `reflect.DeepEqual` is significantly slower due to reflection overhead, and `slices.Equal` is a generic function that is less recognizable in idiomatic byte-handling code than `bytes.Equal`. The Go standard library convention for byte-slice equality is `bytes.Equal`; the linter's suggested fix should produce idiomatic output.

### Consequences

#### Positive
- Eliminates two unnecessary heap allocations per `[]byte` equality comparison at the 3 known sites and any future occurrences.
- Provides an auto-applicable `SuggestedFix` compatible with `gopls` and `go fix`, reducing friction to fix violations.
- Zero false-positive rate: the linter's type-aware check (`pass.TypesInfo`) ensures it only fires when both sides are `string()` conversions of `[]byte` arguments.
- Consistent enforcement: future `string(a) == string(b)` regressions are caught at lint time rather than discovered in code review.

#### Negative
- Callers that are auto-fixed will need the `bytes` package imported; though `goimports` handles this automatically, editors without import management will see a compile error until the import is added manually.
- Adds a new linter to maintain: API-breaking changes in `golang.org/x/tools/go/analysis` would require updating this analyzer alongside the 39 others.

#### Neutral
- The linter skips test files (`filecheck.IsTestFile`) consistent with the project's convention; `string(a) == string(b)` patterns in test code are not flagged.
- The linter count in `pkg/linters/doc.go` increases from 39 to 40; this counter must be updated whenever analyzers are added or removed.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
