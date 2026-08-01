# ADR-49454: Propagate Model Config into External Threat Detection Path

**Date**: 2026-08-01
**Status**: Draft
**Deciders**: Unknown

---

### Context

The threat detection engine supports two execution paths: an inline path (`buildDetectionEngineExecutionStep`) and an external path (`buildExternalDetectorExecutionStep`). Workflows using `features: gh-aw-detection: true` (the external path) were failing with HTTP 400 errors from the AWF API proxy. The root cause was that `buildExternalDetectorExecutionStep` did not copy `Model`, `ModelMappings`, or `DefaultAiCreditsPricing` from the main `WorkflowData` into `threatDetectionData`. Without an explicit model, `addCopilotModelEnv` emits `COPILOT_MODEL: auto` (when no org vars are set), and the AWF API proxy has no pricing for `copilot/auto` and no default pricing configured, causing every inference attempt to return 400.

### Decision

We will propagate `Model`, `ModelMappings`, and `DefaultAiCreditsPricing` from `WorkflowData` into `threatDetectionData` in `buildExternalDetectorExecutionStep`, applying the same resolution priority already used in the inline path: per-workflow detection model override → main workflow model → env default → engine default. Pi→Copilot normalization (`extractPiModelID`) is applied where needed, mirroring the inline path exactly.

### Alternatives Considered

#### Alternative 1: Configure a Default Model/Pricing in the AWF API Proxy

The AWF API proxy could be updated to accept `copilot/auto` by adding a fallback pricing entry or a default model alias. This would have fixed the 400 errors without any changes to the detection builder code. However, it would mask the underlying configuration gap and could lead to unpredictable model selection for other consumers of the proxy, making billing and audit trails harder to reason about.

#### Alternative 2: Require Org Variables as a Prerequisite

Users of the external detection path could be required to set `GH_AW_MODEL_DETECTION_COPILOT` or `GH_AW_DEFAULT_MODEL_COPILOT` at the org level as a precondition for enabling the feature. This avoids any code change but creates an undocumented operational dependency that is easy to miss during onboarding and provides no clear error message when omitted — the same silent 400 failure would occur.

### Consequences

#### Positive
- External detector path now behaves identically to the inline path with respect to model resolution, eliminating the HTTP 400 errors.
- The model resolution order is explicit and deterministic in code, making it auditable without inspecting org-level variables.

#### Negative
- The model-resolution logic is now duplicated between the inline and external builder functions. A future change to the resolution priority in one path must be manually mirrored in the other, creating a maintenance surface.
- The fix is narrowly scoped to model propagation; any other fields that diverge between the two paths remain undiscovered until they also produce failures.

#### Neutral
- A new test (`TestExternalDetectorPropagatesModel`) covers four scenarios: main model inheritance, model mappings, detection-specific override, and default AI credits pricing propagation. This raises test coverage for the external path to match existing inline-path tests.
- `detection_runs_comment.md` was reformatted as a Markdown table as part of this PR (cosmetic change only).

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
