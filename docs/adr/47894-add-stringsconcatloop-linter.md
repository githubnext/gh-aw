# ADR-47894: Add stringsconcatloop Linter to Detect O(n²) String Concatenation in Loops

**Date**: 2026-07-25
**Status**: Draft
**Deciders**: Unknown

---

### Context

Go's `string` type is immutable. Every `string +=` inside a loop allocates a brand-new string and copies the previous content, yielding O(n²) time and memory as the loop count grows. The idiomatic fix — `strings.Builder` — avoids re-allocation by accumulating bytes and materialising the final string once. The `gh-aw` repository maintains a collection of custom `go/analysis` analyzers to enforce code-quality conventions at lint time; adding a dedicated analyzer for this pattern ensures the issue is caught before code merges rather than discovered in production profiling.

### Decision

We will add a new `stringsconcatloop` analyzer under `pkg/linters/stringsconcatloop/` that walks `*ast.AssignStmt` nodes with `token.ADD_ASSIGN`, verifies the LHS has a `string` underlying type via the type-checker, and reports a diagnostic when the assignment is enclosed within a `for` or `range` loop body — stopping at `func` literal boundaries to avoid false positives on closures. The analyzer honours `//nolint:stringsconcatloop` directives and skips generated files via the existing `filecheck` and `nolint` infrastructure already used by other linters in the collection.

### Alternatives Considered

#### Alternative 1: Rely on Runtime Profiling and Benchmarks

Profile-guided discovery (pprof, benchmarks) would catch hot O(n²) concat paths only after the code is merged and exercised. This defers detection to a later, more expensive phase of the development cycle and misses cold-path code that is rarely profiled. Static analysis at lint time prevents the pattern from entering the codebase at all.

#### Alternative 2: Adopt a Third-Party Linter (gocritic / staticcheck)

`gocritic` (rule `appendAssign`) and `staticcheck` include heuristics that overlap with this pattern, but neither covers the full set of cases the repository cares about (e.g., named string types, cursor-based AST traversal for accurate loop ancestry). Importing a third-party tool would also add a binary dependency and version-management burden, whereas a custom analyzer integrates cleanly with the existing `go/analysis` harness, `nolint` index, and `filecheck` generated-file detection already shared across all 58 analyzers in this collection.

#### Alternative 3: Extend an Existing String-Manipulation Linter

Folding this check into `stringbytesroundtrip` or another existing `string*` analyzer was considered but rejected: each analyzer in the collection has a single, clearly scoped responsibility, and conflating unrelated patterns makes the diagnostic messages ambiguous and the test surface harder to reason about.

### Consequences

#### Positive
- String O(n²) concat patterns are caught at `golangci-lint` / CI run time, before merging.
- The implementation reuses the shared `filecheck`, `nolint`, and `astutil.IsStringType` infrastructure, keeping the new analyzer consistent with the rest of the linter collection.
- Named-string-type aliases (e.g., `type myString string`) are also flagged, closing a gap that a purely syntax-level check would miss.

#### Negative
- Short loops (two or three iterations) incur no meaningful allocation overhead; the analyzer will flag these and require either a `//nolint` suppression or a `strings.Builder` rewrite that is arguably less readable in that context.
- Developers unfamiliar with the linter collection must learn one more rule and its suppression mechanism.

#### Neutral
- The linter count increments from 57 to 58; `doc.go`, `registry.go`, `spec_test.go`, and `README.md` all receive corresponding mechanical updates.
- The `//nolint:stringsconcatloop` escape hatch is available for the rare case where `+=` is intentional and the allocation cost is acceptable.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
