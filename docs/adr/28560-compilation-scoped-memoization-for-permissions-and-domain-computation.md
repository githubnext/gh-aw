# ADR-28560: Compilation-Scoped Memoization for Permissions and Domain Computation

**Date**: 2026-04-26
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `WorkflowData` struct is the central data bag threaded through the compiler pipeline. During a single compilation run, several expensive operations — notably YAML deserialization via `goccy/go-yaml.Unmarshal` (inside `NewPermissionsParser`) and domain-list construction (inside `mergeDomainsWithNetworkToolsAndRuntimes`) — were being called redundantly by independent compilation steps that each needed the same derived value. Benchmarking revealed a 21% regression in `BenchmarkCompileSimpleWorkflow` (2.76 ms → 3.34 ms), with profiling attributing ~7% CPU to repeated `NewPermissionsParser` calls and ~4% to repeated domain computation. Because `WorkflowData` is not mutated between these call sites during compilation, the derived values are stable across the run.

### Decision

We will add lazy memoization fields (`CachedPermissions *Permissions` and `CachedAllowedDomainsStr string`) to `WorkflowData`. The first caller in the compilation pipeline that needs a parsed permissions object or a domain string computes it and stores it on the struct; all subsequent callers read the cached value. `applyDefaults` is the designated site for populating `CachedPermissions` since it already performs all permission mutations. `computeAllowedDomainsForSanitization` self-populates `CachedAllowedDomainsStr` on first call and returns it directly thereafter.

### Alternatives Considered

#### Alternative 1: Eager Pre-Computation of All Derived Fields in `applyDefaults`

Compute parsed permissions and the domain string unconditionally during `applyDefaults`, regardless of whether any later step actually needs them. This is simpler than lazy caching because there is no conditional branch at each call site. It was not chosen because it unconditionally pays the cost of YAML parsing and domain computation even for compilation paths (e.g., minimal workflows) that never reach the affected steps, and it requires `applyDefaults` to know about domain computation logic that is otherwise encapsulated in the compiler.

#### Alternative 2: Thread Computed Values Through Call-Site Parameters

Pass the pre-computed `*Permissions` and domain string as explicit function parameters to every function that needs them, rather than storing them on `WorkflowData`. This avoids adding mutable state to the struct. It was not chosen because it requires cascading signature changes across many callers (five call sites for permissions alone), increases call-site cognitive overhead, and does not naturally compose with the existing variadic-fallback pattern already used in `filterJobLevelPermissions`. The mutable-field approach is more incremental and confined to `WorkflowData`, which is already the designated carrier for compilation-scoped state.

### Consequences

#### Positive
- Benchmark shows −8.4% ns/op, −3.8% B/op, and −14.8% allocs/op on `BenchmarkCompileSimpleWorkflow`.
- No behavioral changes: all cached values are pure functions of `WorkflowData` fields that are frozen before the caching call sites are reached.
- The variadic `cachedPerms ...*Permissions` signature for `filterJobLevelPermissions` provides a zero-friction migration path: callers without a cached value continue to work unchanged.

#### Negative
- `WorkflowData` gains mutable cache fields, introducing an implicit requirement that `CachedPermissions` is populated by `applyDefaults` before any downstream compiler step reads it. Violating this ordering silently falls back to re-parsing rather than failing loudly.
- `CachedAllowedDomainsStr` uses an empty-string sentinel to detect "not yet computed," which means an actual empty domain list (theoretically possible in an edge case) would be recomputed on every call instead of being cached.
- New cache fields must be considered whenever `WorkflowData` is shallow-copied or reset between compilation phases.

#### Neutral
- The `Permissions.HasContentsReadAccess()` method added to `permissions_operations.go` duplicates logic already present in `PermissionsParser.HasContentsReadAccess()`. Both must be kept in sync when permission semantics change.
- New test coverage for `HasContentsReadAccess` and `filterJobLevelPermissions` with a cache argument is added in `permissions_operations_test.go`.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Cache Population

1. `applyDefaults` **MUST** populate `WorkflowData.CachedPermissions` after all permission mutation steps are complete, so that downstream compiler steps can rely on the cached value being present.
2. Compiler steps that need a parsed `*Permissions` object **MUST** prefer `data.CachedPermissions` over calling `NewPermissionsParser(data.Permissions)` when `data.CachedPermissions` is non-nil.
3. `computeAllowedDomainsForSanitization` **MUST** store its result in `data.CachedAllowedDomainsStr` on first computation and **MUST** return that cached value on all subsequent calls within the same compilation.

### Cache Invalidation

1. Cache fields **MUST NOT** be selectively cleared or overwritten by any compiler step after `applyDefaults` has run, as the cached values are assumed to be stable for the lifetime of the compilation.
2. If `WorkflowData` is copied for a new compilation pass, the copy **SHOULD** clear `CachedPermissions` and `CachedAllowedDomainsStr` to avoid propagating stale values.

### Method Parity

1. `Permissions.HasContentsReadAccess()` **MUST** return the same result as `PermissionsParser.HasContentsReadAccess()` for any given permissions configuration.
2. When permission semantics change (e.g., new shorthand values), both implementations **MUST** be updated together.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically: (1) `applyDefaults` populates `CachedPermissions` before any caching call site executes; (2) all identified call sites prefer the cached value when available; (3) `computeAllowedDomainsForSanitization` self-memoizes its result on `WorkflowData`; and (4) `HasContentsReadAccess` on `Permissions` and on `PermissionsParser` remain behaviourally equivalent.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24954049328) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
