# ADR-45799: Add nilctxpassed Static Analyzer to Catch nil context.Context Arguments

**Date**: 2026-07-15
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Passing `nil` as a `context.Context` argument compiles cleanly in Go because `nil` is a valid zero value for any interface type, including `context.Context`. However, any call to a method on a `nil` context (e.g., `ctx.Deadline()`, `ctx.Done()`, `ctx.Value()`) causes a runtime panic that is difficult to diagnose. The existing linter suite (`pkg/linters/`) already uses the `golang.org/x/tools/go/analysis` framework and shared helpers (`astutil`, `nolint`, `filecheck`) to catch similar classes of bugs at analysis time. Adding `nilctxpassed` closes the gap between "compiles successfully" and "safe to run" for a common mistake pattern.

### Decision

We will add a new `nilctxpassed` analyzer (`pkg/linters/nilctxpassed/nilctxpassed.go`) that inspects every call expression in non-generated Go source files, resolves the callee's `*types.Signature`, and reports a diagnostic whenever the predeclared `nil` (confirmed via `*types.Nil`) is passed in a position whose parameter type is identical to `context.Context`. The analyzer supports variadic positions and respects `//nolint:nilctxpassed` suppression. It is registered in the `cmd/linters/main.go` multichecker binary alongside all existing analyzers.

### Alternatives Considered

#### Alternative 1: Rely on the Go type system alone

Go's compiler rejects most non-interface mismatches, but `nil` is assignable to any interface type including `context.Context`. No compiler error or warning is emitted when a developer writes `f(nil)` where `f` accepts `context.Context`. This alternative provides zero protection against the pattern at build time, so it was rejected.

#### Alternative 2: Runtime guard / defensive nil checks inside every context-accepting function

Each function that accepts a `context.Context` could panic or return an error immediately if `ctx == nil`. This defensive check is idiomatic in some codebases but requires manual discipline across every function signature, adds per-call overhead, produces less actionable error messages than a linter diagnostic with source location, and does not scale to third-party code that does not own the function definitions. It was rejected in favour of a single-point static check that enforces the constraint uniformly.

#### Alternative 3: Use an off-the-shelf linter (e.g., `noctx` from `golangci-lint`)

Existing third-party linters target related but distinct problems (e.g., HTTP requests made without a context). None of the commonly used linters in `golangci-lint` precisely flag nil-as-context-argument using type-identity checks. Additionally, the project's linter suite is a custom multichecker binary that requires analyzers to be written using the same internal helpers (`astutil`, `nolint`, `filecheck`) for consistent nolint suppression and generated-file skipping. Adopting a third-party linter would mean a different suppression mechanism and would not integrate with the shared `nolint.Analyzer` dependency. A bespoke analyzer was preferred.

### Consequences

#### Positive
- Eliminates an entire class of runtime panic (`nil` context) by making it a build-time error, catching bugs before they reach staging or production.
- Follows the established pattern of the linter suite: same `go/analysis` framework, same shared helpers, same `//nolint:<linter>` suppression mechanism — no new concepts for contributors to learn.
- The `analysistest`-based test suite provides deterministic, reproducible verification of both flagged and non-flagged cases without runtime execution.
- Generated files are automatically excluded from analysis via the existing `filecheck.Analyzer` dependency.

#### Negative
- Adds one more analysis pass to the multichecker binary, marginally increasing total analysis time (proportional to the number of call expressions in the codebase).
- The `//nolint:nilctxpassed` escape valve could be misused to suppress legitimate diagnostics in cases where `nil` is intentionally passed (e.g., tests that explicitly exercise nil-context handling). Teams must review nolint suppressions in code review.
- The analyzer only catches the predeclared `nil` identifier (confirmed via `*types.Nil`); a typed `var ctx context.Context` (whose zero value is also nil) would not be flagged, leaving a gap for that pattern.

#### Neutral
- The linter count in `pkg/linters/doc.go` and the spec test count increment from 44 to 45; the spec test `documentedAnalyzers()` list and README table must be kept in sync when adding future analyzers.
- The binary at `linters` (tracked in version control) was updated as part of this change.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
