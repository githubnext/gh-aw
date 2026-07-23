# ADR-47571: Propagate models.providers Custom Pricing to API Proxy defaultAiCreditsPricing

**Date**: 2026-07-23
**Status**: Draft
**Deciders**: Unknown

---

### Context

Workflows that configure custom or BYOK (Bring Your Own Key) models via the `models.providers` frontmatter field supply per-token pricing for models that are not in the AWF API proxy's built-in pricing table. Before this fix, that pricing data was parsed into `WorkflowData.ModelCosts` on the agent side but was never forwarded to `awf-config.json`. As a result, the AWF API proxy would reject any AI credit budget check for such models with an HTTP 400 error (`unknown_model_ai_credits`), blocking both the main workflow agent and any threat-detection sub-agents from running at all. The fix was needed because per-token credit enforcement is active whenever `maxAiCredits` is set, and there was no other way for end-users to unblock BYOK models without modifying the AWF built-in pricing table.

### Decision

We will extract the first usable per-model pricing entry from `WorkflowData.ModelCosts["providers"]` during `BuildAWFConfigJSON` and populate `apiProxy.defaultAiCreditsPricing` in `awf-config.json` with that entry, converted from per-token to per-million-token units (multiplied by 1e6). Model selection prefers an exact case-insensitive match on the workflow's `model:` field, falling back to the first parseable entry found across all providers. The `buildThreatDetectionWorkflowData` helper is also updated to copy `ModelCosts` from the parent `WorkflowData` so the detection sub-agent inherits the same pricing fallback.

### Alternatives Considered

#### Alternative 1: Add every custom model to the AWF built-in pricing table

Custom/BYOK model identifiers would be submitted to the AWF team (or via a config file owned by that team) and added to the static table. This eliminates the runtime extraction logic but requires cross-team coordination for every new model, a process with uncertain latency. It also does not scale to BYOK scenarios where the model identifier is user-defined and not known in advance. Not chosen because the coordination overhead is high and the approach is not self-serve.

#### Alternative 2: Disable AI credit enforcement for unrecognised models

The API proxy could be configured to skip credit tracking when the model is not in the built-in table, allowing requests through unconditionally. This avoids the extraction logic but silently removes the credit budget safety net for custom models, creating an unbounded spend risk for BYOK use cases. Not chosen because it undermines the purpose of `maxAiCredits`.

#### Alternative 3: Accept all pricing entries and build a full per-model override map

Rather than selecting a single `defaultAiCreditsPricing` entry, the proxy could receive a full `{"modelId": {pricing}}` map. This is more accurate when multiple custom models with different pricing are used in one workflow. Not chosen at this time because the AWF schema only supports a single `defaultAiCreditsPricing` object (not a per-model map), so this would require an AWF schema change first.

### Consequences

#### Positive
- Custom/BYOK models configured via `models.providers` frontmatter no longer fail with `unknown_model_ai_credits`; they are priced using the declared per-token rates.
- The threat-detection sub-agent inherits the same pricing fallback, making the fix consistent across both the main agent run and the detection guardrail.
- The extraction logic is entirely on the agent side and requires no AWF schema changes beyond the already-supported `defaultAiCreditsPricing` field.
- Comprehensive unit tests cover nil inputs, string and float64 cost values, exact-model-match preference, and fallback behaviour.

#### Negative
- `defaultAiCreditsPricing` is a single fallback value, not a per-model table. If a workflow declares multiple custom models with different pricing, only the pricing for the workflow's primary `model:` field is forwarded (or the first parseable entry if `model:` is unset). Other custom models use the same fallback rate, which may be inaccurate.
- The second-pass fallback iterates over Go maps, whose iteration order is not guaranteed. In workflows with multiple custom models and no `model:` field set, the selected pricing entry is non-deterministic between runs.
- Pricing values from the frontmatter are trusted verbatim; there is no server-side validation that they match the actual provider's published rates.

#### Neutral
- The per-token-to-per-million conversion (`× 1e6`) is explicit in `parseCostFieldToPerMillion`, which makes the unit boundary visible in the code but adds a conversion step that must be kept in sync with any future AWF schema changes.
- Lock file workflows (`daily-model-inventory.lock.yml`, `smoke-copilot.lock.yml`) are regenerated to include a hardcoded `defaultAiCreditsPricing` in their embedded JSON; these will need to be regenerated again whenever the pricing defaults change.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
