# ADR-49814: Support `jobs.agent.if` for First-Class Agent Job Conditional Gating

**Date**: 2026-08-02
**Status**: Accepted

---

### Context

The gh-aw workflow compiler generates a set of built-in jobs (`agent`, `activation`, `pre_activation`, etc.) from frontmatter configuration. Prior to this change, workflow authors who needed to gate the generated agent job on a custom setup job's output (e.g., "only run the agent if the build step failed") had to route control flow through the top-level workflow `if:` or the `on.needs` + cascade pattern. This workaround had a critical limitation: it prevented referencing `needs.<job>.outputs.*` from within the agent job, since those references are only valid when the job itself declares a `needs` dependency on the upstream job.

The compiler already supported additive `needs` via `jobs.<built-in>.needs`, but lacked the corresponding support for `jobs.<built-in>.if`, leaving agent-level conditional gating as a second-class pattern.

### Decision

We extend the compiler's built-in job augmentation system to also accept a `jobs.<built-in>.if` field. When present, the user-supplied condition string is combined with any compiler-generated `if` condition on that job using logical `&&`. This is applied during the same `applyBuiltinJobNeedsAugmentations` pass that handles additive `needs`, keeping both augmentation types co-located and consistent.

Two additional correctness invariants are maintained:

1. **Prefix normalization**: The `if:` field value is normalized to a bare expression (the `"if: "` prefix is stripped) before storage, consistent with how the `Job.If` field is used throughout the compiler.

2. **Status function bypass guard**: GitHub Actions removes the implicit `success()` gate from all `needs` entries when any status check function (`always`, `failure`, `cancelled`, `success`) appears in a job's `if` expression. Since compiler-owned prerequisites such as `activation` perform permission and security checks, the compiler materializes explicit `needs.<compiler-job>.result == 'success'` guards whenever the user-supplied condition contains a status function. This preserves the activation safety contract regardless of the user's gating expression. User-supplied `needs` are intentionally excluded from the guard: authors choose their own result semantics for setup jobs they own.

### Alternatives Considered

#### Alternative 1: Continue using top-level workflow `if` + `on.needs` cascade

The existing workaround required authors to add the conditional at the workflow level rather than the job level. This approach was rejected because `needs.<job>.outputs.*` expressions are only valid inside a job that declares an explicit `needs` on the upstream job; a top-level workflow `if` cannot reference output values that require a job-level dependency.

#### Alternative 2: Introduce a dedicated `pre_activation` / `activation` hook for conditional logic

The compiler already exposes `pre_activation` and `activation` as extensible hooks. Authors could theoretically wire conditional execution through those layers. This was rejected because it adds indirection, requires deeper knowledge of internal compiler phases, and is more complex than a direct `jobs.agent.if` field that mirrors the GitHub Actions native syntax.

#### Alternative 3: Disallow status functions in user-supplied conditions

Refusing `always()` / `failure()` at compile time would prevent the bypass entirely, but it would also prevent legitimate use cases like "run agent cleanup after a failed setup job." The explicit guard approach is less restrictive while preserving the activation safety invariant.

### Consequences

#### Positive
- Workflow authors can express agent-level gating directly in `jobs.agent.if`, using standard GitHub Actions expression syntax and `needs.<job>.outputs.*` references.
- The feature is consistent with the existing `jobs.<built-in>.needs` augmentation contract: additive, non-destructive, and transparent to compiler-managed behavior.
- The API surface matches the mental model of GitHub Actions authors who already know how `jobs.<job>.if` works natively.
- Status function bypass of compiler-owned prerequisites is prevented automatically without restricting the expressive power of user conditions.

#### Negative
- The compiler now owns condition-merging logic (`combineJobIfConditions`, `guardIfAgainstStatusFuncBypass`), which must correctly handle precedence, parenthesization, expression stripping, and status function detection — increasing compiler complexity.
- The merged `if` value is only visible in the compiled `.lock.yml`, not in the source frontmatter, which may surprise authors debugging unexpected skip behavior.
- The status function guard uses string-based detection (substring search for `always(`, `failure(`, etc.) rather than AST-level detection, because the expression parser represents these calls as opaque `ExpressionNode` literals. This may produce false positives for hypothetical constructs like `notfailure(...)`, which are not valid in GitHub Actions but are theoretically possible in future expression language extensions.

#### Neutral
- The `if` augmentation is validated at compile time (non-string values produce an error), which is consistent with how `needs` augmentation is validated.
- Existing frontmatter that does not use `jobs.agent.if` is unaffected; the feature is purely additive.
- The error reported when a built-in job is not generated correctly names the configured field (`.if` or `.needs`) to aid diagnostics.

