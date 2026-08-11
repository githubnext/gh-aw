# ADR-52109: Split compiler_types.go by Lifecycle Responsibility

**Date**: 2026-08-11
**Status**: Draft
**Deciders**: pelikhan (via copilot-swe-agent, PR #52109)

---

### Context

`pkg/workflow/compiler_types.go` had grown to mix three distinct lifecycle categories in a single file:

1. **Build-time functional options** — `CompilerOption` type, `With*` builder functions, and the `NewCompiler` constructor.
2. **Post-construction runtime mutators/accessors** — ~38 `Set*/Get*` methods on `*Compiler` (e.g. `SetContext`, `SetStrictMode`, `GetSharedActionResolver`).
3. **Pure type declarations** — the `Compiler` struct, `FileCreationTracker` interface, `logTypes`, `FileCreationTracker`, and `allowedDomain`.

The file exceeded 400 lines and made it hard to locate any given responsibility. This also prevented adding tests alongside each group without placing everything in a single, oversized test file.

### Decision

We will split `pkg/workflow/compiler_types.go` into three focused files within the same Go package (`package workflow`), each owning exactly one lifecycle group:

- **`compiler_options.go`** — `CompilerOption` type, `With*` builders, and `NewCompiler`.
- **`compiler_mutators.go`** — all `*Compiler` setter/getter methods and lazily-initialized shared cache helpers (`ensureSharedActionCacheAndResolver`, `getSharedImportCache`).
- **`compiler_types.go`** — the `Compiler` struct, `FileCreationTracker` interface, `logTypes`, and `allowedDomain`.

This is a pure mechanical refactor with no behavior change. All symbols remain in the same Go package, so no import paths change.

### Alternatives Considered

#### Alternative 1: Keep Everything in compiler_types.go

Retain the status quo and leave all three lifecycle groups in one file. This avoids any file proliferation and is the zero-effort option.

Rejected because the file already exceeded 400 lines and was on a growth trajectory. Mixing construction-time and runtime-mutation responsibilities in one file makes code review harder and obscures the public API surface for each lifecycle.

#### Alternative 2: Split by Public vs. Private, Not by Lifecycle

Group all public symbols in one file and all private helpers in another, regardless of their lifecycle role.

Rejected because this does not capture the semantically important boundary between construction-time options (only called in `NewCompiler`) and post-construction mutators (called by callers after the compiler is built). The lifecycle split is the conceptually cleaner boundary for navigation and future extension.

### Consequences

#### Positive
- Each file has a single clear responsibility; the correct file to open for any given symbol is immediately obvious from the file name.
- Test files can be co-located with the files they cover (`compiler_options_test.go`, `compiler_mutators_test.go`), keeping tests close to the code they exercise.
- The public mutator API (38 relocated symbols) is now isolated in `compiler_mutators.go`, making it easy to audit what callers can change post-construction.
- Smaller individual files speed up code review of future changes to any one lifecycle group.

#### Negative
- More files to navigate: contributors unfamiliar with the split must learn which file owns which kind of symbol (options vs. mutators vs. types).
- Future additions to `Compiler` require a judgment call about which file to place them in; the lifecycle boundary is not always clear-cut (e.g., `GetVersion` is a read-only accessor placed in `compiler_mutators.go` rather than `compiler_options.go`).

#### Neutral
- No behavior change — this is a pure mechanical refactor. Existing tests pass without modification.
- `pkg/workflow/README.md` symbol table was updated to reference the new file names for the 38 relocated symbols.
- The orphaned doc comment for `SkipIfMatchConfig` (whose type lives in `workflow_data.go`) was removed as part of the cleanup.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
