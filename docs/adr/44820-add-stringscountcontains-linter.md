# ADR-44820: Add stringscountcontains Linter

**Date**: 2026-07-11
**Status**: Draft
**Deciders**: pelikhan (PR author)

---

### Context

The `pkg/linters/` package contains a suite of custom Go static-analysis analyzers that enforce project-specific code quality rules. A common Go antipattern is using `strings.Count(s, sub)` compared against `0` or `1` as a containment check (e.g., `strings.Count(s, sub) > 0`). `strings.Count` traverses the entire string to count all occurrences, whereas `strings.Contains` short-circuits on the first match. The antipattern is therefore both less readable — it expresses counting, not containment — and potentially less efficient. No existing linter in the suite flags this pattern.

### Decision

We will add a new `stringscountcontains` analyzer to `pkg/linters/` that flags `strings.Count(s, sub)` comparisons with `0` or `1` (in canonical and yoda order) and emits a `SuggestedFix` rewriting them to `strings.Contains(s, sub)` or `!strings.Contains(s, sub)`. The analyzer is registered in the multichecker binary and documented in the README. It follows the same structural conventions as the existing `stringsindexcontains` linter (AST traversal, `nolint` directive support, `_test.go` file exclusion, `analysistest`-based tests with a `.go.golden` file).

### Alternatives Considered

#### Alternative 1: Rely on external community linters (e.g., `gocritic`)

`gocritic` includes checks that overlap with this pattern. However, integrating a large external linter as a dependency would add transitive dependency weight, reduce control over diagnostic messages and fix behaviour, and introduce versioning friction with the rest of the custom linter suite. The project has deliberately invested in a custom linter framework to keep full control over rule semantics, fix generation, and `nolint` integration.

#### Alternative 2: Manual code review enforcement only

Documenting the preferred pattern in style guides and relying on reviewers to catch it during PR review is zero-cost to implement. However, it does not scale: reviewers are inconsistent, the pattern is easy to miss, and it provides no automated-fix path. The existing suite demonstrates a project preference for automated, machine-checkable rules over guidance-only policies.

#### Alternative 3: No action (accept the pattern as harmless)

The performance difference between `strings.Count` and `strings.Contains` for containment checks is typically negligible. One could argue the antipattern does not warrant enforcement. However, the readability argument is independent of performance: code that uses `strings.Count` to test containment misleads readers about intent, and the project's existing `stringsindexcontains` linter already enforces the analogous `strings.Index` → `strings.Contains` rewrite, establishing a precedent that this family of patterns should be machine-checked.

### Consequences

#### Positive
- All `strings.Count`-as-containment antipatterns in non-test code will be flagged automatically, improving readability and intent clarity.
- The `SuggestedFix` allows batch automated repair (e.g., via `gopls` or `go fix`), reducing developer friction when adopting the rule on an existing codebase.
- Extends the existing linter suite consistently: the `stringsindexcontains` precedent is maintained.

#### Negative
- Existing code that uses the flagged pattern must be fixed or suppressed with `//nolint:stringscountcontains`, which is a one-time migration cost.
- Adds a new package to maintain; future changes to `strings.Count` semantics or Go AST APIs require updating this analyzer.

#### Neutral
- The linter skips `_test.go` files, so test code using the pattern for clarity is unaffected.
- The 40-analyzer count in `spec_test.go` and README must be kept in sync when adding future linters — this PR establishes that the count is a documented invariant.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
