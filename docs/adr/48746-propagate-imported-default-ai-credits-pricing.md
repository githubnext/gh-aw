# ADR-48746: Propagate Imported Default AI-Credits Pricing Through Import Aggregation Pipeline

**Date**: 2026-07-29
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Workflows that import shared AWF config (e.g., `.github/workflows/shared/otlp.md`) were compiling without `apiProxy.defaultAiCreditsPricing` in the output, even when the shared import defined `models.default-ai-credits-pricing`. This caused `API Error: 400` failures (surfaced as `missing_model_pricing` / `unknown_model_ai_credits`) across multiple workflows using models such as `claude-opus-5` that lack catalog pricing. The failure was systemic and recurring: per-workflow patches had been applied (e.g., PR #48292) but the same regression kept appearing for newly authored workflows. A repo-wide fallback mechanism was needed that would automatically apply to any workflow importing the shared config.

### Decision

We will extend the parser import result model to carry `default-ai-credits-pricing` extracted from imported config, propagate it through import aggregation (first-wins across all imports), and resolve it in the workflow builder so that the compiled AWF config includes `apiProxy.defaultAiCreditsPricing` whenever the main workflow does not define its own pricing. The shared import `.github/workflows/shared/otlp.md` will define the canonical fallback values (`input: 5.0`, `output: 25.0`).

### Alternatives Considered

#### Alternative 1: Per-Workflow Pricing Patches

Apply `default-ai-credits-pricing` individually to each workflow that encounters the missing-pricing error. This was the approach used before this PR (e.g., PR #48292). It is directly traceable and has no impact on unrelated workflows, but it is not sustainable: each new workflow that imports shared config without its own pricing definition regresses into the same failure, requiring another manual patch. The issue tracker at the time of this ADR listed at least six affected workflows with more expected.

#### Alternative 2: Centralised Static Config File (Non-Import Mechanism)

Define a global defaults config file (e.g., `.github/workflows/shared/pricing-defaults.yml`) that the compiler always applies, independent of imports. This avoids the need to thread pricing through the import aggregation model. However, it introduces a new implicit loading mechanism outside the existing import pipeline, which increases compiler surface area and makes pricing override precedence harder to understand. The import-propagation approach reuses the established import pipeline and makes the relationship between shared config and compiled output explicit.

### Consequences

#### Positive
- Any workflow that imports the shared otlp/shared config automatically inherits the pricing fallback without further manual intervention, eliminating the recurrence pattern seen across at least six workflows.
- The precedence rule (main-workflow pricing overrides imported default) is explicit and tested, making it safe to define per-workflow pricing when needed without breaking the fallback.

#### Negative
- First-wins semantics across multiple imports means that if two imports each define `default-ai-credits-pricing`, only the first encountered value is used and a warning is emitted. Authors of new shared imports must be aware of this behaviour.
- Adding `default-ai-credits-pricing` to `.github/workflows/shared/otlp.md` couples a pricing concern to a telemetry-oriented shared config, which may surprise maintainers of that file.

#### Neutral
- The import result model gains a new optional field; existing imports that do not define `default-ai-credits-pricing` are unaffected.
- New tests cover extraction, first-wins behaviour, invalid-shape warning, and builder resolution precedence, increasing test surface area for the import pipeline.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
