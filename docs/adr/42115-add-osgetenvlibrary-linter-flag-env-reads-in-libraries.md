# ADR-42115: Add osgetenvlibrary Linter to Flag os.Getenv/LookupEnv in Library Packages

**Date**: 2026-06-28
**Status**: Draft
**Deciders**: Unknown (automated PR by linter-miner)

---

### Context

Library packages (non-main, non-test Go packages) that call `os.Getenv` or `os.LookupEnv` directly couple themselves to the process environment, making configuration invisible to callers, hiding dependency paths from function signatures, and requiring tests to manipulate environment variables as side effects. A code scan of `pkg/` found six existing call sites in non-main library packages. The repository already enforces a complementary rule (`ossetenvlibrary`) that prevents library code from *writing* to the process environment; this PR extends that boundary to cover *reading* from it.

### Decision

We will add a new `osgetenvlibrary` static analysis linter that flags any call to `os.Getenv` or `os.LookupEnv` in non-main, non-test Go packages. The analyzer uses type-aware `*types.Func` matching (the same approach as `ossetenvlibrary`) and supports `//nolint:osgetenvlibrary` escape hatches for exceptional cases. Library authors must pass configuration through explicit parameters, constructor arguments, or config structs instead of reading from the process environment.

### Alternatives Considered

#### Alternative 1: Documentation Only — Accept Environment Coupling and Publish a Convention

Acknowledge `os.Getenv` usage in library code as acceptable and document the pattern as a team convention rather than enforcing it via a linter. This preserves short-term development velocity and requires no refactoring of existing call sites. It was not chosen because undocumented conventions drift over time, the six existing violations demonstrate the pattern is already spreading, and documentation alone provides no enforcement or discoverability in the IDE/CI loop.

#### Alternative 2: Extend the Existing ossetenvlibrary Linter Rather Than Creating a New Package

Add `Getenv`/`LookupEnv` detection directly into the existing `ossetenvlibrary` analyzer to keep the two concerns in one place. This was not chosen because combining read and write checks in a single analyzer conflates two distinct concerns (environment pollution vs. hidden reads), makes the linter name misleading, and complicates targeted suppression—callers who need to suppress a write check but not a read check (or vice versa) would have no granular escape hatch.

### Consequences

#### Positive
- Library packages become independently testable: callers can supply configuration through explicit parameters without setting environment variables.
- Configuration dependencies are made visible in function and constructor signatures, improving API discoverability and reducing hidden coupling.
- Completes the environment-boundary enforcement story alongside `ossetenvlibrary`, covering both reads and writes.

#### Negative
- Six existing `pkg/` call sites must be refactored to thread configuration explicitly through call stacks, representing near-term churn.
- Library authors who currently rely on `os.Getenv` for optional defaults must update their APIs, which may involve adding new parameters or config structs to public interfaces.

#### Neutral
- The `//nolint:osgetenvlibrary` escape hatch is available for cases where environment reads are genuinely appropriate (e.g., environment-inspection utilities), but each suppression requires explicit opt-out rather than opt-in.
- Main packages (`cmd/` paths and packages named `main`) and test files are exempt from the rule, consistent with the `ossetenvlibrary` scoping.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
