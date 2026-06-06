# ADR-37339: Decompose activation-job builder into single-responsibility helpers

**Date**: 2026-06-06
**Status**: Draft

## Context

Daily `largefunc` lint tracking flagged widespread violations (functions exceeding 60 lines) concentrated in `pkg/workflow`. In `compiler_activation_job_builder.go`, four methods had grown into long, multi-concern procedures: `newActivationJobBuildContext` (context init + pre-step caching + setup/workflow_call assembly + engine wiring), `addActivationFeedbackAndValidationSteps` (app-token minting + reactions + secret validation + cross-repo guidance), `addActivationRepositoryAndOutputSteps` (checkout/restore + lock-file check + version check + text sanitization + status comment + issue lock + output finalization), and `buildActivationPermissions` (base + command/event + label + script-inferred permission assembly). These methods mixed orchestration with detailed step-emission logic, were cyclomatic-complexity hotspots, and made individual branches hard to read or test in isolation — while their generated-YAML output contract (`setup-*`, `engine_id`, `model`, `target_*`, stale-lock outputs, comment outputs) was correct and had to be preserved exactly.

## Decision

We decided to decompose each oversized method into a thin orchestrator that delegates to focused, single-responsibility helpers, preserving existing output and behavior exactly. `newActivationJobBuildContext` now calls `newActivationBuildContext`, `cacheActivationPreStepPermissions`, `addActivationSetupAndWorkflowCallSteps`, and `addActivationEngineOutputs`. `addActivationFeedbackAndValidationSteps` delegates to `maybeAddActivationAppTokenMintStep` (with `activationJobNeedsAppToken` and `buildActivationAppTokenPermissions`), `addActivationReactionStep`, `addActivationSecretValidationStep`, and `addActivationCrossRepoGuidanceStep`. `addActivationRepositoryAndOutputSteps` delegates to per-step helpers (`addActivationCheckoutAndBaseRestoreStep`, `addActivationLockFileStep`, `addActivationVersionCheckStep`, `addActivationTextOutputStep`, `addActivationStatusCommentStep`, `addActivationIssueLockStep`, `ensureActivationCommentOutputs`) plus reusable composition helpers (`computeActivationSanitizationDomains`, `buildActivationTextOutputEnvLines`, `appendActivationSafeOutputMessagesEnv`). `buildActivationPermissions` delegates to staged map builders (`buildActivationBasePermissions`, `addCentralizedCommandActivationPermissions`, `addActivationLabelPermissions`, `addActivationScriptPermissions`). This follows the same plain-helper-function convention established by [ADR-36694](36694-decompose-buildcustomjobs-into-helpers.md).

## Alternatives Considered

### Alternative 1: Leave the methods as-is (suppress the lint rule)
We could have left the monolithic methods in place and suppressed the `largefunc` warnings with inline directives or a per-file exclusion. This avoids churn and regression risk, but does nothing to reduce cyclomatic complexity, keeps the multi-concern branches hard to test, and normalizes silencing the lint signal the daily tracking exists to surface. Rejected because it defers the maintainability problem indefinitely.

### Alternative 2: Introduce a dedicated `ActivationJobBuilder` type
We could have extracted the logic into a builder struct holding `ctx`, `data`, and compiler context, exposing fluent methods like `withFeedback()` and `withRepositorySteps()`. This would centralize state, but introduces a new type and lifecycle, diverges from the free-function/method style used throughout the activation builder and the precedent set by ADR-36694, and is a larger structural change than a lint-driven decomposition warrants. Rejected in favor of plain helpers that match existing conventions.

## Consequences

### Positive
- Each orchestrator drops below the `largefunc` threshold, making the high-level activation-job assembly flow readable at a glance.
- Individual concerns (app-token permissions, secret validation, text-output sanitization, permission staging) can now be reasoned about and tested independently.
- Extracted helpers like `computeActivationSanitizationDomains` and `appendActivationSafeOutputMessagesEnv` remove inline branching and make the step-emission logic reusable and self-documenting.

### Negative
- The number of functions in `compiler_activation_job_builder.go` increases substantially, so navigation now requires jumping between many small functions rather than scrolling one method.
- Behavior-preserving refactors of this size carry regression risk in subtle output ordering and conditional emission (e.g., the order in which steps and `ctx.outputs` entries are appended must match exactly); this relies on existing snapshot/compiler tests to catch divergence.

### Neutral
- Generated workflow YAML output shape is unchanged; this is purely an internal restructuring with no user-visible or schema impact.
- Several helpers were converted from operating on a local `data` variable to reading `ctx.data` directly, with no change in resolved values.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/27066992595) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
