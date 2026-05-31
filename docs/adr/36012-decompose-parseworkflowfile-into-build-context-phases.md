# ADR-36012: Decompose ParseWorkflowFile Orchestration into Build-Context Phases

**Date**: 2026-05-31
**Status**: Accepted — `ParseWorkflowFile` now runs as a four-phase build-context pipeline.
**Deciders**: pelikhan, Copilot

## Context

`ParseWorkflowFile` in `pkg/workflow/compiler_orchestrator_workflow.go` is the central orchestration entry point that coordinates every compilation phase: frontmatter parsing, engine/imports setup, tools and markdown processing, a long sequence of validations, and the population/merge of imported configuration. Over time it grew into a single ~330-line function that threaded many local variables (`cleanPath`, `content`, `frontmatter`, `markdownDir`, `engineSetup`, `toolsResult`, `workflowData`) through inline blocks and carried dense, deeply nested merge logic (observability/OTLP endpoint dedupe, env source attribution) directly in the body. A custom large-function lint pass flagged this method as exceeding the project's complexity threshold. The change must reduce that complexity **without altering compiler behavior** — validation order, merge precedence, and the exact formatted-error output must remain identical.

## Decision

We will decompose `ParseWorkflowFile` into four explicit, sequentially-invoked phases coordinated through a shared `workflowBuildContext` struct that carries phase state instead of long parameter lists:

1. **Type gating** — `validateParseResultWorkflowType` rejects shared and redirect-only workflows.
2. **Setup** — `setupWorkflowBuildContext` runs engine/imports setup, tools/markdown processing, and builds the initial `WorkflowData`.
3. **Validation** — `validateWorkflowBuildContext` runs model-alias, engine-setting, and tool-policy checks.
4. **Population** — `populateWorkflowBuildContext` extracts YAML sections and merges imported configuration (observability, env, features, steps, services, on-fields).

Dense inline logic is extracted into focused, behavior-preserving helpers (e.g. `mergeImportedObservability`, `mergeRawOTLPEndpoints`, `applyMergedRawObservability`, `mergeWorkflowEnv`, `buildMergedEnvSources`), and the two error-wrapping paths are isolated into `formatEngineSetupError` / `formatToolsProcessingError` so that already-formatted compiler errors are never double-wrapped. The top-level function becomes a short, readable pipeline of the four phase calls.

## Alternatives Considered

### Alternative 1: Keep the monolithic function and suppress the lint warning
We could annotate `ParseWorkflowFile` to exempt it from the large-function lint rule and leave the structure as-is. This avoids churn and the risk of subtly reordering validations, but it permanently institutionalizes a function that is hard to read, hard to test in isolation, and that new contributors must hold entirely in their head. It also undermines the lint rule the team chose to adopt. Rejected because it trades a one-time refactoring cost for ongoing maintenance drag.

### Alternative 2: Split into free functions that thread explicit parameters
We could extract the same phases as standalone functions that each receive the seven-plus loose values as explicit arguments rather than a shared context struct. This keeps state immutable and dependencies visible at each call site, but it produces unwieldy signatures (already the original pain point) and forces every new piece of phase state to ripple through multiple signatures. Rejected in favor of a context struct that localizes the threading cost; it was a close call on the immutability/clarity trade-off.

## Consequences

### Positive
- Each phase is independently readable and unit-testable, and the top-level function now reads as a four-step pipeline.
- Per-function cyclomatic complexity drops below the lint threshold, satisfying the large-function pass.
- Error-formatting and merge logic are named and reusable, reducing the chance of inconsistent error wrapping.
- Helper-level tests now lock in validation ordering, OTLP endpoint de-duplication/counting, and env source attribution so future cleanups can preserve those contracts.

### Negative
- More indirection: a reader tracing one validation must now navigate across several helper functions rather than reading top-to-bottom.
- `workflowBuildContext` is a mutable struct shared and progressively filled across phases, introducing ordering coupling — a phase that runs out of order would observe nil fields.

### Neutral
- The change is behavior-preserving by design; correctness now depends on the existing compiler tests plus focused helper tests covering validation order, merge precedence, and env-source attribution.
- The `workflowBuildContext` pattern establishes a template that future orchestration refactors in this package are likely to follow (cf. ADR-35812).
