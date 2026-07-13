# ADR-44994: ParseWorkflow Hot-Path Caching via Process-Lifetime Singletons

**Date**: 2026-07-13
**Status**: Draft
**Deciders**: Unknown

---

### Context

`ParseWorkflowFile` regressed from 371μs to 479μs (+29.2%) due to three per-call allocations that were unnecessary for the overwhelmingly common case where workflows have no imported models or frontmatter overrides. Specifically: (1) a 5KB deep copy of the 52-entry builtin model alias map on every parse call, (2) a full 52-node DFS cycle-detection traversal over static builtin aliases that never change, and (3) re-creation of a 14-entry `knownTools` map literal inside `NewTools()` on every invocation. With `ParseWorkflowFile` called on every workflow file load, these allocations accumulate significantly under load.

### Decision

We will eliminate redundant per-call allocations in `ParseWorkflowFile` using three complementary techniques: (1) a process-lifetime singleton for the builtin model alias map, returned directly on the fast path via `getBuiltinOnlyAliasMap()`; (2) a `sync.Once`-cached cycle-detection result for the builtin alias DFS, bypassed via `isBuiltinOnlyAliasMap()` which uses `unsafe.Pointer` map-header comparison for zero-cost identity checks; and (3) promotion of the `knownTools` map to a package-level variable. Template validation functions also gain `strings.Contains` fast-paths to skip regex when no template blocks are present.

### Alternatives Considered

#### Alternative 1: Retain deep copy on every call (status quo)

The original code deep-copied the builtin alias map on every `ParseWorkflowFile` call. This is safe and simple but wastes ~5KB of allocation and the full DFS traversal cost on the majority of calls where no custom aliases are used. Rejected because the regression was measurably impactful and the common case (no overrides) is overwhelmingly dominant.

#### Alternative 2: Use reflect.DeepEqual or content hashing for map identity

Instead of `unsafe.Pointer` header extraction, map identity could be established by hashing map contents or using `reflect.DeepEqual`. This avoids relying on Go runtime internals but is significantly more expensive than a single pointer comparison. Rejected because the purpose is a zero-overhead fast-path guard; a costly identity check defeats the optimization.

#### Alternative 3: Refactor callers to pass an explicit "no custom aliases" flag

Callers could explicitly signal "no custom aliases" and skip alias merging and validation entirely at a higher level. This would be more architecturally clean but requires threading a flag through multiple call sites and changes the API contract of `ParseWorkflowFile`. Rejected as disproportionately invasive relative to the focused caching approach.

### Consequences

#### Positive
- `ParseWorkflow` throughput improves ~50% below the regression baseline and ~35% below the historical baseline (~240,000 ns/op vs. 479,440 regression vs. 371,023 historical).
- Per-call allocations are eliminated for the common case (no imported models, no frontmatter overrides), reducing GC pressure under load.
- The `knownTools` promotion is a straightforward, safe refactor with no behavioral change.
- Template validation fast-paths reduce regex execution on the majority of workflow files that contain no template blocks.

#### Negative
- `isBuiltinOnlyAliasMap()` uses `unsafe.Pointer` to extract the Go map header pointer, relying on a runtime implementation detail (single machine-word map representation). This layout has been stable since Go 1.0 but is not part of the language specification.
- `MergeImportedModelAliases` now returns a shared, read-only map in the common case, introducing a mutation hazard: callers that previously assumed the returned map was freely mutable could silently corrupt shared state. The contract is documented in comments but not enforced by the type system.
- The singleton approach ties correctness to initialization order (`builtinOnlyAliasMapID` must be populated before `isBuiltinOnlyAliasMap` can return `true`), adding a subtle ordering dependency.

#### Neutral
- Two `sync.Once` variables and one `uintptr` are added as package-level state, increasing the package's global footprint slightly.
- `detectCircularModelAliases` is split into an outer dispatcher and `detectCircularModelAliasesUnoptimized`, slightly increasing call-stack depth for non-builtin alias maps.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
