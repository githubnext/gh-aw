# ADR-48107: Propagate `models.providers` Pricing into AWF `apiProxy` Config

**Date**: 2026-07-26
**Status**: Draft
**Deciders**: Unknown

---

### Context

The AWF runtime enforces AI-credits consumption through an API proxy guardrail that resolves per-model pricing at request time. Pricing data for built-in models is embedded in the proxy's static catalog. When users configure custom or BYOK models (e.g., via `models.providers` overrides), those pricing entries are registered only in the agent-side catalog; the API proxy evaluates a separately generated `awf-config.json` that contains no provider pricing, causing every custom-model request to fail at turn 1 with `unknown_model_ai_credits`. The same data flow gap exists for threat-detection runs, which construct their own `WorkflowData` from the primary run's data without propagating `ModelCosts`.

### Decision

We will embed `models.providers` pricing overlays from `WorkflowData.ModelCosts` into the generated `awf-config.json` under the `apiProxy.providers` key, and propagate `ModelCosts` into the `WorkflowData` constructed for threat-detection runs. This makes the API proxy's pricing resolver self-contained within the AWF config it receives, removing the dependency on a separate catalog lookup at proxy evaluation time.

### Alternatives Considered

#### Alternative 1: Shared Pricing Service / Catalog Query at Proxy Evaluation Time

The API proxy could query a shared in-process or network pricing service at request evaluation time rather than reading pricing from the config. This would decouple pricing data from the serialized config artifact.

Not chosen because it would require a new inter-component dependency (the proxy calling back into the agent process or an external service), adding latency on every guarded request and significant complexity to both the proxy and the agent runtime. The config-embedding approach requires no new interfaces.

#### Alternative 2: Default Fallback Cost for Unknown Models

The API proxy could apply a configurable default cost for any model not found in its pricing table, rather than failing with `unknown_model_ai_credits`.

Not chosen because a silent fallback would allow under-accounting of AI-credit consumption for BYOK models — a correctness and auditability concern. Users configure explicit pricing precisely to get accurate accounting. Propagating actual pricing preserves correctness without requiring any policy decisions about default values.

### Consequences

#### Positive
- Custom and BYOK models resolve pricing correctly at the API proxy layer, eliminating `unknown_model_ai_credits` failures for all runs using `models.providers` overrides.
- Threat-detection runs gain pricing parity with primary agent runs, ensuring that detection phases do not fail or under-account when a custom model is configured.
- The AWF config is self-contained: all information the proxy needs to evaluate a request is present in the config artifact, simplifying debugging and offline replay.
- A focused test suite regression-guards both paths (main `BuildAWFConfigJSON` and detection step generation).

#### Negative
- The `apiProxy.providers` field is typed as `map[string]any`, offering no compile-time validation of pricing structure; malformed cost entries (e.g., non-numeric strings) would be forwarded to the proxy silently.
- Each `awf-config.json` artifact now embeds a copy of the provider pricing catalog, increasing config size in proportion to the number of custom models configured.

#### Neutral
- The AWF config JSON schema (`awf-config.schema.json`) is extended to allow `apiProxy.providers` with `additionalProperties: true`, preventing schema-validation rejection of the new field while intentionally not constraining the nested pricing shape.
- Existing runs with no `models.providers` configuration are unaffected: `extractModelCostProviders` returns `nil` when `ModelCosts` is empty or has no `providers` key.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
