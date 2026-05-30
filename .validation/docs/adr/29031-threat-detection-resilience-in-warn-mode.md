# ADR-29031: Threat Detection Parse-Step Resilience in Warn Mode

**Date**: 2026-04-29
**Status**: Draft
**Deciders**: Unknown (automated fix by copilot-swe-agent)

---

## Part 1 — Narrative (Human-Friendly)

### Context

The agentic workflow framework (AWF) includes a `detection` job that runs an AI-powered threat detection scan and feeds its result into a `safe_outputs` job gated on `needs.detection.result == 'success'`. This design means that any failure in the `detection` job — including infrastructure failures such as an unhealthy squid proxy container or an AI model that produces no `THREAT_DETECTION_RESULT` token — blocks the `safe_outputs` step and marks the entire workflow run as failed. The threat detection system operates in two modes: *warn mode* (the default, `continue-on-error: true`), where detection failures are surfaced as warnings but should not stop the workflow, and *strict mode* (`continue-on-error: false`), where detection failures must block `safe_outputs`. Infrastructure transients were incorrectly treated the same as real threat detections under both modes, causing unnecessary workflow failures.

### Decision

We will add `continue-on-error: true` to the "Parse and conclude threat detection" YAML step when the workflow is in warn mode, and will wrap the entire JavaScript parse script body in a top-level `try/catch` that routes unanticipated runtime exceptions through `setDetectionFailure()` rather than letting them crash the step without setting outputs. These two measures form a defence-in-depth: the YAML flag prevents the detection job from failing when the parse step throws; the JS wrapper ensures step outputs are always set even when an unexpected error occurs inside the parse logic. In strict mode, neither measure is applied so that genuine detection failures continue to block `safe_outputs`.

### Alternatives Considered

#### Alternative 1: YAML `continue-on-error` only (no JS try/catch)

Adding `continue-on-error: true` to the parse step at the YAML level alone would prevent the detection job from failing due to the step exit code. However, if the parse script crashes unexpectedly before calling `core.setOutput`, downstream steps that read `steps.detection_conclusion.outputs.*` receive empty strings, which can produce incorrect conclusions. The JS try/catch is necessary to guarantee outputs are always set regardless of where inside the script an error occurs.

#### Alternative 2: Relax the `safe_outputs` gate condition

Instead of patching the detection job, we could change the `safe_outputs` dependency from `needs.detection.result == 'success'` to a weaker condition such as `needs.detection.result != 'cancelled'`. This would make `safe_outputs` tolerate a failed detection job outright. This was not chosen because it weakens the strict-mode guarantee: in strict mode a real threat detection must prevent `safe_outputs`, so the gate must remain strict. Patching resilience into the detection job itself preserves the gate semantics while fixing the infrastructure-failure case.

### Consequences

#### Positive
- In warn mode, transient infrastructure failures (unhealthy squid container, model producing no `THREAT_DETECTION_RESULT` token, I/O errors) no longer block `safe_outputs` or fail workflow runs.
- Step outputs (`conclusion`, `success`, `reason`) are guaranteed to be set on every execution path, eliminating silent empty-output failures.

#### Negative
- In warn mode, a genuine parse failure (e.g., model output format regression) now surfaces as a `warning` conclusion rather than blocking `safe_outputs`. Teams relying on warn mode may miss real detection regressions unless they monitor the warning conclusion explicitly.
- The warn/strict mode bifurcation in the compile step (`buildDetectionConclusionStep`) and in the JS script adds branching complexity that must be kept in sync.

#### Neutral
- All 204 compiled `.lock.yml` files must be recompiled when the Go template changes; this is handled automatically by the compiler and is not a manual burden.
- The `TestDetectionConclusionStepContinueOnError` test suite codifies the warn/strict distinction so future regressions are caught at compile time.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Threat Detection Parse Step — Warn Mode

1. When the threat detection configuration specifies warn mode (`continue-on-error: true`), the compiled YAML step with `id: detection_conclusion` **MUST** include `continue-on-error: true`.
2. The JavaScript parse script **MUST** wrap its entire execution body in a top-level `try/catch` block so that any uncaught runtime exception is routed through `setDetectionFailure()`.
3. `setDetectionFailure()` **MUST** always call `core.setOutput` for `conclusion`, `success`, and `reason` before returning, regardless of the failure mode.
4. In warn mode, `setDetectionFailure()` **MUST NOT** call `core.setFailed()`, ensuring the step exits with code 0 and the detection job can succeed.

### Threat Detection Parse Step — Strict Mode

1. When the threat detection configuration specifies strict mode (`continue-on-error: false`), the compiled YAML step with `id: detection_conclusion` **MUST NOT** include `continue-on-error: true`.
2. In strict mode, `setDetectionFailure()` **MUST** call `core.setFailed()` so that a detection failure propagates as a job failure and blocks `safe_outputs`.

### Compiler Consistency

1. The Go function `buildDetectionConclusionStep` **MUST** read the `continueOnError` flag from workflow configuration and conditionally emit `continue-on-error: true` only when the flag is true.
2. Changes to the `buildDetectionConclusionStep` function **MUST** be followed by a full recompilation of all lock files; no lock file **MAY** diverge from the compiled output of its source workflow file.
3. The behaviour of `buildDetectionConclusionStep` with respect to the `continue-on-error` flag **MUST** be covered by table-driven tests that assert both warn-mode and strict-mode outputs.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25090276793) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
