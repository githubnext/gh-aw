# ADR-53221: Extract Shared resourcetracker Framework for Deferred-Cleanup Linters

**Date**: 2026-08-16
**Status**: Accepted
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Three linters in `pkg/linters` — `contextcancelnotdeferred`, `fileclosenotdeferred`, and `manualmutexunlock` — each contained a near-identical 60–90 line state machine: acquire a resource, walk the AST inside a function body, detect a non-deferred cleanup call, and report at the acquisition site. Only the tracked object type and diagnostic text differed across the three implementations. Because the logic was duplicated, edge-case fixes (for reassignment, closure boundaries, and `//nolint` suppression) had to be discovered and applied three times, creating an ongoing risk of inconsistent behavior across linters.

### Decision

We will extract the shared bookkeeping into a new internal package, `pkg/linters/internal/resourcetracker`, built around a generic `Config[K comparable]` type and a `NewAnalyzer` constructor. Each linter supplies only its semantics via two callbacks — `Acquisitions` (identifies resource bindings and their tracking key) and `CleanupKey` (identifies which tracked resource a cleanup call targets). All common logic — analyzer construction, nolint/generated-file indexing, `FuncDecl` traversal that stops at `FuncLit`, per-resource state tracking, reassignment detection, suppression filtering, and final diagnostics sorted by acquisition position — lives in `resourcetracker` and is exercised once.

### Alternatives Considered

#### Alternative 1: Keep Duplicated Code Per Linter

Continue maintaining separate, copy-pasted state machines in each linter. This is the status quo and requires the least up-front effort. It was rejected because every edge-case fix or behavioral improvement must be found and applied in each linter independently, which has already caused divergence (e.g., cleanup-in-assignment-RHS detection was present only in `fileclosenotdeferred`).

#### Alternative 2: Code Generation (`go:generate`)

Use a Go code-generation tool to stamp out per-linter boilerplate from a shared template, eliminating the runtime dependency on a shared package while avoiding hand-written generics. This was rejected because generated code still must be re-generated and committed for each change, the diff noise is higher, and the template-to-output mapping makes debugging harder than reading the generic implementation directly.

#### Alternative 3: Non-Generic Shared Package with Per-Type Adapters

Extract a non-generic package using `interface{}` or `any` as the key type, with per-linter adapter structs. This avoids Go generics but requires runtime type assertions and loses compile-time key-type safety. The composite `mutexKey` case in `manualmutexunlock` makes type-safe generics the cleaner fit; a non-generic approach would need unsafe casts or reflection.

### Consequences

#### Positive
- Approximately 380 lines of duplicated control-flow code removed from the three linters.
- Edge-case correctness (reassignment, closure boundary, nolint suppression) is now guaranteed to be consistent across all linters that use `resourcetracker`.
- Diagnostic output is now deterministic (sorted by acquisition position) regardless of Go map-iteration order.
- New resource-tracking linters can be added by supplying only two callbacks rather than copying a full state machine.

#### Negative
- Linters that previously had zero internal dependencies now depend on `pkg/linters/internal/resourcetracker`; a bug in the shared package affects all consumers simultaneously.
- The generic `Config[K comparable]` API is more abstract than the previous inline code, increasing the cognitive load for contributors unfamiliar with Go generics.

#### Neutral
- Existing `testdata` directories for all three linters are unchanged; the `analysistest` suites act as behavior-preservation checks.
- The `resourcetracker` package itself gains its own `analysistest` suite covering the shared scenarios (manual cleanup, deferred cleanup, closure isolation, reassignment, shadowing, and `//nolint` suppression).
- `manualmutexunlock` retains its composite `mutexKey{base, field}` by instantiating the framework at that key type, so `a.mu` and `b.mu` remain independently tracked without special-casing.
