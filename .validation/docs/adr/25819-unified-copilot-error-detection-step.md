# ADR-25819: Unified Copilot Error Detection Step with No-Retry on Policy Errors

**Date**: 2026-04-11
**Status**: Draft
**Deciders**: Unknown (generated from PR #25819 diff)

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw Copilot engine must handle two distinct, non-transient failure classes that manifest at runtime:
(1) **inference access errors** — the `COPILOT_GITHUB_TOKEN` is valid but lacks Copilot inference access (surfaced as "Access denied by policy settings"); and
(2) **MCP policy errors** — an enterprise or organization policy has disabled MCP servers for the Copilot CLI, causing all configured MCP servers to be silently blocked (surfaced as "MCP servers were blocked by policy").
Prior to this change, each failure class was detected by a dedicated bash script (`detect_inference_access_error.sh` and `detect_mcp_policy_error.sh`) injected as separate post-execution steps, and the Copilot driver script would retry on *all* failures — including these permanent configuration errors.
The duplication increased maintenance burden, the bash scripts were difficult to unit-test, and retrying on policy errors wasted compute without any chance of success.

### Decision

We will consolidate both error-detection scripts into a single Node.js script (`detect_copilot_errors.cjs`) that scans the agent stdio log for both error patterns in one pass and sets both GitHub Actions step outputs atomically.
We chose JavaScript over bash because the detection logic already runs in the same Node.js runtime as the Copilot driver, enabling Jest-based unit tests with shared constants and eliminating a class of bash quoting/portability issues.
Additionally, we will add an explicit early-exit path in the driver retry loop for MCP policy errors: when `MCP_POLICY_BLOCKED_PATTERN` is detected, the driver breaks out of the retry loop immediately rather than sleeping and re-invoking the Copilot CLI, because policy configuration is a persistent external constraint that no number of retries can resolve.
The Go compiler (`generateCopilotErrorDetectionStep`) is updated to emit one step instead of two separate functions.

### Alternatives Considered

#### Alternative 1: Retain two separate bash scripts, add a third for MCP policy

The simplest change would have been to add only the no-retry behaviour in the driver and keep the two bash detection scripts unchanged, adding a third bash script for MCP policy detection.
This was rejected because it would have increased the count of runtime scripts to three, kept the bash-based detection that is difficult to unit-test, and missed the opportunity to set both outputs atomically and share pattern constants between the detection script and the driver.

#### Alternative 2: Single bash script consolidation

Both detection patterns could have been merged into a single bash script (replacing the two existing bash scripts) without switching to JavaScript.
This was rejected because the driver retry logic is already implemented in JavaScript and already needed the `MCP_POLICY_BLOCKED_PATTERN` constant for the no-retry decision; duplicating the pattern in bash and JavaScript would create a single point of divergence where a future pattern update could be made in only one place.

#### Alternative 3: Structured error signalling from the Copilot CLI

The ideal long-term solution would be for the Copilot CLI to emit machine-readable exit codes or structured output for each failure class, making log scraping unnecessary.
This was not viable at the time of this decision because the error signals originate inside the Copilot CLI binary, which is a separate codebase not under gh-aw control.
This option remains open for future adoption once structured errors are available upstream.

### Consequences

#### Positive
- Both error types are detected in a single log-scan pass, reducing I/O on large log files.
- The pattern constants (`INFERENCE_ACCESS_ERROR_PATTERN`, `MCP_POLICY_BLOCKED_PATTERN`) are shared between `detect_copilot_errors.cjs` and `copilot_driver.cjs`, ensuring they are always in sync.
- The detection script is covered by Jest unit tests with explicit fixture strings, giving higher confidence than bash `grep` scripts.
- Retrying on MCP policy errors is eliminated, preventing wasted compute budget and confusing multi-attempt logs when a configuration-level problem will never self-heal.
- Actionable guidance (how to re-enable MCP servers in enterprise/org settings) is surfaced in failure issues via a Markdown template with progressive disclosure.

#### Negative
- The detection step depends on Node.js being available in `${RUNNER_TEMP}/gh-aw/actions/`; if the setup action fails to copy scripts, neither error type will be detected.
- Detection remains log-scraping based, which is fragile against Copilot CLI message changes. Any future rewording of the error strings requires a coordinated pattern update in gh-aw.
- The no-retry early exit means that a transient network hiccup that coincidentally produces an MCP-policy-like log line would not be retried (though such coincidental matches are considered unlikely given the specificity of the pattern).

#### Neutral
- The two bash scripts (`detect_inference_access_error.sh`, `detect_mcp_policy_error.sh`) are removed, reducing the total number of runtime scripts by two.
- All compiled workflow lock files (`.lock.yml`) are regenerated to reference `detect-copilot-errors` instead of `detect-inference-error` and to include the new `mcp_policy_error` output wire.
- The Go compiler function `generateInferenceAccessErrorDetectionStep` (and the now-removed `generateMCPPolicyErrorDetectionStep`) are replaced by the single `generateCopilotErrorDetectionStep` function.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Error Detection Script

1. Implementations **MUST** use a single JavaScript step (`detect_copilot_errors.cjs`) to detect both `inference_access_error` and `mcp_policy_error` from the agent stdio log in one execution.
2. Implementations **MUST NOT** use separate bash scripts for individual Copilot CLI error type detection; all Copilot-specific error detection **MUST** be consolidated in `detect_copilot_errors.cjs`.
3. The detection script **MUST** set both `inference_access_error` and `mcp_policy_error` as GitHub Actions step outputs regardless of which (if any) error was detected.
4. The detection step **MUST** run with `if: always()` and `continue-on-error: true` so that detection failures do not block the workflow conclusion.
5. Pattern constants used for log scanning **MUST** be exported from `detect_copilot_errors.cjs` (via `module.exports`) so that they can be imported and reused by other scripts (e.g., the driver) without duplication.

### Driver Retry Behaviour

1. The Copilot driver **MUST NOT** retry a run whose output matches `MCP_POLICY_BLOCKED_PATTERN`; it **MUST** break out of the retry loop immediately and propagate the exit code.
2. The driver **SHOULD** log a human-readable message when skipping a retry due to a policy error, explaining that the failure is a configuration issue rather than a transient error.
3. For all other failure classes, the driver retry rules **MUST** remain unchanged: up to `MAX_RETRIES` attempts with exponential back-off.

### Compiler Step Generation (Go)

1. The Go compiler **MUST** emit exactly one `detect-copilot-errors` step for the Copilot engine (replacing any prior separate inference/MCP detection steps).
2. The compiler **MUST NOT** emit separate `detect-inference-error` or `detect-mcp-policy-error` steps.
3. The agent job outputs **MUST** include both `inference_access_error` and `mcp_policy_error` wired from `steps.detect-copilot-errors.outputs.*` for the Copilot engine.
4. These outputs **MUST NOT** be emitted for non-Copilot engines.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
