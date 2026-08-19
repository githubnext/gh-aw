# ADR-53838: Registry-Based Repo Target Accessor Pattern

**Date**: 2026-08-19
**Status**: Draft
**Deciders**: Unknown

---

### Context

`pkg/workflow/safe_outputs_tools_repo_params.go` contained a single function (`addRepoParameterIfNeeded`) with a 28-case `switch` statement that duplicated the same `AllowedRepos` / `TargetRepoSlug` extraction pattern for every safe-output tool. Adding a new tool required inserting identical boilerplate into multiple branches of that switch with no compile-time or test-time guard against omissions, making the code easy to implement inconsistently.

### Decision

We will replace the monolithic switch with a `repoTargetAccessors` registry: a package-level `map[string]repoTargetAccessor` that maps each tool name to a one-liner function returning a `*repoTargetConfig`. `addRepoParameterIfNeeded` performs a single map lookup and delegates to the accessor, reducing the function from ~160 lines to ~10. A dedicated test (`TestRepoTargetAccessorsCoverRepoTargetTools`) asserts that the registry contains exactly the expected set of tool names, acting as a coverage guard for future additions.

### Alternatives Considered

#### Alternative 1: Keep the switch with a coverage test

Add the `TestRepoTargetAccessorsCoverRepoTargetTools`-style test against the existing switch (e.g., by enumerating all `case` labels via reflection or a maintained list), leaving the implementation unchanged. This would surface omissions at test time but would not reduce the per-tool duplication or shorten `addRepoParameterIfNeeded`.

#### Alternative 2: Interface-based tool registry

Define a `RepoTargetProvider` interface and have each tool's config struct implement it, removing the accessor functions entirely. This would eliminate all per-tool boilerplate in the registry at the cost of touching every config struct and coupling the config layer to the repo-parameter generation logic.

### Consequences

#### Positive
- Adding a new tool now requires a single, uniform registry entry instead of a switch case in an already-large function.
- The coverage test fails fast if a tool is added to the tool list but omitted from the registry.
- `addRepoParameterIfNeeded` is reduced from ~160 lines to ~10, making it easy to understand at a glance.

#### Negative
- Each of the 28 registry entries still contains near-identical boilerplate (`if output := config.X; output != nil { return &repoTargetConfig{...} }`), so per-entry verbosity is unchanged.
- The registry is a package-level `var`, which is initialized at program startup; errors in registry construction surface only at runtime, not at compile time.

#### Neutral
- The `repoTargetConfig` struct is now the canonical representation of per-tool repo-target state, which is a minor new abstraction other callers could reuse.
- The test coverage guard encodes the full list of repo-target tools in the test file; this list must be kept in sync when tools are added or removed.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
