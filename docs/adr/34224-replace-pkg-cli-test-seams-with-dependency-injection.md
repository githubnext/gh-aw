# ADR-34224: Replace mutable pkg/cli test seams with per-flow dependency injection

**Date**: 2026-05-23
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

`pkg/cli` exposed seven package-level mutable function variables (e.g., `getLatestActionReleaseFn`, `runGHReleasesAPIFn`, `getActionSHAForTagFn`, `getLatestActionReleaseViaGitFn`, `getReleasePublishedAtFn`, `runWorkflowReleasesAPIFn`, `resolveCurrentRepoDefaultBranchFn`) used purely as test seams. Tests had to snapshot each variable, mutate it for the duration of the test, and restore the original in a `defer`/`t.Cleanup`. This pattern blocked safe `t.Parallel()` usage because any two tests touching the same global would race, forced repetitive restore boilerplate, and obscured which collaborators a given flow actually depends on. The CLI flows affected (`update_actions`, `update_workflows`, `update_cooldown`, `codemod_workflow_run_branches`) are pure logic flows where the seams are network and process boundaries, so they are natural candidates for explicit dependency wiring.

### Decision

We will replace the package-level mutable function variables with per-flow dependency-injection structs. Each affected flow defines a `<flow>Deps` struct that groups its injectable collaborators, a `default<Flow>Deps()` constructor returning the production wiring, and an internal `<entrypoint>WithDeps(...)` (or equivalent `internal<entrypoint>(...)`) helper that accepts the struct. Exported entrypoints keep their existing signatures and delegate to the default deps so external callers are unaffected. Tests construct a default deps value and override only the fields they need, enabling `t.Parallel()` and eliminating restore boilerplate.

### Alternatives Considered

#### Alternative 1: Keep package-level function vars and rely on `t.Cleanup`

Continue using mutable globals but standardise on `t.Cleanup` to restore originals. This is the lowest-effort change and preserves the existing public surface. It was rejected because it does not solve the core problem: globals remain shared mutable state across goroutines, so `t.Parallel()` is still unsafe for any test touching a seam, and the restore boilerplate still has to live in every test.

#### Alternative 2: Singleton/registry-based dependency container

Introduce a `cli` package-level registry (or context-scoped service locator) where collaborators are registered and looked up by interface. This would centralise wiring but adds indirection, hides dependencies behind a registry, and reintroduces shared mutable state if the registry is package-scoped. It was rejected as over-engineered for the small, fixed set of injection points and inconsistent with idiomatic Go.

#### Alternative 3: Per-flow dependency structs (chosen)

Each flow declares its own small struct of function-typed fields, constructed once per call. Dependencies are explicit at the call site, tests build their own deps locally, and there is no shared mutable state. Slight verbosity at the public/internal boundary is the trade-off, but it scales linearly with the number of seams and stays close to standard Go practice.

### Consequences

#### Positive

- Tests in the refactored flows can use `t.Parallel()` without races, shortening the test suite wall-clock time.
- Each flow's external dependencies are visible at a glance via its `<flow>Deps` struct, improving readability and onboarding.
- Test boilerplate is reduced: no `orig := fn; defer func() { fn = orig }()` pattern; tests construct a local deps value and override fields directly.
- The pattern is consistent across all four refactored flows, making future additions predictable.

#### Negative

- Every public entrypoint now has a paired internal `WithDeps` (or unexported equivalent) helper, roughly doubling the number of functions to maintain at each boundary.
- The deps struct must be threaded through internal call chains (e.g., `updateActions → updateActionsInWorkflowFiles → updateActionRefsInContentWithDeps`), so adding a new collaborator means touching multiple signatures.
- Slight call-site verbosity: production code now passes `defaultXxxDeps()` explicitly at internal boundaries.

#### Neutral

- The public API of `pkg/cli` is unchanged: exported function signatures (e.g., `UpdateActions`, `UpdateActionsInWorkflowFiles`, `resolveLatestRelease`, `checkReleaseCoolDown`, `getWorkflowRunBranchesCodemod`) are preserved and delegate to the default deps.
- The pattern is currently applied only to the four flows in this PR; other `pkg/cli` flows still use package-level seams and may follow incrementally.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Dependency Injection Pattern in `pkg/cli`

1. New flows in `pkg/cli` that require injectable collaborators for testing **MUST NOT** introduce package-level mutable function variables (e.g., `var xxxFn = ...`) as test seams.
2. New flows requiring injectable collaborators **MUST** define a per-flow dependency struct (e.g., `type <flow>Deps struct { ... }`) whose fields are function types representing each injectable collaborator.
3. Each per-flow dependency struct **MUST** be paired with a `default<Flow>Deps()` constructor returning the production wiring.
4. Exported entrypoints for a flow **SHOULD** preserve their existing signatures and delegate to an internal `<entrypoint>WithDeps` (or equivalent unexported) helper that accepts the dependency struct as a parameter.
5. Internal helpers within a flow **SHOULD** accept the dependency struct as a parameter rather than constructing it themselves, so collaborators are wired exactly once per public call.
6. Tests in `pkg/cli` **SHOULD** construct a default deps value (via the `default<Flow>Deps()` constructor) and override only the fields whose behaviour is under test, rather than mutating package-level state.
7. Tests in `pkg/cli` that previously relied on package-level seam mutation **SHOULD** be migrated to the dependency-injection pattern when the surrounding flow is refactored, and **SHOULD** add `t.Parallel()` where the test no longer depends on shared mutable state.
8. Existing package-level mutable function variables outside the four flows refactored by this ADR **MAY** remain until those flows are migrated; new code **MUST NOT** add to them.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
