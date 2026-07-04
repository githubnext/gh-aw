# ADR-43135: Extract Runtime Extra-Field Helpers into Shared Functions

**Date**: 2026-07-03
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `generateSetupStep` function in `pkg/workflow/runtime_step_generator.go` contained three separate, nearly-identical blocks of code that each merged `Runtime.ExtraWithFields` with `RuntimeRequirement.ExtraFields` and then emitted the merged map as sorted YAML `with:` entries. These blocks existed for the gh-aw runtime path, the Go `go-version-file` path, and the generic runtime path. Having three diverging copies created risk of inconsistent precedence rules, formatting differences, and non-deterministic output ordering. The lack of shared abstraction also made targeted test coverage difficult.

### Decision

We will extract two package-level helper functions—`mergeRuntimeWithFields(req *RuntimeRequirement) map[string]string` and `appendSortedWithFieldEntries(step GitHubActionStep, withFields map[string]string) GitHubActionStep`—into `runtime_step_generator.go` and replace all three duplicated inline blocks with calls to these helpers. User-provided `req.ExtraFields` continue to take precedence over runtime-default `ExtraWithFields`; output key order remains stable via sorted iteration.

### Alternatives Considered

#### Alternative 1: Keep Duplication, Add Code-Review Guard

The three copies could be kept with a comment or code-ownership annotation requiring simultaneous edits. This avoids new abstractions but provides no compile-time guarantee that the paths stay in sync, and enforcement would require vigilance rather than structure.

#### Alternative 2: Struct Method on RuntimeRequirement

The merge logic could become a method `(r *RuntimeRequirement) MergedWithFields() map[string]string` directly on `RuntimeRequirement`. This makes the dependency explicit at the call site. However, the sort-and-append step operates on an arbitrary `map[string]string`, not just on `RuntimeRequirement`, so a method would only cover the merge half; the append helper would remain a package-level function regardless.

### Consequences

#### Positive
- Single source of truth for extra-field merge semantics eliminates drift risk across the three setup code paths.
- Deterministic YAML output (stable sorted key order) is now enforced in one place and directly covered by focused tests.

#### Negative
- The helper functions are unexported, so they can only be exercised through the existing `GenerateRuntimeSetupSteps` public surface; there are no direct unit tests for the helpers in isolation.
- Adds a new abstraction layer: developers debugging setup-step generation must now trace into helper calls even though the per-hop logic is trivial.

#### Neutral
- The `sliceutil` import moves from `gh_aw_setup_steps.go` to `runtime_step_generator.go`, reflecting where the sort-and-append logic now lives.
- Three focused tests were added to lock in override and ordering behavior for all three setup paths (gh-aw, generic runtime, and Go go-version-file).

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
