# ADR-49255: Extract File-Local Helper Functions to Satisfy largefunc Lint Rule

**Date**: 2026-07-31
**Status**: Draft
**Deciders**: Unknown (Copilot SWE agent, pelikhan)

---

### Context

The project enforces a custom `largefunc` lint rule via `make golint-custom` that flags functions exceeding a maximum line-count threshold. Several command-flow functions in `pkg/cli` — including `compileSpecificFiles` (~274 lines), `compileAllFilesInDirectory` (~295 lines), `runCommandForOrg` (~262 lines), `downloadRunArtifacts` (~386 lines), and multiple functions in `update_workflows.go` and `update_extension_check.go` — were flagged as violations. Leaving these violations unresolved blocks the project's lint gate and accumulates technical debt that makes individual code paths harder to read and test in isolation.

### Decision

We will decompose over-sized functions in `pkg/cli` by extracting focused, file-local helper functions (unexported, defined in the same `.go` file). Each extracted helper captures a single named concern (e.g., `compileBatchLoop`, `runBatchSecurityTools`, `checkArtifactsCached`). No behavior, flags, logging, or output formatting are changed — the refactor is purely structural to satisfy the lint threshold.

### Alternatives Considered

#### Alternative 1: Suppress the lint rule per-function with directive comments

Each flagged function could receive a `//nolint:largefunc` (or equivalent) directive to silence the linter without changing code structure. This is faster but does not address the readability and testability problems that motivated the rule in the first place, and it creates precedent for bypassing the lint gate rather than complying with it.

#### Alternative 2: Move related logic into new sub-packages

Related helpers could be promoted to dedicated sub-packages (e.g., `pkg/cli/compileutil`) to enforce stricter separation of concerns and improve reusability. This would be a more significant structural change that could require interface changes, exported types, and updated import paths across the codebase — scope beyond what the lint violations require. A sub-package extraction may be warranted later if these helpers grow, but is premature for the immediate lint compliance goal.

### Consequences

#### Positive
- All flagged `largefunc` violations in the targeted files are resolved, unblocking the lint gate.
- Smaller, named helpers (`checkArtifactsCached`, `runBatchSecurityTools`, etc.) are individually testable and easier to reason about in isolation.
- Shared sub-logic (e.g., `compileBatchLoop` reused by both `compileSpecificFiles` and `compileAllFilesInDirectory`) is deduplicated, reducing maintenance surface.

#### Negative
- File-local helpers are not accessible outside their containing file; any future cross-file reuse requires promotion to exported functions or a separate package.
- A reader following control flow must navigate more call sites; the top-level function bodies become a sequence of helper calls rather than self-contained code, which can obscure the overall pipeline at a glance.

#### Neutral
- All new helpers are unexported and file-local — no public API surface changes.
- The `sort` import is replaced with `slices` (using `slices.SortFunc`) to conform to modern Go idiom; this is a minor dependency change with no behavioral impact.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
