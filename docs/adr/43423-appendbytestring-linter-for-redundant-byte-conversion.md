# ADR-43423: Add appendbytestring Linter to Flag Redundant []byte Conversion in append Calls

**Date**: 2026-07-05
**Status**: Draft
**Deciders**: pelikhan, linter-miner (automated)

---

### Context

The gh-aw codebase maintains a suite of custom Go static analysis linters via `golang.org/x/tools/go/analysis` registered in a `multichecker` binary. A recurring anti-pattern was identified in `pkg/workflow/action_cache.go` and potentially elsewhere: `append(b, []byte(s)...)` where `b` is `[]byte` and `s` is a string. This conversion is unnecessary because Go natively allows `append(b, s...)` to append a string directly to a byte slice without allocating an intermediate `[]byte`. Each redundant conversion allocates a temporary byte slice on the heap, adding GC pressure in hot paths that construct byte buffers from string literals.

### Decision

We will add a new custom linter package `pkg/linters/appendbytestring` implementing a `golang.org/x/tools/go/analysis` pass that detects `append(b, []byte(s)...)` where the first argument is `[]byte` and the inner argument of the `[]byte(...)` conversion is a `string`. The linter provides a suggested fix that rewrites the expression to the idiomatic `append(b, s...)`. It is registered in the `cmd/linters/main.go` multichecker, respects `//nolint:appendbytestring` suppressions, and skips test files. This choice integrates naturally with the existing in-house linter framework already used for similar micro-pattern enforcement.

### Alternatives Considered

#### Alternative 1: Rely on an external linter (e.g., gocritic or staticcheck)

`gocritic` and `staticcheck` are popular third-party linters that detect many Go anti-patterns. `gocritic` in particular has a `appendAssign` check and similar micro-pattern checkers. However, gh-aw's custom multichecker pipeline does not currently integrate these tools end-to-end, and adopting one would introduce a large external dependency for the sake of a single narrowly-scoped check. Furthermore, the existing in-house linter infrastructure (shared `astutil`, `filecheck`, `nolint` packages) already provides the scaffolding needed to write focused analyzers with minimal friction. The net cost of adding external tooling outweighs the benefit for a single pattern.

#### Alternative 2: One-off manual cleanup without ongoing enforcement

The specific instances in `pkg/workflow/action_cache.go` could be fixed manually in a single PR without adding a linter. This would remove the existing allocation overhead but would not prevent the pattern from re-appearing in future code. Given that the anti-pattern is easy to write by habit (especially when converting from C or Java backgrounds where explicit conversions are common), enforcement at CI time is preferable over relying solely on code review vigilance to catch regressions.

### Consequences

#### Positive
- New instances of `append(b, []byte(s)...)` will be caught at CI analysis time, preventing performance regressions from this specific allocation pattern.
- The suggested fix enables automated or semi-automated remediation, reducing the manual effort needed to bring existing code into compliance.
- Consistent with the established pattern of in-house micro-linters, keeping the enforcement surface unified within a single multichecker binary.

#### Negative
- Adds one more analysis pass to the CI linter run, marginally increasing analysis time (though `golang.org/x/tools/go/analysis` passes are generally fast).
- Developers unfamiliar with the Go idiom of appending a string directly to a `[]byte` may initially find the diagnostic confusing until they understand the underlying language semantics.

#### Neutral
- The linter skips test files by convention (consistent with other linters in this suite), meaning test code using the pattern will not be flagged even if it is technically wasteful.
- The `//nolint:appendbytestring` escape hatch is available for the rare legitimate use case where the explicit `[]byte(...)` conversion aids readability at the cost of an allocation.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
