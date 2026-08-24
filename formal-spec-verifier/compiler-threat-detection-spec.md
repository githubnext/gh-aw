# Formal Notes: compiler-threat-detection-spec.md

**Last formalized**: 2026-08-24-15-45-11
**Notation**: SMT-LIB / Z3-style guard conjunction
**Issue**: (created via safe-output; number assigned post-processing)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `PositiveIntLiteralAccepted` | Positive integer literal (int/int64/uint64/float64 whole) accepted for any job; TimeoutMinutesExpression cleared |
| P2 | `NonPositiveRejected` | Zero/negative numeric value rejected for all jobs |
| P3 | `GeneratedJobExpressionRejected` | `agent`/`detection` jobs reject any string value, including well-formed `${{ ... }}` expressions |
| P4 | `NonGeneratedExpressionAccepted` | Non-generated jobs may use a valid GHA expression string; TimeoutMinutes=0, expression stored |
| P5 | `EmptyStringRejected` | Empty/whitespace-only string rejected regardless of job |
| P6 | `NonIntegralFloatRejected` | float64 with fractional part rejected |
| P7 | `OutOfRangeFloatRejected` | float64 > 2^53 (maxLosslessIntFloat64) rejected |
| P8 | `UnsupportedTypeRejected` | bool/slice/map/nil rejected via default case |
| P9 | `AbsentFieldIsNoOp` | Missing `timeout-minutes` key => nil error, no mutation |

## Key Invariants

- `extractCustomJobTimeoutMinutes` (pkg/workflow/compiler_custom_job_properties.go) is a hard compile-time boundary in all modes (not strict-mode-only) per CTR-026.
- Generated jobs (`agent`, `detection`) are structurally distinguished from custom jobs by `jobName` equality against `constants.AgentJobName`/`constants.DetectionJobName`; this check gates whether string/expression values are permitted at all.
- `TimeoutMinutesExpression` and `TimeoutMinutes` are mutually exclusive in valid states: exactly one is meaningfully set for any accepted config.

## Edge Cases Identified

- float64 exceeding 2^53 lossless-integer threshold (precision-loss guard), distinct from simple non-integral rejection.
- Absent field vs. invalid field must be distinguished (no-op vs. error).
- Non-generated custom job accepting expressions while generated jobs never do — asymmetric behavior gated purely on job identity string comparison.

## Notes for Future Runs

- This run focused narrowly on CTR-026 (newest rule, v1.0.26), the only rule not yet formalized in the 2026-08-10 pass (which covered CTR-022/CTR-023).
- CTR-016/CTR-017 (manifest drift, secret leakage in env) remain good candidates for deeper formalization, per prior notes.
- A future run could extend this suite to cover `resolveAgentJobTimeoutValue`/`resolveDetectionJobTimeoutValue` (compiler_main_job.go / threat_detection_job.go) which produce the default expression when timeout-minutes is unset in frontmatter, complementing this predicate set.
- Catalog-level predicates (P1-P10 from 2026-07-28) and CTR-022/CTR-023 predicates (P1-P12 from 2026-08-10) remain valid — not re-derived this run.

---
(Prior run notes retained below for continuity)

## Prior Run (2026-08-10) — CTR-022/CTR-023 Predicates

P1 SafeRefAccepted, P2 SafePathAccepted, P3 HyphenPrefixRejected, P4 NulByteRejected, P5 TraversalRejected,
P6 AbsolutePathRejected, P7 EmptyValueRejected, P8 BashWildcardIsSafe, P9 BashAbsentIsSafe,
P10 BashFalseIsRestriction, P11 BashEmptyListIsRestriction, P12 BashNamedListIsRestriction.

## Prior Run (2026-07-28) — Catalog-Level Model

P1 RuleModelComplete, P2 DeterministicResponse, P3 SecureOutcome, P4 NoHyphenPrefixPassthrough (CTR-013),
P5 WorkflowsFieldMandatory (CTR-021), P6 BranchScopeModeSensitive (CTR-021), P7 DeprecationRetainsEntry,
P8 DeprecatedTestsNotRequired, P9 VersionSyncInvariant, P10 Conformance (conjunction of P1-P9).
