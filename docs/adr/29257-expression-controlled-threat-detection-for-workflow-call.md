# ADR-29257: Expression-Controlled Threat Detection for `workflow_call` Reuse

**Date**: 2026-04-30
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `safe-outputs.threat-detection` field previously accepted only compile-time boolean or object values. This made it impossible for reusable `workflow_call` workflows to expose threat detection on/off control to callers via `inputs` — every combination required a separate workflow file. The same limitation applied to the `enabled` and `continue-on-error` sub-fields of the object form. The codebase had already established a pattern of accepting GitHub Actions expression strings for other parameterizable fields (PR #29212 for list constraints; PR #29230 for `protected-files` and `patch-format`); this PR applies the same pattern to the threat detection subsystem. Unlike those prior cases, threat detection is compiled as a separate GitHub Actions job rather than a handler configuration value, so allowing runtime control requires changes to the `if:` conditions on the `detection` job and on all downstream jobs that depend on its result (`safe_outputs`, `safe_jobs`, `upload_assets`).

### Decision

We will allow `safe-outputs.threat-detection` to accept a GitHub Actions expression string (matching `${{...}}`), and will allow the `enabled` and `continue-on-error` sub-fields of the object form to accept `templatable_boolean` (bool literal or expression string). When an expression is detected at parse time, the compiler stores it in `EnabledExpr` / `ContinueOnErrorExpr` fields of `ThreatDetectionConfig` and always emits the `detection` job (never skips compilation). The detection job's `if:` condition is extended with the raw caller expression so GitHub Actions evaluates it at runtime and skips the job when the expression resolves to `false`. Because a skipped `detection` job would cause downstream jobs that depend on it with a strict success condition to also be skipped silently, the `safe_outputs`, `safe_jobs`, and `upload_assets` jobs switch to `always() && (success || skipped)` semantics (`buildDetectionPassedCondition`) whenever the detection configuration is expression-controlled.

### Alternatives Considered

#### Alternative 1: Separate Workflow Files per Detection Mode

Callers could maintain distinct workflow copies for each threat detection configuration. This was the status quo before this PR. It was rejected for the same reason as in ADR-29230: the duplication does not scale, and every structural change to the base workflow must be propagated to all variants, which teams routinely fail to do.

#### Alternative 2: Add a New Dedicated `threat-detection-expression` Field

A separate field (e.g., `threat-detection-enabled-expression`) could accept only expression strings while the existing field remained bool-only. This avoids changing the type of `threat-detection` but doubles the surface area and requires callers to know which field to use in which context. It was rejected as unnecessarily complex given that the `${{...}}`-pattern detection approach is already established in the codebase and yields a single, self-describing field.

#### Alternative 3: Relax Gate Conditions on All Downstream Jobs Unconditionally

Instead of conditionally switching downstream jobs to `always() && (success || skipped)` only when detection is expression-controlled, the gate condition could be relaxed for all workflows regardless of detection configuration. This was not chosen because it would weaken the strict-mode guarantee for workflows with static detection configurations, where a failed detection job must still block `safe_outputs`.

### Consequences

#### Positive
- A single reusable `workflow_call` workflow can expose threat detection as a caller-controlled input without duplication.
- Literal boolean and object forms remain fully backward-compatible; existing workflows are unaffected.
- The `detection` job is always compiled when an expression is provided, ensuring the compiled lock file is static regardless of which runtime value the expression resolves to.
- The fail-closed default (`always()` + `success || skipped` on downstream jobs) ensures that a misconfigured expression cannot silently skip safe-output handling.

#### Negative
- Expression values are only validated at runtime. A typo in a `workflow_call` input default (e.g., `default: ${{ inputs.typo }}`) will not be caught until the workflow executes.
- All three downstream job builders (`buildConsolidatedSafeOutputsJob`, `buildSafeJobs`, `buildUploadAssetsJob`) must independently check `IsConditionalDetection()` and apply the extended condition, adding three separate call sites to keep in sync.
- The `continue-on-error` step emitter (`buildDetectionConclusionStep`) must distinguish between a literal-true, literal-false, and expression value, adding a third branch to a function that previously had two.

#### Neutral
- The `extractRawExpression` helper was added to strip `${{` / `}}` wrappers from expression strings before embedding them in the YAML `if:` expression tree; this is a small pure utility with no broader side effects.
- The `IsConditionalDetection` package-level helper provides a single authoritative check that is reused across all three affected compiler functions.
- The `templatable_boolean` JSON schema `$ref` was already defined in the schema for other fields; this PR reuses it for `enabled` and `continue-on-error` without introducing a new definition.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Schema and Parsing

1. The `safe-outputs.threat-detection` field **MUST** accept either a boolean literal, an object form, or a GitHub Actions expression string matching `^\$\{\{.*\}\}$`.
2. The `enabled` and `continue-on-error` sub-fields of the object form **MUST** accept either a boolean literal or a GitHub Actions expression string (i.e., they **MUST** be typed as `templatable_boolean`).
3. When `threat-detection` is a boolean literal `false`, or the object form sets `enabled: false`, the parser **MUST** return `nil` (detection disabled).
4. When `threat-detection` is an expression string, the parser **MUST** store the full `${{...}}` string in `ThreatDetectionConfig.EnabledExpr` and **MUST NOT** treat the expression as a disabled state.
5. When `continue-on-error` is an expression string, the parser **MUST** store it in `ThreatDetectionConfig.ContinueOnErrorExpr` and **MUST NOT** set `ContinueOnError` (the literal bool field) for that configuration.

### Detection Job Compilation

1. When `ThreatDetectionConfig.EnabledExpr` is non-nil, the compiler **MUST** always emit a `detection` job (it **MUST NOT** skip compiling the job based on the expression value).
2. The `detection` job's `if:` condition **MUST** include the raw caller expression (the `${{...}}` wrappers stripped via `extractRawExpression`) appended with `&&` to the existing content guard condition.
3. The `continue-on-error` step field on the detection conclusion step **MUST** emit the expression string verbatim (unquoted) when `ContinueOnErrorExpr` is non-nil, rather than a `"true"` or `"false"` literal.
4. The `GH_AW_DETECTION_CONTINUE_ON_ERROR` environment variable on the detection conclusion step **MUST** emit the expression string verbatim when `ContinueOnErrorExpr` is non-nil.

### Downstream Job Conditions

1. When `IsConditionalDetection(data.SafeOutputs)` returns `true`, the `safe_outputs` job's `if:` condition **MUST** use `always() && <agent-not-skipped> && buildDetectionPassedCondition()` (accepting both `success` and `skipped` results from the detection job).
2. When `IsConditionalDetection(data.SafeOutputs)` returns `true`, every `safe_jobs` custom job's base condition **MUST** be wrapped with `always() && <safe-output-type-check> && buildDetectionPassedCondition()`.
3. When `IsConditionalDetection(data.SafeOutputs)` returns `true`, the `upload_assets` job's `if:` condition **MUST** use `always() && <upload-asset-check> && buildDetectionPassedCondition()`.
4. When `IsConditionalDetection(data.SafeOutputs)` returns `false` and detection is statically enabled, downstream jobs **MUST** continue to use the strict `needs.detection.result == 'success'` gate (via `buildDetectionSuccessCondition()`).

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement — in particular: skipping the detection job compilation for an expression-controlled config, failing to wrap downstream jobs with `always()` when detection is conditional, or emitting a quoted literal instead of the raw expression in step fields — constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25170616496) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
