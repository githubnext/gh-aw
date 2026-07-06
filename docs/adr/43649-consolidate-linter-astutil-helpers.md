# ADR-43649: Consolidate Duplicated AST/Context Helpers into internal/astutil

**Date**: 2026-07-06
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `pkg/linters` suite contained multiple independent linter packages, each with its own private copies of the same small helper functions: `enclosingFuncType` (extract `*ast.FuncType` from a `FuncDecl` or `FuncLit`), `contextContextType` (resolve `context.Context` via type-checker imports), `contextParamName` (find the first `context.Context` parameter name in a function signature), `calledOSFunc` (check whether a call targets a named function in the `os` package), and `flipOp` / `flipComparisonOp` (flip comparison operators for normalization). At least eight analyzer packages (`execcommandwithoutcontext`, `httpnoctx`, `timesleepnocontext`, `ctxbackground`, `osgetenvlibrary`, `ossetenvlibrary`, `lenstringzero`, `stringsindexcontains`) maintained independent copies, creating a realistic drift risk: a correctness fix or a nil-guard addition applied to one copy could silently leave the others broken.

### Decision

We will centralize all shared AST and type-checker utility helpers into the existing `pkg/linters/internal/astutil` package and delete per-package copies. The consolidated helpers are exported as `EnclosingFuncType`, `ContextContextType`, `ContextParamName`, `CalledOSFunc` (redesigned with a variadic `allowedNames` filter for cleaner call sites), and `FlipComparisonOp`. Each affected linter is rewired to import and call the shared versions, and unit tests covering the new functions are added to `astutil_test.go`. The `internal` placement restricts access to packages within `pkg/linters/...`, which is the correct scope.

### Alternatives Considered

#### Alternative 1: Keep Per-Package Copies with No Shared Abstraction

Each analyzer would continue to own its local private helper. This is the status quo and requires zero refactoring effort. It was rejected because identical helper functions across eight-plus packages are a demonstrated maintenance hazard: divergent nil-guards and differing function-name filters were already present across packages, confirming that drift was occurring. There is no mechanism to enforce consistency without a single source of truth.

#### Alternative 2: Code-Generate Per-Package Helpers from a Template

A generator (e.g., using `go generate`) could stamp identical helper bodies into each package from a shared template, keeping copies in sync while preserving package-private visibility. This eliminates the behavioral drift risk without changing import paths. It was rejected because it introduces generator tooling complexity, requires each package's generated file to be regenerated whenever the template changes, and provides no advantage over a shared package for helpers that carry no package-specific state. The `internal` package approach is simpler and idiomatic in the Go standard library and `x/tools` ecosystem.

### Consequences

#### Positive
- Single source of truth for shared helpers: a bug fix or a nil-guard addition needs to be made only once and takes effect across all linters immediately.
- `CalledOSFunc` gains a variadic `allowedNames` parameter, making call sites more readable (`astutil.CalledOSFunc(pass, call, "Getenv", "LookupEnv")` vs. a hardcoded name comparison inside a local copy).
- New linter authors can discover and reuse existing utilities rather than copy-pasting, lowering the per-linter authoring cost.
- Expanded unit test coverage for `astutil` provides a regression harness for the shared utilities independent of any individual linter's test suite.

#### Negative
- The `internal` package boundary limits these helpers to packages under `pkg/linters/...`; code outside that tree (e.g., a future top-level analysis driver) cannot import them without restructuring.
- Callers now take an indirect dependency on `pkg/linters/internal/astutil` — a change to a helper's signature requires updating every consumer in the same commit, rather than each package being independently evolvable.

#### Neutral
- No behavioral change is intended; existing linter diagnostics and suggested fixes should be identical before and after the migration.
- The `astutil` package already existed and was in use; this PR extends it rather than introducing a new dependency.
- Test coverage for the consolidated helpers lives in `astutil_test.go`, separate from individual linter testdata directories.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
