# ADR-28160: Interface-Level Node Driver Detection for the Threat Detection Job

**Date**: 2026-04-23
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

The agentic workflow compiler generates a separate *threat detection job* that runs on a fresh GitHub Actions runner. Unlike the main agent job, this detection job does not call `DetectRuntimeRequirements`, so any runtime tooling the engine needs (beyond what its install steps bring) must be emitted explicitly. The Copilot engine wraps its CLI with a Node.js driver script (`GetDriverScriptName() != ""`), but the detection job compiler never provisioned Node.js before this PR — causing `node: command not found` at the `Execute GitHub Copilot CLI` step on any runner without ambient Node, followed by a silent `No THREAT_DETECTION_RESULT found`. Claude and Codex were unaffected because their install steps bundle `Setup Node.js` via `BuildStandardNpmEngineInstallSteps`.

### Decision

We will detect whether an engine needs a Node.js setup step by performing a Go interface type-assertion to `DriverProvider` and inspecting `GetDriverScriptName()`. If the method returns a non-empty string, the detection job compiler prepends `GenerateNodeJsSetupStep()` to the install steps — unless the engine's own install steps already contain a `Setup Node.js` step (guarded by `installStepsContainNodeSetup`, which uses the same `extractStepName` matcher as `JobManager.ValidateDuplicateSteps`). This interface-level approach was chosen over config-keyed or engine-type-switch strategies because it scales automatically to any future `DriverProvider` engine without per-engine boilerplate and keeps the contract at the Go type boundary rather than in YAML configuration.

### Alternatives Considered

#### Alternative 1: Config-keyed flag (`requires_node_driver: true`)

Add an explicit boolean field to `EngineConfig` that workflow authors set when their engine uses a Node.js driver. This makes the dependency visible in YAML but requires every driver-using engine to declare it explicitly, creates a documentation burden, and allows misconfiguration (flag set but no driver used, or vice versa).

#### Alternative 2: Engine type switch in the detection compiler

Use a Go `type switch` (e.g., `case *CopilotEngine`) inside `buildDetectionEngineExecutionStep` to emit the setup step only for known concrete types. This was rejected because it creates a hardcoded list that must be updated every time a new driver-based engine is added, coupling the detection compiler to every concrete engine type.

#### Alternative 3: Call `DetectRuntimeRequirements` from the detection job

Run the same runtime detection logic the main agent job uses. This was rejected because `DetectRuntimeRequirements` is designed to operate on the primary execution context and carries side effects (setting job-level env vars, modifying the primary job's step list) that are incorrect for the isolated detection job runner.

### Consequences

#### Positive
- Any future engine that implements `DriverProvider` with a non-empty driver script name automatically gets `Setup Node.js` in its detection job without a code change in the detection compiler.
- The dedup guard reuses the same `extractStepName` matcher as `ValidateDuplicateSteps`, so it cannot drift from what the validator treats as a duplicate step.
- `DriverScript` is now preserved when the detection engine config is rebuilt from the primary config (was silently dropped before, causing detection-specific driver overrides to be ignored).

#### Negative
- The Node.js dependency is expressed through Go interface implementation, which is invisible in the compiled YAML — a future maintainer reading a detection job's YAML cannot see *why* `Setup Node.js` appears without tracing back to `engineRequiresNodeDriver`.
- An engine that implements `DriverProvider` but whose driver does not actually require Node.js would incorrectly receive a `Setup Node.js` step. The assumption that `GetDriverScriptName() != ""` implies a Node.js driver is implicit and undocumented at the interface boundary.

#### Neutral
- The 200+ compiled lock files were regenerated as a bulk change in this PR, making the diff large. Copilot detection jobs gain `Setup Node.js` immediately before `Install GitHub Copilot CLI`; Claude/Codex detection jobs are unchanged.
- The fix also corrects a pre-existing bug where `DriverScript` was silently dropped from the detection engine config rebuild — this is a separate correctness fix bundled into the same commit.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Node.js Setup Step Emission

1. The detection job compiler **MUST** emit a `Setup Node.js` step for any engine that satisfies `DriverProvider` and returns a non-empty value from `GetDriverScriptName()`.
2. The detection job compiler **MUST NOT** emit a duplicate `Setup Node.js` step when the engine's own install steps already contain one.
3. The dedup guard **MUST** use the same step-name extraction logic (`extractStepName`) as `JobManager.ValidateDuplicateSteps` to determine whether a `Setup Node.js` step is already present.
4. Implementations **MUST NOT** use a hardcoded type switch over concrete engine types to decide whether to emit the Node.js setup step.

### Detection Engine Config Preservation

1. When rebuilding the detection engine config from the primary engine config, implementations **MUST** copy the `DriverScript` field alongside all other config fields (`ID`, `Model`, `Version`, `Env`, `Config`, `Args`, `APITarget`).
2. Implementations **MUST NOT** silently drop fields from the primary engine config during the detection config rebuild.

### Interface Contract

1. A `DriverProvider` implementation whose driver script requires Node.js **MUST** return a non-empty string from `GetDriverScriptName()`.
2. A `DriverProvider` implementation whose driver script does **not** require Node.js **SHOULD** return an empty string from `GetDriverScriptName()`, or **SHOULD NOT** implement `DriverProvider` at all, to avoid an incorrect `Setup Node.js` emission in the detection job.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
