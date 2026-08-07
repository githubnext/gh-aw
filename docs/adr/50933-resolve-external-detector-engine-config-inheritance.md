# ADR-50933: Resolve External Detector Engine Config via Three-Level Precedence Hierarchy

**Date**: 2026-08-07
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `gh-aw` compiler generates GitHub Actions workflows for AI agent workflows. When a workflow uses an external threat-detection engine, the compiler builds a separate execution environment via `buildExternalDetectorWorkflowData`. Previously, the external detector's `EngineConfig` was determined by `canReuseThreatDetectionEngineConfigForExternalDetector`, which reused the `safe-outputs.threat-detection.engine` override only when its declared ID exactly matched the resolved engine ID, and always fell back to a bare `&EngineConfig{ID: engineID}` otherwise. This meant pinned engine versions declared via shared engine definitions (applied to `EngineConfig.Version` at import time) were silently discarded for the external detector path, causing it to install the package's latest version rather than the same pinned version used by the main agent job.

### Decision

We will replace `canReuseThreatDetectionEngineConfigForExternalDetector` with `resolveExternalDetectorEngineConfig`, implementing a three-level precedence hierarchy for external detector engine config resolution:

1. An explicit `safe-outputs.threat-detection.engine` override, cloned with its ID normalized to the resolved detection engine ID (handles mismatched IDs such as pi→copilot normalization).
2. When no override is present, inherit `Version/Config/Args/HarnessScript/Driver` from the main `EngineConfig` (ensuring the same pinned version runs in both the agent job and the detection job).
3. A minimal config containing only the resolved engine ID as a final fallback.

This mirrors the precedence already used by the inline detection path (`buildDetectionEngineExecutionStep`).

### Alternatives Considered

#### Alternative 1: Only copy Version when IDs match

Extend the existing `canReuseThreatDetectionEngineConfigForExternalDetector` boolean helper to additionally copy the `Version` field when reusing the config. This would address the immediate version-drift symptom but would not inherit `Config/Args/HarnessScript/Driver`, would not handle mismatched ID overrides (e.g., pi→copilot detection normalization where the declared override ID differs from the engine actually used), and would perpetuate the asymmetry with the inline detection path.

#### Alternative 2: Unconditionally clone from main engine config

Copy all fields from the main `EngineConfig` without applying override precedence. This would fix version drift but would break existing use cases where users intentionally configure a different engine for threat detection via `safe-outputs.threat-detection.engine`, overriding the main engine completely.

### Consequences

#### Positive
- The external detector installs the same pinned engine version as the main agent job, eliminating silent version drift on the external detection path.
- Explicit override IDs that differ from the resolved engine ID (e.g., pi→copilot normalization) are now handled correctly via ID normalization in `cloneThreatDetectionEngineConfig`.
- The external detection path is now semantically consistent with the inline detection path.

#### Negative
- The resolution logic is more complex and requires understanding the three-level precedence to reason about which config fields are active.
- Inheriting all fields from the main engine config when no override is set means unexpected or unintended main-engine configuration (e.g., a test-only `HarnessScript`) could silently affect the external detector's behavior.

#### Neutral
- The old `canReuseThreatDetectionEngineConfigForExternalDetector` boolean helper is removed; callers now use `resolveExternalDetectorEngineConfig`, which always returns a non-nil `*EngineConfig`.
- Three regression tests are added covering the three precedence cases.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
