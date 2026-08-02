# ADR-49814: Support `jobs.agent.if` for First-Class Agent Job Conditional Gating

**Date**: 2026-08-02
**Status**: Draft
**Deciders**: Unknown (copilot-swe-agent, pelikhan)

---

### Context

The gh-aw workflow compiler generates a set of built-in jobs (`agent`, `activation`, `pre_activation`, etc.) from frontmatter configuration. Prior to this change, workflow authors who needed to gate the generated agent job on a custom setup job's output (e.g., "only run the agent if the build step failed") had to route control flow through the top-level workflow `if:` or the `on.needs` + cascade pattern. This workaround had a critical limitation: it prevented referencing `needs.<job>.outputs.*` from within the agent job, since those references are only valid when the job itself declares a `needs` dependency on the upstream job.

The compiler already supported additive `needs` via `jobs.<built-in>.needs`, but lacked the corresponding support for `jobs.<built-in>.if`, leaving agent-level conditional gating as a second-class pattern.

### Decision

We will extend the compiler's built-in job augmentation system to also accept a `jobs.<built-in>.if` field. When present, the user-supplied condition string is combined with any compiler-generated `if` condition on that job using logical `&&`. This is applied during the same `applyBuiltinJobNeedsAugmentations` pass that handles additive `needs`, keeping both augmentation types co-located and consistent.

### Alternatives Considered

#### Alternative 1: Continue using top-level workflow `if` + `on.needs` cascade

The existing workaround required authors to add the conditional at the workflow level rather than the job level. This approach was rejected because `needs.<job>.outputs.*` expressions are only valid inside a job that declares an explicit `needs` on the upstream job; a top-level workflow `if` cannot reference output values that require a job-level dependency.

#### Alternative 2: Introduce a dedicated `pre_activation` / `activation` hook for conditional logic

The compiler already exposes `pre_activation` and `activation` as extensible hooks. Authors could theoretically wire conditional execution through those layers. This was rejected because it adds indirection, requires deeper knowledge of internal compiler phases, and is more complex than a direct `jobs.agent.if` field that mirrors the GitHub Actions native syntax.

### Consequences

#### Positive
- Workflow authors can express agent-level gating directly in `jobs.agent.if`, using standard GitHub Actions expression syntax and `needs.<job>.outputs.*` references.
- The feature is consistent with the existing `jobs.<built-in>.needs` augmentation contract: additive, non-destructive, and transparent to compiler-managed behavior.
- The API surface matches the mental model of GitHub Actions authors who already know how `jobs.<job>.if` works natively.

#### Negative
- The compiler now owns condition-merging logic (`combineJobIfConditions`), which must correctly handle precedence, parenthesization, and expression stripping for all current and future compiler-generated conditions — increasing compiler complexity.
- The merged `if` value is only visible in the compiled `.lock.yml`, not in the source frontmatter, which may surprise authors debugging unexpected skip behavior.

#### Neutral
- The `if` augmentation is validated at compile time (non-string values produce an error), which is consistent with how `needs` augmentation is validated.
- Existing frontmatter that does not use `jobs.agent.if` is unaffected; the feature is purely additive.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
