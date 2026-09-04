# Formal Notes: security-architecture-spec.md

**Last formalized**: 2026-09-04-15-31-28
**Notation**: TLA+-style predicate logic
**Issue**: pending (see workflow run)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| TD01 | `TD01_AutomatedDetectionRequired` | Unconfigured threat-detection key still yields a non-nil, runnable default config |
| TD02/TD03 | `TD02_TD03_DisableSemantics` | `true` enables; `false` / `enabled:false` disable (nil config) |
| TD04 | `TD04_DetectionCategoriesComplete` | Three mandatory categories (prompt_injection, secret_leak, malicious_patch) |
| TD08 | `TD08_StructuredOutputShape` | Detection output has the 4 required JSON keys, reasons always present |
| TD09 | `TD09_AnyThreatBlocksSafeOutputs` | Threat detected + not continue-on-error => safe outputs blocked |
| TD10 | `TD10_ReasonsExplainDetectedThreats` | reasons populated when any threat flag true |
| TD12 | `TD12_CustomPromptAppendsNotReplaces` | Custom prompt stored verbatim, additive to default prompt |
| TD13/TD14 | `TD13_TD14_EngineOverride` | String engine override and full engine object override |
| TD15 | `TD15_EngineFalseKeepsCustomSteps` | engine:false + custom steps keeps job runnable |
| PM12 | `PM12_FailedRoleCheckAnalogue` | Fail-closed gate analogy between pre_activation role check and threat-detection gate |

## Key Invariants

- Threat detection is auto-enabled whenever safe-outputs is configured; only explicit `false` (top-level or `enabled: false`) disables it.
- Detection output is always a 4-key structured object; `reasons` is never nil, non-empty exactly when a threat flag is true.
- Blocking of safe outputs is a two-factor gate: threat detected AND NOT continue-on-error (default continue-on-error is true, i.e. warn-only by default).
- Custom prompts are additive-only; the parser never truncates or replaces baseline instructions.
- `engine: false` disables only the AI engine, not custom `steps`; runnability depends on whether any steps remain.
- Expression-controlled `enabled` (e.g. `${{ inputs.x }}`) forces the detection job to always compile (`IsConditional()` true), deferring the true enable/disable decision to runtime.

## Edge Cases Identified

- `engine: false` with zero custom `steps` — job has nothing to run, `HasRunnableDetection()` must be false.
- `enabled` given as a GitHub Actions expression string rather than a literal bool — `EnabledExpr` is set and the job always compiles.
- `threat-detection` given as a non-expression, non-boolean plain string — invalid per JSON schema; parser ignores it and falls through to default (enabled) config rather than erroring.

## Notes for Future Runs

- This run added `pkg/workflow/threat_detection_formal_test.go` covering Section 9 (TD-01..TD-15) of `specs/security-architecture-spec.md`, the layer previously flagged as uncovered by the 2026-08-05 run.
- Remaining under-covered areas per the prior run's assessment: Section 10.6-10.8 (Action Pinning, Deprecated Features, Compile-Time vs Runtime tradeoffs) and Section 7.6 pre_activation pattern (PM-10a/b/c/d) still lack dedicated formal predicate test files — good candidates for a future run.
- The detection *output* JSON schema (TD-08) has no exported Go type yet; tests use a local `stubDetectionOutput` struct. If/when the codebase adds an exported detection-output type, replace the stub and re-point TD08/TD09/TD10 tests at it.
- Because `parseThreatDetectionConfig` is unexported, the new test file lives in `package workflow` (not `workflow_test`), consistent with existing `threat_detection_config_test.go`.
