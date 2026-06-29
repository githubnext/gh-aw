# ADR-42351: Map Copilot SDK Provider Type via Multi-Level Inference Instead of Hardcoding "openai"

**Date**: 2026-06-29
**Status**: Draft
**Deciders**: pelikhan (PR author), copilot-swe-agent (implementation)

---

### Context

The GitHub Copilot SDK's `ProviderConfig.type` field controls which wire format the SDK uses when communicating with a BYOK (Bring Your Own Key) endpoint: `"anthropic"` for the Anthropic Messages API, `"azure"` for Azure OpenAI, or `"openai"` for OpenAI-compatible APIs. The harness previously hardcoded `type: "openai"` for all BYOK endpoints, which caused silent failures when users configured Anthropic API endpoints — the SDK was sending OpenAI-format requests to an Anthropic API, breaking all Anthropic BYOK workflows. A reliable way to determine the correct provider type from existing endpoint and model metadata was needed, without requiring additional user configuration.

### Decision

We will infer the Copilot SDK provider type using a four-level resolution cascade implemented in `inferProviderTypeForModel(endpointProvider, modelName, modelsJson)`: (1) map the endpoint's `provider` field directly (`"anthropic"` → `"anthropic"`, `"azure*"` → `"azure"`, `"openai"` → `"openai"`); (2) look up the model in the `models.json` catalog's `provider_type` field; (3) apply well-known model name heuristics (`claude-*`, `-opus`/`-haiku`/`-sonnet` suffix → `"anthropic"`; `gpt-*`, `o1`/`o3`/`o4` → `"openai"`); (4) default to `"openai"`. The resolved type is passed from the harness to the SDK driver via the `GH_AW_COPILOT_SDK_PROVIDER_TYPE` environment variable.

### Alternatives Considered

#### Alternative 1: Require Explicit User Configuration

Users would set a new environment variable (e.g. `GH_AW_COPILOT_SDK_PROVIDER_TYPE`) to specify the provider type manually, with no inference logic in the harness. This is simple and unambiguous but places the burden on every user who configures a BYOK endpoint, making Anthropic BYOK setups require an extra manual step that is easy to overlook and produces confusing failures when omitted.

#### Alternative 2: Binary Anthropic/Non-Anthropic Detection Based on Endpoint Name Only

Detect only whether the endpoint provider is `"anthropic"` (from the reflect data's `provider` field), and fall back to `"openai"` for all other cases, without catalog lookup or model name heuristics. This fixes the immediate BYOK Anthropic breakage but does not handle Azure endpoints correctly, misses Anthropic models served through a generic `"copilot"` endpoint (e.g. Claude served via GitHub Copilot routing), and requires the `models.json` catalog to be expanded later anyway.

### Consequences

#### Positive
- Anthropic BYOK workflows now correctly use the Anthropic Messages API wire format, fixing the broken use case that motivated this change.
- Azure OpenAI endpoints are now handled correctly without additional user configuration.
- The four-level resolution order is documented and extensible: new providers can be supported by updating the endpoint name mapping, the `models.json` catalog, or the heuristics without requiring environment changes from users.
- The `models.json` catalog gains a `provider_type` field per model entry, creating a single authoritative source of truth for provider-to-model mapping.

#### Negative
- Model name heuristics (level 3) are inherently fragile: new Anthropic or OpenAI models with non-standard naming conventions will fall through to the `"openai"` default and silently use the wrong wire format until the catalog or heuristics are updated.
- The four-level cascade increases the cognitive complexity of provider resolution; a future developer debugging a wrong `ProviderConfig.type` must trace through all four levels to find the active rule.
- The `GH_AW_COPILOT_SDK_PROVIDER_TYPE` environment variable coupling between the harness and the driver is an implicit contract that is not validated by type-checking at the boundary.

#### Neutral
- A new `loadModelsJson()` function in `model_costs.cjs` provides cached raw-JSON access to the catalog, which may be reused by other callers needing direct catalog access beyond the cost-computation API.
- The resolved provider type is now logged in the harness and driver startup messages, improving observability for BYOK debugging.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
