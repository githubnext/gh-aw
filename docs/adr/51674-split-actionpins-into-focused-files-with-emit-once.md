# ADR-51674: Split `pkg/actionpins` into Focused Files and Centralize Warning Deduplication

**Date**: 2026-08-10
**Status**: Draft
**Deciders**: Unknown

---

### Context

`pkg/actionpins/actionpins.go` (595 lines) combined unrelated concerns in a single file: type definitions, embedded JSON data loading, pin resolution logic, reference formatting, action/container mappings, and warning deduplication. Warning deduplication was duplicated inline across multiple functions using the pattern `initWarnings(ctx)` followed by a manual `if !ctx.Warnings[key]` check and a bare `fmt.Fprintln(os.Stderr, ...)` call, making it easy for new callers to skip deduplication accidentally. Similarly, `pkg/agentdrain` had `FlattenEvent`/`Tokenize`/`StageSequence` helpers living in `mask.go` (unrelated to masking) and `extractParams` in `miner.go` (unrelated to cluster mining).

### Decision

We will split `pkg/actionpins/actionpins.go` into six focused files (`types.go`, `data.go`, `references.go`, `resolve.go`, `mappings.go`, `warnings.go`) and centralize all deduped stderr output through a new `PinContext.emitOnce(key, msg, formatFn)` method. We will also relocate `FlattenEvent`, `Tokenize`, and `StageSequence` into a new `pkg/agentdrain/event.go`, and move `extractParams` to `cluster.go` alongside the template/merge logic it supports. All public APIs and existing pin-resolution behavior are preserved unchanged.

### Alternatives Considered

#### Alternative 1: Improve Internal Organization with Comments/Sections in a Single File

Keep everything in `actionpins.go` but add section dividers and regional comments to separate concerns. This approach has near-zero migration cost and avoids readers needing to jump across files. It was rejected because it does not enforce single-responsibility at a reviewable level, does not eliminate the repeated `initWarnings` + inline-guard boilerplate, and would still leave the file at 595+ lines with all concerns entangled.

#### Alternative 2: Extract Concerns into Sub-packages

Create sub-packages such as `pkg/actionpins/resolution`, `pkg/actionpins/pindata`, and `pkg/actionpins/mappings`. This would enforce separation via Go's import graph and make dependencies explicit. It was rejected because `pkg/actionpins` is already deliberately free of external package dependencies to avoid import cycles, splitting into sub-packages adds cross-package imports where the same concern must access shared types, and Go convention for packages of this size favors file-level decomposition within a single package rather than a sub-package hierarchy.

### Consequences

#### Positive
- Each file has a single, clearly named responsibility, reducing cognitive load when navigating or reviewing changes.
- `PinContext.emitOnce` is the single, canonical path for deduped stderr output; new callers cannot accidentally skip deduplication.
- Event-related helpers (`FlattenEvent`, `Tokenize`, `StageSequence`) now live in `event.go`, co-located with their domain, reducing misattribution when reading `mask.go` or `coordinator.go`.
- `extractParams` lives beside `mergeTemplate` in `cluster.go`, where it is logically coupled.

#### Negative
- Developers unfamiliar with the new layout must learn which file to open; the old single-file approach had all code in one scroll.
- Reviewing changes that span multiple concerns now requires opening multiple files instead of one.

#### Neutral
- No public API surface change; all callers import the same `pkg/actionpins` package path.
- Test coverage moves alongside production code; existing tests continue to pass without modification to test logic.
- The `Warnings map[string]bool` field on `PinContext` remains the backing store for `emitOnce`; callers that inspect it directly continue to work.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
