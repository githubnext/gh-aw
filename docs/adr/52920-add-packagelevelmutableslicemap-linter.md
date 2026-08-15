# ADR-52920: Add packagelevelmutableslicemap Static Analysis Linter

**Date**: 2026-08-15
**Status**: Draft
**Deciders**: pelikhan

---

### Context

Package-level `var` declarations of slices and maps in Go are shared across every goroutine and every call into the package for the lifetime of the process. When a function body mutates such a variable (via `append()` re-assignment, index assignment, or `delete()`), it risks data races under concurrent access and can leak state across otherwise-unrelated calls (e.g. between test runs or concurrent request handlers).

Issue #52683 identified a concrete instance of this pattern in the codebase — a module-level mutable array (`_allResults`) reset and appended to across `main()` invocations, which is a latent cross-run data-corruption risk if the calling architecture ever changes to parallel dispatch. A broader codebase scan found additional package-level slice/map `var` declarations in `pkg/` that exhibit the same structural risk. The project's existing `go/analysis`-based linter suite in `pkg/linters/` already enforces similar structural constraints (e.g. `manualmutexunlock`, `goroutinemissingrecover`) and provides shared infrastructure (`analyzerutil`, `filecheck`, `nolint`) that makes adding a new checker straightforward.

No built-in Go tool or existing third-party linter in the project's toolchain specifically targets this class of mutation at static analysis time.

### Decision

We will add a new custom `go/analysis` linter, `packagelevelmutableslicemap`, under `pkg/linters/packagelevelmutableslicemap/`. The analyzer scans package-scope `var` declarations with slice or map underlying types and flags any mutation of those variables from inside a function body via `append()` re-assignment, index assignment (`m[k] = v`), or `delete(m, k)`. Object identity via `types.Object` is used to correctly exempt local variables that shadow a package-level name. A `//nolint:packagelevelmutableslicemap` directive on the mutating line suppresses the diagnostic, consistent with sibling linters. The analyzer is registered in `pkg/linters/registry.go` alongside the existing 64 analyzers.

### Alternatives Considered

#### Alternative 1: Rely on Go's built-in race detector (`-race`)

The race detector (`go test -race` / `go run -race`) detects data races dynamically at runtime. It is thorough but requires actual concurrent execution of the conflicting code paths during the test run. Mutations of package-level slices/maps that happen sequentially — or that are only exercised under production load — will not trigger it. It also does not catch the cross-call state-leak pattern (where sequential mutations corrupt state for a later call), which is a distinct risk from a data race. Static analysis at lint time catches the structural smell unconditionally, regardless of how tests are structured.

#### Alternative 2: Code review convention and documentation

The team could document a convention prohibiting mutable package-level slice/map state and rely on human reviewers to enforce it. This has zero tooling cost but scales poorly: conventions drift, reviewers miss cases under time pressure, and new contributors are unaware of the rule until they encounter a review comment. Given that the project already invests in automated linting for analogous structural constraints, automation is the consistent choice.

### Consequences

#### Positive
- Package-level mutable slice/map bugs are caught at lint time — before they manifest as intermittent data races or cross-test contamination in production or CI.
- The implementation follows established patterns in `pkg/linters/`, reusing `analyzerutil`, `filecheck`, and `nolint` infrastructure; the incremental cost per new analyzer is low, and the approach is familiar to contributors.
- The `//nolint:packagelevelmutableslicemap` escape hatch allows intentionally synchronized global state (e.g., a mutex-protected registry) to opt out without refactoring.

#### Negative
- The analyzer will produce false positives for package-level state that is intentionally and correctly synchronized (e.g., a `sync.Mutex`-protected global cache). Each such site requires a `//nolint` directive or a refactor; if there are many such sites in the codebase, this creates short-term maintenance work.
- The linter cannot detect all mutable-shared-state hazards — it only covers `var` declarations with `slice` or `map` underlying types mutated directly. Aliased mutations (e.g., passing the global slice to a helper that appends internally) are not flagged.

#### Neutral
- The analyzer count in `pkg/linters/doc.go` and `README.md` increases from 64 to 65; the `spec_test.go` count assertion is updated accordingly. This is a minor bookkeeping change with no behavioral impact.
- Existing code in the repository that triggers the new linter will begin failing lint checks once the analyzer is active; a sweep of existing violations may be needed before enabling the linter in CI.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
