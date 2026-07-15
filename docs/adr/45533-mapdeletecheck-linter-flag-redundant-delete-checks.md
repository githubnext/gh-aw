# ADR-45533: Add mapdeletecheck Linter to Flag Redundant Map Existence Checks Before delete

**Date**: 2026-07-15
**Status**: Draft
**Deciders**: Unknown (automated PR by Linter Miner workflow)

---

### Context

Go's built-in `delete(m, k)` is a no-op when key `k` is absent from map `m` — this is guaranteed by the Go specification. However, developers unfamiliar with this guarantee often defensively guard the call with an existence check:

```go
if _, ok := m[k]; ok {
    delete(m, k)
}
```

This pattern adds code noise, extra cyclomatic complexity, and cognitive overhead without any correctness benefit. The codebase already runs a suite of custom Go static analysis linters (`pkg/linters/`) to enforce idiomatic patterns. This linter fits naturally into that suite.

### Decision

We will add a new custom `go/analysis` pass, `mapdeletecheck`, that reports any `if _, ok := m[k]; ok { delete(m, k) }` block where the existence check is provably redundant — specifically, where there is no else branch and the body is exactly the `delete(m, k)` call with the same map and key as the index expression. The analyzer includes a suggested fix that rewrites the entire block to the bare `delete(m, k)` call. It will be registered alongside all other custom linters in `cmd/linters/main.go`.

### Alternatives Considered

#### Alternative 1: Rely on Code Review

Human reviewers could catch and flag this pattern during PR review. This requires no new tooling and imposes no maintenance burden. It was not chosen because human review is inconsistent — the pattern can easily slip through, especially in larger diffs — and the project already invests in automated linting precisely to free reviewers for higher-order concerns.

#### Alternative 2: Adopt an Existing Third-Party Linter

Tools like `staticcheck` and `go-critic` provide broad idiomatic-Go rule sets that may cover or could cover this pattern. Using an upstream linter avoids maintaining custom analysis code. This was not chosen because the project runs its own tailored linter suite where rules are scoped tightly to patterns observed in the codebase; depending on a third-party tool for a single rule adds an external dependency for limited gain and may introduce unrelated rules that require suppression.

### Consequences

#### Positive
- Code noise from the defensive existence-check pattern is automatically detected and can be auto-fixed, keeping the codebase idiomatic.
- The linter scope is deliberately narrow (no else branch, single-statement body, same map and key) minimizing false positives.
- A `SuggestedFix` is embedded so developers can apply the simplification in one step with a supporting tool.
- Consistent enforcement across all contributors, not dependent on reviewer familiarity with the Go spec guarantee.

#### Negative
- Adds one more analyzer to the custom linter suite, which must be maintained if the Go AST APIs or internal utility helpers change.
- Developers encountering this rule for the first time must learn to use `//nolint:mapdeletecheck` or update their code; it may surface as unexpected CI friction in existing code.
- Slightly increases overall static analysis runtime (marginal, one additional AST traversal).

#### Neutral
- The analyzer skips test files (`filecheck.IsTestFile`) consistent with the conventions of other linters in the suite.
- The `sameExpr` helper uses type-object identity for identifiers and falls back to source-text comparison for non-identifier expressions; this is a pragmatic heuristic that matches the approach already used in the codebase.
- Registration in `cmd/linters/main.go` follows the existing alphabetical import and slice-append convention; no structural changes to the binary are required.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
