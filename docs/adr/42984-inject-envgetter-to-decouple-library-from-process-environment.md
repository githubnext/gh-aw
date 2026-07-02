# ADR-42984: Inject EnvGetter to Decouple Library Code from Process Environment

**Date**: 2026-07-02
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Library packages `pkg/workflow/compilerenv` and `pkg/workflow/action_mode` were calling `os.Getenv` directly to read runtime configuration from the process environment. A custom linter (`osgetenvlibrary`) flags direct `os.Getenv` calls in library code because they couple the library to the process environment, making the code harder to test reliably and introducing implicit global state. Seven linter violations existed across these two packages before this change. Library code that reads the process environment directly cannot be tested without mutating the process environment (via `t.Setenv`), which is not concurrency-safe.

### Decision

We will introduce an `EnvGetter func(string) string` type that mirrors `os.Getenv`'s signature and inject it into a new `Manager` struct in `compilerenv` and into the private `detectActionMode` function in `action_mode`. All `os.Getenv` call-sites inside library logic are replaced with calls through the injected getter. A package-level `defaultManager = New(os.Getenv)` and a thin `DetectActionMode` wrapper pass `os.Getenv` as a *function reference* (not a call), satisfying the linter. All existing package-level `Resolve*` function signatures are preserved as one-line wrappers so callers require no changes.

### Alternatives Considered

#### Alternative 1: Suppress linter violations with `//nolint:osgetenvlibrary`

Inline suppression comments would silence the linter immediately with zero refactoring. This was rejected because it accumulates technical debt — the underlying coupling between library logic and process state remains — and any future audit would surface the same violations without the context of why they were suppressed.

#### Alternative 2: Control environment in tests with `t.Setenv`

Using `t.Setenv` in unit tests allows setting environment variables for the duration of a test and automatically restores the original value afterward. This was rejected because it still requires mutating the process environment (which is not safe under `t.Parallel()`), does not change the library's dependency on `os.Getenv` at the source, and would continue to trigger the `osgetenvlibrary` linter.

#### Alternative 3: Accept an `*EnvSource` interface

Defining an interface with a `Getenv(string) string` method is idiomatic for larger systems. It was not chosen because a plain function type (`EnvGetter`) achieves the same result with less ceremony, and `os.Getenv` can be passed directly without a wrapper adapter.

### Consequences

#### Positive
- All seven `osgetenvlibrary` linter violations are eliminated, unblocking CI
- Library functions are now independently testable with a map-backed `EnvGetter` — no process-environment mutation needed in tests
- The `Manager` struct can be constructed per-test with deterministic, concurrency-safe env values
- Zero public API breakage: all existing `Resolve*` package-level function signatures are preserved

#### Negative
- The `compilerenv` package now exports three new symbols (`EnvGetter`, `Manager`, `New`) that become part of its public API surface and must be maintained
- New callers who want custom env control must learn the `Manager` constructor pattern; the old "just call the package-level function" model is less visible
- The `action_mode` private function split (`DetectActionMode` → `detectActionMode`) adds a layer of indirection that must be kept in sync if the public function's signature changes

#### Neutral
- Tests that previously relied on `t.Setenv` can be migrated to the injected getter pattern, but existing tests remain valid and are not broken
- The `defaultManager` package-level variable is initialised at program startup and cannot be swapped at runtime (intentional: runtime env injection is not a goal of this change)

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
