# ADR-53278: Decompose Safe-Outputs Config Pipeline Large Functions into Focused Helpers

**Date**: 2026-08-17
**Status**: Draft
**Deciders**: pelikhan (copilot-swe-agent, PR author)

---

### Context

The safe-outputs config pipeline in `pkg/workflow/` contained five functions that had grown to hundreds of lines each:
`extractSafeOutputsConfig`, `extractGlobalConfigFields`, `validateSafeOutputsMax`, `generateSafeOutputsConfig`, and `computeEnabledToolNames`.
These functions each mixed concerns from multiple semantic domains — core handlers, fallback handlers, max validation, config generation, and enabled tool computation — inside a single flat body with inline nil-checks and comment delimiters.
The `golint-custom` linter flagged all five as large-function hotspots (issue #53269).
The sole engineering constraint was structural: reduce function size while preserving all existing parsing, defaulting, and interface semantics with zero behavioral change.

### Decision

We will decompose each large function into a set of domain-grouped, focused helper functions that each handle one semantic concern: extraction of core handlers, review/security handlers, issue/PR mutation handlers, workflow dispatch handlers, fallback handlers, and global config sub-domains (domain/reference, execution, messaging, failure-reporting, runtime/dependency, environment/custom).
For `validateSafeOutputsMax` we introduce a `maxFieldCheck` struct and grouped `appendChecksGroupN` helpers; for `generateSafeOutputsConfig` we introduce standalone `addXxx` builders per config category.
All existing public interfaces, struct fields, and defaulting behavior are preserved exactly.

### Alternatives Considered

#### Alternative 1: Keep monolithic functions

Retain the existing large function bodies unchanged.
Simple call graph, no additional indirection, no risk of grouping mismatch.
Rejected because the linter violations remain unresolved and the functions continue to accumulate new handlers unboundedly; any further feature addition makes them harder to read and review.

#### Alternative 2: Table-driven registry for handler dispatch

Represent each safe-output handler as a registry entry `{name, builder, configField}` and iterate once over the registry for both extraction and validation.
More data-driven and naturally extensible — adding a new handler requires only a single registry line rather than edits to two or three functions.
Rejected for this PR because it requires new generic types or a large interface change to accommodate the varying config field types across handlers, and the PR scope was intentionally limited to structural cleanup; the registry approach is a valid future step.

### Consequences

#### Positive
- Individual helper functions are small enough (< 40 lines each) to read and reason about in isolation; the `largefunc` lint violations are resolved.
- Each semantic group (`extractCoreSafeOutputHandlers`, `extractReviewAndSecurityHandlers`, `extractIssueAndPRMutationHandlers`, etc.) is independently testable and reviewable without touching unrelated code.
- Future additions to one handler group require edits only to the relevant helper, reducing the risk of accidental cross-group regressions.

#### Negative
- The call stack is deeper: callers must navigate through `extractSafeOutputsConfig → extractAdditionalSafeOutputHandlers → extractReviewAndSecurityHandlers` instead of reading one flat body; this can make single-step debugging harder.
- The group boundaries (`appendChecksGroup1–5`, etc.) are somewhat arbitrary and may need reorganization again as new handler types are introduced; the groupings carry no semantic invariant beyond lint-size constraints.

#### Neutral
- All existing tests continue to exercise the same public interfaces; no behavioral change was introduced.
- The `computeEnabledToolNames` declarative check list approach (`predefinedToolChecks` + shared `addEnabledTools` helper) is a partial step toward the table-driven registry described in Alternative 2.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
