package workflow

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var compilerModelPricingLog = logger.New("workflow:compiler_model_pricing")

// resolveModelPricingIfMissing checks whether the workflow's configured model has pricing
// in the frontmatter ModelCosts overlay. When pricing is absent it calls
// c.modelPricingResolver (injected by the cli package) to fetch pricing from the
// models.dev catalog and merges the result into ModelCosts so it is serialised into
// GH_AW_INFO_MODEL_COSTS in the compiled lock.yml.
//
// Frontmatter-provided pricing always takes precedence; models already present in the
// embedded actions/setup/js/models.json are skipped by the resolver (the runtime will
// supply their pricing without an override).
func (c *Compiler) resolveModelPricingIfMissing(modelCosts map[string]any, engineConfig *EngineConfig) map[string]any {
	if c.modelPricingResolver == nil {
		return modelCosts
	}
	if engineConfig == nil || engineConfig.Model == "" {
		return modelCosts
	}

	provider, model, ok := resolveProviderAndModelForPricing(engineConfig)
	if !ok {
		compilerModelPricingLog.Printf("Skipping external pricing lookup: unable to normalize provider/model for %q", engineConfig.Model)
		return modelCosts
	}

	// If the frontmatter overlay already supplies pricing for this model, leave it intact.
	if modelCostsHasPricingFor(modelCosts, provider, model) {
		compilerModelPricingLog.Printf("Pricing already in ModelCosts for %s/%s — skipping models.dev lookup", provider, model)
		return modelCosts
	}

	ctx := c.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	pricing, ok := c.modelPricingResolver(ctx, provider, model)
	if !ok || len(pricing) == 0 {
		compilerModelPricingLog.Printf("No external pricing found for model %q (provider=%q) — cost accounting may be unavailable", model, provider)
		return modelCosts
	}

	compilerModelPricingLog.Printf("Resolved pricing for %s/%s from models.dev — injecting into lock.yml GH_AW_INFO_MODEL_COSTS", provider, model)
	return mergeModelPricingIntoModelCosts(modelCosts, provider, model, pricing)
}

// resolveEngineProviderForPricing returns the inference provider string to use for pricing
// lookup. It checks the EngineConfig fields in priority order and falls back to a
// well-known mapping from engine ID.
func resolveEngineProviderForPricing(engineConfig *EngineConfig) string {
	if engineConfig.LLMProvider != "" {
		return normalizeProviderForPricing(engineConfig.LLMProvider)
	}
	if engineConfig.InlineProviderID != "" {
		return normalizeProviderForPricing(engineConfig.InlineProviderID)
	}
	// Infer provider from the built-in engine ID.
	switch strings.ToLower(strings.TrimSpace(engineConfig.ID)) {
	case "claude":
		return "anthropic"
	case "codex":
		return "openai"
	case "copilot", "":
		return "github-copilot"
	default:
		return ""
	}
}

func normalizeProviderForPricing(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "github", "copilot", "github_models":
		return "github-copilot"
	default:
		return strings.ToLower(strings.TrimSpace(provider))
	}
}

func resolveProviderAndModelForPricing(engineConfig *EngineConfig) (string, string, bool) {
	provider := resolveEngineProviderForPricing(engineConfig)
	model := strings.ToLower(strings.TrimSpace(engineConfig.Model))
	if model == "" {
		return "", "", false
	}

	if strings.Contains(model, "/") {
		parts := strings.SplitN(model, "/", 2)
		if parts[0] == "" || parts[1] == "" {
			return "", "", false
		}
		embeddedProvider := normalizeProviderForPricing(parts[0])
		embeddedModel := strings.TrimSpace(parts[1])
		provider = embeddedProvider
		model = strings.ToLower(embeddedModel)
	}

	if provider == "" || model == "" {
		return "", "", false
	}
	return provider, model, true
}

// modelCostsHasPricingFor reports whether the ModelCosts overlay already contains a models
// entry for the given provider/model. An empty provider matches any provider entry.
func modelCostsHasPricingFor(modelCosts map[string]any, provider, model string) bool {
	if len(modelCosts) == 0 {
		return false
	}
	rawProviders, ok := modelCosts["providers"]
	if !ok {
		return false
	}
	providersMap, ok := rawProviders.(map[string]any)
	if !ok {
		return false
	}
	for pName, pData := range providersMap {
		if provider != "" && !strings.EqualFold(pName, provider) {
			continue
		}
		pMap, ok := pData.(map[string]any)
		if !ok {
			continue
		}
		rawModels, ok := pMap["models"]
		if !ok {
			continue
		}
		modelsMap, ok := rawModels.(map[string]any)
		if !ok {
			continue
		}
		for mName := range modelsMap {
			if strings.EqualFold(mName, model) {
				return true
			}
		}
	}
	return false
}

// mergeModelPricingIntoModelCosts builds (or extends) the ModelCosts overlay map with a
// pricing entry for the given provider/model. Returns a new map to avoid mutating the
// input. Per-token float64 pricing values are serialized as decimal strings to match the
// models.json schema expected by merge_frontmatter_models.cjs at runtime.
func mergeModelPricingIntoModelCosts(modelCosts map[string]any, provider, model string, pricing map[string]float64) map[string]any {
	// Serialise float64 per-token prices to strings (models.json cost format).
	cost := make(map[string]string, len(pricing))
	for key, val := range pricing {
		cost[key] = fmt.Sprintf("%g", val)
	}
	modelEntry := map[string]any{"cost": cost}

	// Shallow-clone the top-level map.
	result := make(map[string]any, len(modelCosts)+1)
	maps.Copy(result, modelCosts)

	// Shallow-clone the providers map.
	providers := make(map[string]any)
	if rawProviders, ok := result["providers"]; ok {
		if pm, ok := rawProviders.(map[string]any); ok {
			maps.Copy(providers, pm)
		}
	}

	// Shallow-clone the target provider entry.
	providerEntry := make(map[string]any)
	if rawProvider, ok := providers[provider]; ok {
		if pm, ok := rawProvider.(map[string]any); ok {
			maps.Copy(providerEntry, pm)
		}
	}

	// Shallow-clone the models sub-map and insert the new entry.
	modelsMap := make(map[string]any)
	if rawModels, ok := providerEntry["models"]; ok {
		if mm, ok := rawModels.(map[string]any); ok {
			maps.Copy(modelsMap, mm)
		}
	}
	modelsMap[model] = modelEntry
	providerEntry["models"] = modelsMap
	providers[provider] = providerEntry
	result["providers"] = providers

	return result
}
