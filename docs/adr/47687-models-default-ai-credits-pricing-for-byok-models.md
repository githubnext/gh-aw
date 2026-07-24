# ADR-47687: Expose `models.default-ai-credits-pricing` as a Compiler Frontmatter Field for BYOK/Self-Hosted Models

**Date**: 2026-07-24
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The AWF API proxy enforces AI credit accounting via `maxAiCredits`, which is always active by default. When a workflow uses a model not present in the AWF built-in pricing table — such as a self-hosted BYOK Ollama model (`qwen2.5:0.5b`) — the proxy rejects all requests with HTTP 400 `unknown_model_ai_credits`. The `Daily BYOK Ollama Test` had no fallback pricing configured, causing 8+ consecutive daily CI failures. The compiler had no mechanism for workflow authors to supply a fallback pricing rate; the only workaround would have required changes inside the AWF firewall component itself.

### Decision

We will add a `models.default-ai-credits-pricing` frontmatter field to the gh-aw compiler. When present, the compiler populates `apiProxy.defaultAiCreditsPricing` in the generated AWF config JSON with the author-supplied `input` and `output` token rates ($/1M tokens). Self-hosted or free models set both rates to `0`. This gives workflow authors a per-workflow escape hatch for models unrecognized by AWF, without requiring changes to the shared AWF firewall service.

### Alternatives Considered

#### Alternative 1: Add self-hosted models to the AWF built-in pricing table

AWF maintains a centralized pricing table in the firewall service. Adding free/self-hosted models there would remove the need for a per-workflow field. This was not chosen because it requires changes in a separate repository and a firewall service release cycle. The fix would not be scoped to specific workflows; any workflow using an unrecognized model would silently succeed, which could mask mis-configuration.

#### Alternative 2: Disable `maxAiCredits` enforcement for BYOK workflows automatically

The proxy could detect that a workflow uses a BYOK key and skip credit enforcement entirely. This was not chosen because it would require the proxy to understand BYOK semantics, coupling two concerns. It would also silently disable credit accounting for all models in a BYOK workflow, not just the unrecognized ones, making budget enforcement unreliable.

#### Alternative 3: Set a global default pricing rate in the AWF config for unknown models

The `apiProxy.defaultAiCreditsPricing` field already exists in AWF config schema. A global operator-level default could have been set in the runner infrastructure. This was not chosen because it would apply the same fallback rate to all workflows across all tenants, making it impossible for individual workflow authors to specify correct rates for their specific models.

### Consequences

#### Positive
- Fixes the 8+ consecutive `Daily BYOK Ollama Test` failures without requiring a firewall service change or release.
- Workflow authors can now configure per-token pricing for any self-hosted or unrecognized model without needing operator intervention.
- The `input: 0, output: 0` convention clearly expresses "this model is free/local" in the workflow frontmatter.

#### Negative
- Workflow authors can supply incorrect pricing values (e.g., non-zero rates for free models), which will cause AI credit accounting to record costs for usage that is actually free.
- The field is silently ignored when `maxAiCredits` is not active; authors may be confused if the setting has no visible effect in that context.

#### Neutral
- A new required sub-field pair (`input`, `output`) is enforced by JSON schema, so partial configurations are rejected at compile time rather than producing surprising runtime behavior.
- The implementation adds a new `AiCreditsPricingConfig` struct in `pkg/workflow/sandbox.go` and a parallel `AWFDefaultAiCreditsPricingConfig` in `pkg/workflow/awf_config.go`; future changes to pricing semantics must update both.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
