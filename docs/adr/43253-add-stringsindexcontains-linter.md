# ADR-43253: Add stringsindexcontains Custom Go Analysis Linter

**Date**: 2026-07-04
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `gh-aw` codebase uses a custom Go static analysis framework (`pkg/linters/`) to enforce code style and correctness automatically. A scan of `pkg/` and `cmd/` revealed 42 occurrences of the pattern `strings.Index(s, substr) != -1` (and equivalent variants such as `>= 0`, `== -1`, `< 0`). These comparisons are semantically equivalent to `strings.Contains(s, substr)` or `!strings.Contains(s, substr)`, but are less readable and more error-prone to write correctly. The codebase already ships several analogous custom linters (e.g., `stringreplaceminusone`, `lenstringzero`) that follow the same `go/analysis` pass pattern.

### Decision

We will add a new `stringsindexcontains` custom `go/analysis` linter to `pkg/linters/stringsindexcontains/` that reports `strings.Index(s, substr)` comparisons against `-1` or `0` and suggests replacing them with `strings.Contains(s, substr)` or `!strings.Contains(s, substr)`. The linter will emit `SuggestedFix` text edits to enable automated batch repair. It will be registered in `cmd/linters/main.go` alongside existing analyzers.

### Alternatives Considered

#### Alternative 1: Rely on manual code review

Code reviewers would be expected to flag `strings.Index` containment-check patterns during PR review. This approach requires no new tooling and imposes no maintenance burden. It was not chosen because it is inconsistent — reviewers can miss patterns, especially in large diffs — and it does not help with the 42 existing occurrences already in the codebase.

#### Alternative 2: Enable an existing third-party linter

Linters such as `gocritic` (via `sloppyReassign` or similar heuristics) or `staticcheck` cover some idiomatic Go patterns. Using a pre-built linter avoids writing and maintaining custom code. This was not chosen because no widely-adopted third-party linter precisely covers all six operator/literal combinations (`!= -1`, `>= 0`, `> -1`, `== -1`, `< 0`, `<= -1`) with yoda-order variants and automated fix suggestions, and integrating a new external dependency would require vetting and approval across the toolchain.

### Consequences

#### Positive
- Automatically detects all six semantic variants of the `strings.Index` containment-check anti-pattern, including yoda-order forms.
- Provides `SuggestedFix` text edits that allow the linter runner to apply fixes automatically, enabling bulk remediation of the 42 known occurrences.
- Follows the established pattern for custom linters in this codebase, making future maintenance and extension straightforward.

#### Negative
- Adds a new package (`pkg/linters/stringsindexcontains/`) that must be maintained when upstream `go/analysis` APIs or internal utilities change.
- The linter will flag only the specific operator/literal combinations listed; any variant that was intentionally left as `strings.Index` for a non-containment reason (e.g., comparing against a non-`-1`/`0` threshold) is already excluded by design, but edge cases may surface over time.

#### Neutral
- The linter is registered in `cmd/linters/main.go` alongside all other custom analyzers; it participates in the same execution pipeline with no special orchestration.
- Test fixtures in `pkg/linters/stringsindexcontains/testdata/` document both flagged and acceptable patterns, serving as living documentation of the linter's intent.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
