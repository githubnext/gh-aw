package workflow

import "maps"

// copilotAutoPricingProvider is the models.json provider key used by the AWF API proxy
// pricing table for the Copilot inference gateway.
const copilotAutoPricingProvider = "github-copilot"

// copilotAutoPricingModel is the Copilot server-side model selector. It is a valid
// builtin Copilot model id (the "auto" alias resolves to "copilot/auto"), but it names
// no concrete model, so the models.dev catalog mirrored in models.json has no pricing
// entry for it.
const copilotAutoPricingModel = "auto"

// copilotAutoCost is the built-in per-token USD rate published for "copilot/auto".
//
// Without an entry the AWF API proxy rejects every inference request for the model with
// HTTP 400 (missing_model_pricing) whenever maxAiCredits is active, which breaks the
// zero-config path (engine: copilot with no model:, which sends "auto"). Copilot picks
// the served model server-side, so gh-aw publishes the Sonnet-class rate — the tier the
// "large" fallback chain lands on — as the accounting rate for the selector.
var copilotAutoCost = map[string]string{
	"input":       "3e-06",
	"output":      "1.5e-05",
	"cache_read":  "3e-07",
	"cache_write": "3.75e-06",
}

// withCopilotAutoPricing returns the apiProxy.providers overlay with a built-in pricing
// entry for "copilot/auto" added when the workflow does not already supply one through
// models.providers frontmatter. The input map is never mutated; frontmatter pricing
// always wins.
func withCopilotAutoPricing(providers map[string]any) map[string]any {
	if modelCostsHasPricingFor(map[string]any{"providers": providers}, copilotAutoPricingProvider, copilotAutoPricingModel) {
		awfConfigLog.Printf("API proxy: frontmatter already prices %s/%s; skipping builtin overlay", copilotAutoPricingProvider, copilotAutoPricingModel)
		return providers
	}

	result := make(map[string]any, len(providers)+1)
	maps.Copy(result, providers)

	providerEntry := make(map[string]any)
	if existing, ok := result[copilotAutoPricingProvider].(map[string]any); ok {
		maps.Copy(providerEntry, existing)
	}

	models := make(map[string]any)
	if existing, ok := providerEntry["models"].(map[string]any); ok {
		maps.Copy(models, existing)
	}

	cost := make(map[string]any, len(copilotAutoCost))
	for k, v := range copilotAutoCost {
		cost[k] = v
	}
	models[copilotAutoPricingModel] = map[string]any{"cost": cost}
	providerEntry["models"] = models
	result[copilotAutoPricingProvider] = providerEntry

	awfConfigLog.Printf("API proxy: added builtin pricing for %s/%s", copilotAutoPricingProvider, copilotAutoPricingModel)
	return result
}
