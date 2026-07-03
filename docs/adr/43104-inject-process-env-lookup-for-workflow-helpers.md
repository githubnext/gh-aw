# ADR-43104: Inject Process Env Lookup for Workflow Helpers via Global Setter

**Date**: 2026-07-03
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `osgetenvlibrary` linter flags direct `os.Getenv` calls in library code because they couple library logic to the process environment, making unit tests require process environment mutation (`t.Setenv`), which is not safe under `t.Parallel()`. PR #43104 extends env-access decoupling—previously applied to `compilerenv` and `action_mode` (see ADR-42984)—to `pkg/workflow/github_cli.go` and `pkg/workflow/features.go`. Unlike the earlier packages, `github_cli` and `features` expose non-exported functions (`setupGHCommand`, `isFeatureInEnvironment`) that have no existing struct or constructor to hang an injected getter on, and callers do not go through a single manager.

### Decision

We will introduce a package-level `processEnvLookup` variable (type `func(string) (string, bool)`, defaulting to `os.LookupEnv`) protected by a `sync.RWMutex`, and expose a `SetProcessEnvLookup` setter for tests and callers that need to override env resolution. All call sites that previously called `os.Getenv` directly in `github_cli` and `features` are replaced with a thin `lookupProcessEnv(key)` helper that acquires the read lock, calls the configured function, and discards the existence flag to preserve `os.Getenv` semantics. Default behavior is unchanged at runtime.

### Alternatives Considered

#### Alternative 1: Function-parameter injection (pass env lookup as argument)

Each function that reads the environment (`setupGHCommand`, `isFeatureInEnvironment`) could accept a `func(string) string` parameter. This eliminates global state and makes the dependency explicit at each call site. It was not chosen because both functions are internal helpers called through several layers of public API (`ExecGH`, `isFeatureEnabled`), and threading an extra parameter through the entire call chain would require cascading signature changes with no caller-visible benefit — the public API surface would grow without improving encapsulation.

#### Alternative 2: Introduce a Manager struct (following ADR-42984's constructor-injection pattern)

A `Manager` struct could hold an injected `EnvGetter` and expose methods for `SetupGHCommand` and `IsFeatureInEnvironment`. This is the pattern used in `compilerenv` (ADR-42984) and makes the dependency explicit at construction time. It was not chosen here because `github_cli` and `features` are currently flat packages of package-level functions; introducing a manager struct would require every call site to instantiate or share a manager, a larger refactor outside the stated scope of this PR.

#### Alternative 3: Suppress linter violations with `//nolint:osgetenvlibrary`

Inline suppression comments would silence the linter with no code change. Rejected for the same reason as in ADR-42984: the coupling between library logic and the process environment remains, future audits surface the same violations without context, and testability does not improve.

### Consequences

#### Positive
- All `osgetenvlibrary` violations in `github_cli` and `features` are eliminated without changing public API signatures
- `setupGHCommand` and `isFeatureInEnvironment` can be exercised in tests with a synthetic env map, removing the need for `t.Setenv` and enabling safe parallel test execution
- `sync.RWMutex` ensures the lookup function can be swapped in tests without data races even under parallel test execution
- The `lookupProcessEnv` helper centralises env access for the entire `pkg/workflow` package, making future audits straightforward

#### Negative
- Introduces a package-level mutable singleton (`processEnvLookup`), which diverges from ADR-42984's constructor-injection approach; the repository now has two coexisting env-DI patterns that future contributors must understand and choose between
- Tests that override `processEnvLookup` via `SetProcessEnvLookup` must call `SetProcessEnvLookup(nil)` (or use `t.Cleanup`) to restore the default — a hidden invariant that can cause test pollution if cleanup is omitted

#### Neutral
- The `func(string) (string, bool)` signature mirrors `os.LookupEnv` (two return values) rather than the `EnvGetter func(string) string` type defined in ADR-42984; callers that want a unified env abstraction across the whole `pkg/workflow` package will need an adapter
- Existing integration tests that rely on real process environment values remain valid; no tests are broken by this change

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
