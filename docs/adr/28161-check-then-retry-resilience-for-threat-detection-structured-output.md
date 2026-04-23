# ADR-28161: Check-Then-Retry Resilience Pattern for Threat Detection Structured Output

**Date**: 2026-04-23
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

The threat detection pipeline relies on an LLM engine to analyze PR content and emit a structured JSON result preceded by an exact `THREAT_DETECTION_RESULT:` marker line. The parser downstream greps for this marker and fails hard when it is absent. The LLM model intermittently omits the marker — particularly when the prompt context is long and the output format instruction was buried at the end of the prompt — causing recurring parse failures across multiple agentic workflow runs. Because the failure mode is non-deterministic and model-driven, it cannot be eliminated purely through input validation or stricter types.

### Decision

We will add a two-step **check-then-retry** pattern immediately after the primary engine execution step in every detection job. A lightweight shell step (`detection_result_check`) greps the detection log for `THREAT_DETECTION_RESULT:` and sets a `retry_needed` output flag. A conditional retry step (`detection_agentic_execution_retry`) re-runs the full engine execution — skipping reinstall — only when `retry_needed == 'true'`. Because the retry appends to the same log file via `tee -a`, the existing parser picks up the result without any changes to the parsing layer. In parallel, we will move the required output format instruction to the **first section** of the prompt (immediately after the role introduction) to reduce how often the marker is omitted in the first place, addressing root cause alongside the safety net.

### Alternatives Considered

#### Alternative 1: Fuzzy Parsing Without Retry

Instead of retrying the engine, modify the downstream parser to accept the JSON payload even when the `THREAT_DETECTION_RESULT:` marker is absent — using heuristics such as detecting a bare JSON object at the end of the output. This avoids the extra execution cost of a retry, but shifts complexity into the parser, increases the risk of false positives (treating unrelated JSON in agent output as the detection result), and does not improve the model's compliance rate — the root problem remains unfixed. It also requires changing the parser contract that all callers already rely on.

#### Alternative 2: Stricter Prompt Engineering Only, No Retry

Reposition the output format instruction to the top of the prompt and improve its wording without adding a retry step. This lowers failure frequency but cannot reduce it to zero: model behavior is stochastic, and long contexts can still cause the instruction to be overlooked. Accepting residual failures means workflows that do trigger the bug must be re-run manually, which is disruptive in a security-sensitive pipeline where parse failures block merges.

#### Alternative 3: Use Model-Native Structured Output (Function Calling / JSON Mode)

Replace the text-prefixed result format with the model API's native structured output capability (tool use or JSON mode), which guarantees valid JSON without a marker line. This is the most robust long-term solution but requires a significant change to the engine integration layer, is not uniformly available across all engine types used (Copilot CLI vs. Claude Code CLI vs. future engines), and would take longer to ship. The check-then-retry approach provides immediate relief while this direction can be revisited separately.

### Consequences

#### Positive
- Threat detection parse failures auto-recover without human intervention, improving pipeline reliability in a security-sensitive code path.
- The prompt instruction relocation addresses the root cause (model skipping late-prompt instructions), making retries less likely to trigger in practice.
- No changes to the parser or downstream result consumers are required; the fix is fully contained within the workflow orchestration layer.

#### Negative
- When the retry does trigger, the detection job can run up to 20 additional minutes (the retry step timeout), increasing wall-clock latency for the affected workflow run.
- Re-running the full engine execution on retry incurs additional compute and LLM API costs, including cloud runner time and model token costs.
- The `tee -a` log-append behavior is a shared assumption between the primary step and the retry; if the log file path or append behavior changes in future, both steps must be updated in sync.

#### Neutral
- The refactoring of `buildDetectionEngineExecutionStep` into a shared `prepareDetectionEngineAndData` helper is a necessary enablement for the retry builder to reuse engine-resolution logic without duplication.
- The retry step is conditional (`always() && retry_needed == 'true'`), so the common case (successful first run) adds only the negligible cost of the grep check step.
- The pattern is applied uniformly across all lock files that contain a detection job, keeping behavior consistent regardless of which workflow triggers detection.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Detection Result Check Step

1. Implementations **MUST** include a `detection_result_check` step that runs after the primary engine execution step and before artifact upload, conditioned on `always() && steps.detection_guard.outputs.run_detection == 'true'`.
2. The check step **MUST** grep the detection log for the `THREAT_DETECTION_RESULT:` marker and set a `retry_needed` output to either `true` or `false`.
3. Implementations **MUST NOT** fail the workflow at the check step; it **SHALL** only set the output flag and exit with code 0.

### Retry Execution Step

1. Implementations **MUST** include a `detection_agentic_execution_retry` step conditioned on `always() && steps.detection_guard.outputs.run_detection == 'true' && steps.detection_result_check.outputs.retry_needed == 'true'`.
2. The retry step **MUST** use the same engine command, prompt file, and environment variables as the primary execution step.
3. The retry step **MUST** append its output to the same detection log file (via `tee -a`) so the existing parser can find the result without modification.
4. The retry step **SHOULD** skip dependency reinstall steps (e.g., package installs) to minimize added latency.
5. Implementations **MUST NOT** reset or truncate the detection log file before the retry; the log **SHALL** be append-only from the primary step onward.

### Prompt Format Instruction Placement

1. The required output format instruction (`THREAT_DETECTION_RESULT:` marker and JSON schema) **MUST** appear as the first substantive section of the detection prompt, immediately after the role introduction.
2. Implementations **MAY** retain a secondary copy of the format instruction later in the prompt as a reminder, but the primary instruction **MUST** precede all analysis and context sections.

### Engine Logic Reuse

1. Implementations **MUST** extract shared engine-resolution and `WorkflowData` construction logic into a single helper (e.g., `prepareDetectionEngineAndData`) used by both the primary and retry step builders.
2. Implementations **MUST NOT** duplicate engine-resolution logic between the primary and retry step builders.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24860540810) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
