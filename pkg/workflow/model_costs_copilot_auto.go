package workflow

import (
	"maps"
	"strings"
)

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
// HTTP 400 (unknown_model_ai_credits) whenever maxAiCredits is active, which breaks the
// zero-config path (engine: copilot with no model:, which sends "auto"). AWF normally
// accounts against the concrete response model. If Copilot omits it, AWF falls back to
// this request-model rate, so the rate matches the most expensive model in the bundled
// Copilot catalog rather than risking under-accounting.
var copilotAutoCost = map[string]string{
	"input":       "1e-05",
	"input_audio": "1.5e-06",
	"output":      "5e-05",
	"cache_read":  "1e-06",
	"cache_write": "1.25e-05",
}

// shouldInjectCopilotAutoPricing limits the builtin overlay to Copilot workflows whose
// configured model can resolve to auto, on AWF versions that pass apiProxy.providers.
func shouldInjectCopilotAutoPricing(config AWFCommandConfig, firewallConfig *FirewallConfig) bool {
	if !strings.EqualFold(config.EngineName, "copilot") || !awfSupportsAPIProxyProviders(firewallConfig) {
		return false
	}
	if config.WorkflowData == nil {
		return true
	}
	model := strings.TrimSpace(config.WorkflowData.Model)
	return model == "" ||
		strings.EqualFold(model, "auto") ||
		strings.EqualFold(model, "copilot/auto") ||
		containsExpression(model)
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
