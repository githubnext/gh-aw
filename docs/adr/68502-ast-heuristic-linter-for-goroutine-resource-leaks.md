# ADR-68502: AST-Heuristic In-Repo Linter for Goroutine Resource Leaks

**Date**: 2026-10-21
**Status**: Draft
**Deciders**: Unknown (drafted by the Design Decision Gate from PR #68502 evidence)

---

### Context

`gh-aw` launches goroutines throughout the CLI and compiler layers; the PR description reports 92 goroutine launches repository-wide, 41 of them in `pkg/cli/` alone. Several of those launches acquire resources — cancellable contexts, mutexes, channels — whose cleanup is not guaranteed, which can leak contexts and goroutines or deadlock when a function panics. The repository already maintains a family of first-party `golang.org/x/tools/go/analysis` passes under `pkg/linters/` (for example `goroutinemissingrecover`, `httpnoctx`), each with a `testdata/src/a` fixture and registration in `pkg/linters/registry.go`, and `make golint-custom` runs them in CI. The question is how to enforce goroutine resource-cleanup hygiene mechanically rather than through review discipline.

### Decision

We will add a new first-party `go/analysis` pass, `pkg/linters/goroutineresourceleak`, that inspects `*ast.GoStmt` nodes whose call target is a function literal and flags three syntactic patterns inside the literal body: `context.WithCancel`/`WithTimeout`/`WithDeadline` assignments whose cancel variable is never deferred, `Lock()`/`RLock()` calls on `sync.Mutex`/`sync.RWMutex` without a deferred `Unlock()`/`RUnlock()`, and `make(chan T)` results that are never sent to, received from, or closed. The analyzer is registered in `allAnalyzers` in `pkg/linters/registry.go` and reuses the shared `internal/analyzerutil`, `internal/astutil`, `internal/filecheck`, and `internal/nolint` helpers, so it inherits generated-file skipping and `//nolint:goroutineresourceleak` suppression. The primary driver is consistency with the existing linter architecture plus the ability to fail CI on new occurrences at zero marginal infrastructure cost.

### Alternatives Considered

#### Alternative 1: Adopt an existing third-party leak detector (e.g. `go.uber.org/goleak`, `golangci-lint`'s `contextcheck`/`sqlclosecheck` family)

Off-the-shelf tooling would require no analyzer code to maintain. It was rejected because the needed rule set is specific to goroutine *literals* in this codebase and spans three distinct resource kinds; `goleak` is a runtime test-time detector (it needs tests that actually exercise the leaking path, which these launch sites largely lack), and no single existing static rule covers context + mutex + channel cleanup inside `go func()` bodies. Adding a third-party dependency also widens the supply-chain surface for a rule that is ~380 lines locally.

#### Alternative 2: Extend the existing `goroutinemissingrecover` analyzer with the new checks

Since `goroutinemissingrecover` already walks `*ast.GoStmt` nodes with function literals, folding the three resource checks into it would avoid a new package and a second AST traversal. It was rejected because it conflates unrelated diagnostics under one linter name, which makes `//nolint` suppression coarse-grained (you could not silence a channel heuristic without also silencing panic-recovery enforcement) and makes the rule harder to enable/disable independently via `linters -goroutineresourceleak`. This was a close call on traversal cost, but diagnostic separability won.

#### Alternative 3: Use full dataflow/SSA analysis instead of block-level AST heuristics

A `buildssa`-backed pass could follow cancel functions and channels across control flow and across helper calls, eliminating the "only top-level statements of the literal body" limitation of the current `checkCancelLeak`/`checkMutexLeak` implementations. It was rejected for this first iteration because of implementation and runtime cost; the PR reports zero false positives over `./cmd/...` and `./pkg/...`, so the cheaper heuristic already meets the immediate need.

### Consequences

#### Positive
- New goroutine resource leaks are caught mechanically in CI via `make golint-custom` rather than relying on reviewer vigilance.
- The analyzer follows the established `pkg/linters/` conventions (shared `analyzerutil`/`nolint`/`filecheck` helpers, `testdata/src/a` fixture, registry entry), so it needs no new build, test, or CLI wiring.
- Reported coverage of 89.2% with 18 fixture cases (8 positive, 10 negative) documents the intended true-positive and true-negative behaviour for future maintainers.
- Suppression is per-linter (`//nolint:goroutineresourceleak`), so deliberate exceptions stay narrowly scoped and greppable.

#### Negative
- The checks are block-level heuristics, not dataflow: `checkCancelLeak` and `checkMutexLeak` only iterate `body.List`, so a `Lock()` or `context.WithCancel` nested inside an `if`, `for`, or inner block is silently missed — the linter gives weaker guarantees than its diagnostic text implies.
- The channel rule ("created but never sent/received/closed in this body") is an approximation; a channel created in a goroutine and passed to a helper function will be reported as a leak, so future code may need `//nolint` escapes.
- Only `go func() { ... }()` literals are analysed; `go namedFunc()` is out of scope by design, leaving a documented blind spot.
- One more analyzer in `allAnalyzers` adds to custom-lint wall time on every CI run, and the rule becomes repository API: changing its heuristics later can break builds that were previously green.

#### Neutral
- `isMutexType` accepts `sync/atomic` as a package path for `Mutex`/`RWMutex`, which matches no real type; it is harmless today but is dead breadth that may confuse future readers.
- The PR description mentions a fourth intended pattern (goroutines launched in loops without a `WaitGroup`), but no corresponding check is implemented — the scope is three rules, not four.
- Diagnostics are mutually exclusive per goroutine: the first matching category returns early, so a literal with both a cancel leak and a mutex leak reports only the cancel leak.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
