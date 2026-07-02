package cli

import (
	"context"
	_ "embed"
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/modelsdev"
)

var modelCostsLog = logger.New("cli:model_costs")

//go:embed data/models.json
var modelsJSON []byte

type modelsCatalogData struct {
	Providers map[string]modelsCatalogProvider `json:"providers"`
}

type modelsCatalogProvider struct {
	Models map[string]modelCostEntry `json:"models"`
}

type modelCostEntry struct {
	Cost map[string]string `json:"cost"`
}

type modelPriceRecord struct {
	id       string
	provider string
	model    string
	pricing  map[string]float64
}

var (
	modelPriceRecords []modelPriceRecord
	modelPricesOnce   sync.Once
)

func initModelPrices() {
	modelPricesOnce.Do(func() {
		var data modelsCatalogData
		if err := json.Unmarshal(modelsJSON, &data); err != nil {
			return
		}

		modelPriceRecords = make([]modelPriceRecord, 0)
		for providerName, providerData := range data.Providers {
			normalizedProvider := strings.ToLower(strings.TrimSpace(providerName))
			if normalizedProvider == "" { //nolint:tolowerequalfold
				continue
			}
			for modelName, entry := range providerData.Models {
				normalizedModel := strings.ToLower(strings.TrimSpace(modelName))
				if normalizedModel == "" { //nolint:tolowerequalfold
					continue
				}
				normalizedID := normalizedProvider + "/" + normalizedModel
				record := modelPriceRecord{
					id:       normalizedID,
					provider: normalizedProvider,
					model:    normalizedModel,
					pricing:  make(map[string]float64, len(entry.Cost)),
				}
				for key, value := range entry.Cost {
					if parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
						record.pricing[key] = parsed
					}
				}
				modelPriceRecords = append(modelPriceRecords, record)
			}
		}
		modelCostsLog.Printf("Initialized model price catalog: providers=%d, records=%d", len(data.Providers), len(modelPriceRecords))
	})
}

func findModelPricing(provider, model string) (map[string]float64, bool) {
	initModelPrices()

	normalizedProvider := normalizeCatalogProvider(provider)
	normalizedModel := strings.ToLower(strings.TrimSpace(model))
	comparableModel := normalizeComparableModelID(normalizedModel)
	if normalizedModel == "" { //nolint:tolowerequalfold
		return nil, false
	}

	fullID := normalizedModel
	if !strings.Contains(fullID, "/") && normalizedProvider != "" {
		fullID = normalizedProvider + "/" + normalizedModel
	}
	comparableFullID := normalizeComparableModelID(fullID)

	for _, record := range modelPriceRecords {
		if (fullID != "" && record.id == fullID) || (comparableFullID != "" && normalizeComparableModelID(record.id) == comparableFullID) {
			modelCostsLog.Printf("Exact pricing match: provider=%s, model=%s -> %s", provider, model, record.id)
			return record.pricing, true
		}
	}

	var bestProviderScoped map[string]float64
	bestProviderScopedLen := -1
	var bestGeneric map[string]float64
	bestGenericLen := -1

	for _, record := range modelPriceRecords {
		comparableRecordModel := normalizeComparableModelID(record.model)
		if record.model == normalizedModel || comparableRecordModel == comparableModel {
			if normalizedProvider != "" && record.provider == normalizedProvider {
				return record.pricing, true
			}
			if bestGeneric == nil {
				bestGeneric = record.pricing
			}
			continue
		}

		if strings.HasPrefix(normalizedModel, record.model) || strings.HasPrefix(comparableModel, comparableRecordModel) {
			if normalizedProvider != "" && record.provider == normalizedProvider && len(record.model) > bestProviderScopedLen {
				bestProviderScoped = record.pricing
				bestProviderScopedLen = len(record.model)
			}
			if len(record.model) > bestGenericLen {
				bestGeneric = record.pricing
				bestGenericLen = len(record.model)
			}
		}
	}

	if bestProviderScoped != nil {
		modelCostsLog.Printf("Provider-scoped prefix pricing match: provider=%s, model=%s", provider, model)
		return bestProviderScoped, true
	}
	if bestGeneric != nil {
		modelCostsLog.Printf("Generic prefix pricing match: provider=%s, model=%s", provider, model)
		return bestGeneric, true
	}
	modelCostsLog.Printf("No pricing match: provider=%s, model=%s", provider, model)
	return nil, false
}

func normalizeCatalogProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "github", "copilot", "github_models":
		return "github-copilot"
	default:
		return strings.ToLower(strings.TrimSpace(provider))
	}
}

func normalizeComparableModelID(value string) string {
	return strings.NewReplacer(".", "-", "_", "-").Replace(strings.ToLower(strings.TrimSpace(value)))
}

func usdToAIC(usd float64) float64 {
	return usd / 0.01
}

func computeModelInferenceCostUSD(provider, model string, inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens, reasoningTokens int) float64 {
	pricing, ok := findModelPricing(provider, model)
	if !ok {
		return 0
	}

	input := inputTokens
	cacheRead := cacheReadTokens
	if cacheRead > 0 && providerIncludesCacheReadsInInput(strings.ToLower(strings.TrimSpace(provider))) {
		input = max(inputTokens-cacheReadTokens, 0)
	}

	promptPrice := pricing["input"]
	completionPrice := pricing["output"]
	cacheReadPrice := pricing["cache_read"]
	if cacheReadPrice == 0 {
		cacheReadPrice = promptPrice
	}
	cacheWritePrice := pricing["cache_write"]
	if cacheWritePrice == 0 {
		cacheWritePrice = promptPrice
	}
	reasoningPrice := pricing["reasoning"]
	if reasoningPrice == 0 {
		reasoningPrice = completionPrice
	}

	return float64(input)*promptPrice +
		float64(outputTokens)*completionPrice +
		float64(cacheRead)*cacheReadPrice +
		float64(cacheWriteTokens)*cacheWritePrice +
		float64(reasoningTokens)*reasoningPrice
}

func computeModelInferenceAIC(provider, model string, inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens, reasoningTokens int) float64 {
	return usdToAIC(computeModelInferenceCostUSD(provider, model, inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens, reasoningTokens))
}

// FindOrFetchModelPricing resolves per-token pricing for the given provider/model.
// It checks the embedded catalog first; for models absent from the embedded catalog it
// attempts to download the models.dev catalog (at most once per process) and queries
// that. Returns (nil, false) when the model is found in the embedded catalog (runtime
// already has it) or when no pricing can be resolved.
//
// This function is intended for compile-time use: the workflow compiler calls it to
// inject pricing for unknown models into GH_AW_INFO_MODEL_COSTS in the compiled lock.yml
// so that the agent job can perform cost accounting without a live catalog download.
func FindOrFetchModelPricing(ctx context.Context, provider, model string) (map[string]float64, bool) {
	if _, ok := findModelPricing(provider, model); ok {
		// Model is already in the embedded catalog; the runtime actions/setup/js/models.json
		// will supply the pricing — no need to inject it into the lock.yml overlay.
		return nil, false
	}
	return modelsdev.FindPricing(ctx, provider, model)
}
