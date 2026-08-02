# ADR-49863: Extend Activation Job Dependency Resolution to Include engine.env Job References

**Date**: 2026-08-02
**Status**: Draft
**Deciders**: Unknown

---

### Context

The workflow compiler builds a GitHub Actions workflow with a structured job execution order. The `activation` job is a core system job that must run after any custom jobs whose outputs it depends on. The compiler previously resolved activation job dependencies by scanning the markdown prompt body for `needs.<job>.outputs.*` expressions. However, users can also reference custom job outputs in `engine.env` values (e.g., to override `COPILOT_GITHUB_TOKEN` with a dynamically-generated token from a custom job). Without scanning `engine.env`, the activation job would not list those custom jobs as `needs`, causing a race condition where activation starts before the custom job's output is available. Additionally, if the compiler auto-added `activation` as a dependency to those same custom jobs, it would create a circular dependency: `activation → custom_job → activation`.

### Decision

We will extend the compiler's activation job dependency resolution to scan `engine.env` values for `needs.<job>.outputs.*` expressions, using the same "no explicit needs" filter already applied to prompt-body-referenced jobs. A new helper function `getEngineEnvReferencedCustomJobsWithNoExplicitNeeds` encapsulates this logic and is called from both `configureActivationNeedsAndCondition` (to add jobs to the activation `needs` list) and `getCustomJobDependencySets` (to mark those jobs as pre-activation, preventing the compiler from auto-adding `activation` as their dependency and creating a cycle).

### Alternatives Considered

#### Alternative 1: Require users to declare explicit `needs` in custom jobs that provide engine.env outputs

Users whose custom token-provider job is referenced in `engine.env` could explicitly declare `needs: []` or `needs: pre_activation` in their job definition, which would already route them through the existing `getCustomJobsDependingOnPreActivation` path. This avoids any compiler changes.

This was not chosen because it places an invisible, non-obvious burden on users. The `engine.env` expression syntax already expresses the dependency — requiring a separate `needs` declaration is redundant and error-prone. The compiler should infer this automatically from the expression, consistent with how it already handles prompt-body references.

#### Alternative 2: Unify engine.env scanning into the existing `getCustomJobsReferencedInPromptWithNoActivationDep` function

Instead of adding a separate helper, the existing function that scans the markdown body could be extended to also scan `engine.env` values in a single combined pass.

This was not chosen because the two sources — markdown prompt body and `engine.env` — are semantically distinct and have different data access paths. A single function conflating both would be harder to test and reason about. Separate, focused helpers match the existing function-per-source pattern in the codebase and make each concern independently testable.

### Consequences

#### Positive
- Custom jobs referenced in `engine.env` via `needs.<job>.outputs.*` are now automatically added to the activation job's `needs` list, eliminating the race condition without requiring user-level workarounds.
- The circular dependency that would arise from the compiler auto-adding `activation` as a prerequisite of those same custom jobs is prevented by updating `getCustomJobDependencySets` alongside `configureActivationNeedsAndCondition`.
- The fix is consistent with the existing pattern for prompt-body job references, so the mental model for developers maintaining the compiler remains uniform.

#### Negative
- `getEngineEnvReferencedCustomJobsWithNoExplicitNeeds` is called independently from two separate code paths, so `engine.env` is parsed twice per compilation. For typical workflow sizes this is negligible, but it is a redundant computation.
- The new helper is a parallel implementation of part of the existing `getCustomJobsReferencedInPromptWithNoActivationDep` logic, increasing the surface area that must be kept consistent if the "no explicit needs" filter semantics change in the future.

#### Neutral
- Unit tests added in `compiler_activation_outputs_test.go` cover the new behavior with seven test cases (direct reference, `case()` expression, pre_activation presence, no-reference passthrough, built-in job exclusion, nil safety, deduplication). These tests run in the `!integration` build tag.
- The change is a follow-up to PR #30239, which established the same pattern for prompt-body references.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
