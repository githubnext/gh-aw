package cli

// This file provides command-line interface functionality for gh-aw.
// This file (effective_tokens.go) implements the Effective Tokens (ET) specification
// defined in docs/src/content/docs/reference/effective-tokens-specification.md.
//
// Effective Tokens normalize raw token counts across token classes and model pricing
// using the formula:
//
//	base_weighted_tokens = (w_in × I) + (w_cache × C) + (w_out × O) + (w_reason × R)
//	effective_tokens     = m × base_weighted_tokens
//
// where:
//   - I  = input tokens         (w_in    = 1.0 default)
//   - C  = cached input tokens  (w_cache = 0.1 default)
//   - O  = output tokens        (w_out   = 4.0 default)
//   - R  = reasoning tokens     (w_reason = 4.0 default)
//   - m  = per-model multiplier relative to the reference model
//
// Token class weights and model multipliers are loaded from the embedded
// data/model_multipliers.json file and can be updated without recompilation.
//
// Key responsibilities:
//   - Embedding model_multipliers.json at compile time
//   - Applying token class weights before the model multiplier
//   - Computing effective tokens from raw per-model token usage data
//   - Populating effective token counts on TokenUsageSummary after parsing

import (
	_ "embed"
	"encoding/json"
	"maps"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/types"
)

var effectiveTokensLog = logger.New("cli:effective_tokens")

//go:embed data/model_multipliers.json
var modelMultipliersJSON []byte

const (
	defaultMergedModelMultipliersPath = "/tmp/gh-aw/model_multipliers.json"
	mergedModelMultipliersPathEnvVar  = "GH_AW_MERGED_MODEL_MULTIPLIERS_PATH"
	modelMultipliersEnvVar            = "GH_AW_MODEL_MULTIPLIERS"
)

// modelMultipliersData is the top-level structure of model_multipliers.json.
type modelMultipliersData struct {
	Version           string                  `json:"version"`
	Description       string                  `json:"description"`
	ReferenceModel    string                  `json:"reference_model"`
	TokenClassWeights types.TokenClassWeights `json:"token_class_weights"`
	Multipliers       map[string]float64      `json:"multipliers"`
}

// loadedMultipliers is the parsed multiplier table, keyed by lowercase model name.
// Initialized once on first call to effectiveTokenMultiplier.
var loadedMultipliers map[string]float64

// loadedTokenWeights holds the token class weights from the JSON file.
// Initialized once on first call to initMultipliers.
var loadedTokenWeights types.TokenClassWeights

// initMultipliers parses the embedded JSON and populates loadedMultipliers and
// loadedTokenWeights. Safe to call multiple times; only initializes once.
func initMultipliers() {
	if loadedMultipliers != nil {
		return
	}

	data, ok := loadModelMultipliersData()
	if !ok {
		effectiveTokensLog.Print("Failed to load model multipliers from all sources; falling back to defaults")
		loadedMultipliers = make(map[string]float64)
		loadedTokenWeights = defaultTokenClassWeights()
		return
	}

	loadedMultipliers = make(map[string]float64, len(data.Multipliers))
	for model, mult := range data.Multipliers {
		loadedMultipliers[strings.ToLower(model)] = mult
	}

	// Fall back to default weights for any zero-valued field (zero means not set)
	defaults := defaultTokenClassWeights()
	loadedTokenWeights = data.TokenClassWeights
	if loadedTokenWeights.Input == 0 {
		loadedTokenWeights.Input = defaults.Input
	}
	if loadedTokenWeights.CachedInput == 0 {
		loadedTokenWeights.CachedInput = defaults.CachedInput
	}
	if loadedTokenWeights.Output == 0 {
		loadedTokenWeights.Output = defaults.Output
	}
	if loadedTokenWeights.Reasoning == 0 {
		loadedTokenWeights.Reasoning = defaults.Reasoning
	}
	if loadedTokenWeights.CacheWrite == 0 {
		loadedTokenWeights.CacheWrite = defaults.CacheWrite
	}

	effectiveTokensLog.Printf("Loaded %d model multipliers (reference: %s, w_in=%.1f w_cache=%.1f w_out=%.1f)",
		len(loadedMultipliers), data.ReferenceModel,
		loadedTokenWeights.Input, loadedTokenWeights.CachedInput, loadedTokenWeights.Output)
}

func loadModelMultipliersData() (modelMultipliersData, bool) {
	mergedPath := strings.TrimSpace(os.Getenv(mergedModelMultipliersPathEnvVar))
	if mergedPath == "" {
		mergedPath = defaultMergedModelMultipliersPath
	}
	if data, ok := parseModelMultipliersFile(mergedPath); ok {
		effectiveTokensLog.Printf("Loaded model multipliers from file: %s", mergedPath)
		return data, true
	}

	if raw := strings.TrimSpace(os.Getenv(modelMultipliersEnvVar)); raw != "" {
		if data, ok := parseModelMultipliersJSON([]byte(raw)); ok {
			effectiveTokensLog.Printf("Loaded model multipliers from env var: %s", modelMultipliersEnvVar)
			return data, true
		}
		effectiveTokensLog.Printf("Env var %s contained invalid JSON; falling back to built-in multipliers", modelMultipliersEnvVar)
	}

	data, ok := parseModelMultipliersJSON(modelMultipliersJSON)
	if ok {
		effectiveTokensLog.Print("Loaded model multipliers from embedded defaults")
	}
	return data, ok
}

func parseModelMultipliersFile(filePath string) (modelMultipliersData, bool) {
	cleanPath := filepath.Clean(filePath)
	raw, err := os.ReadFile(cleanPath)
	if err != nil {
		return modelMultipliersData{}, false
	}
	return parseModelMultipliersJSON(raw)
}

func parseModelMultipliersJSON(raw []byte) (modelMultipliersData, bool) {
	var data modelMultipliersData
	if err := json.Unmarshal(raw, &data); err != nil {
		effectiveTokensLog.Printf("Failed to parse model multipliers JSON: %v", err)
		return modelMultipliersData{}, false
	}
	return data, true
}

// defaultTokenClassWeights returns the specification-mandated default weights.
func defaultTokenClassWeights() types.TokenClassWeights {
	return types.TokenClassWeights{
		Input:       1.0,
		CachedInput: 0.1,
		Output:      4.0,
		Reasoning:   4.0,
		CacheWrite:  1.0,
	}
}

// populateEffectiveTokensWithCustomWeights is like populateEffectiveTokens but
// merges custom into the built-in weights before computing effective tokens.
// Custom weights take precedence over the defaults loaded from model_multipliers.json.
// It is a no-op when summary is nil.
func populateEffectiveTokensWithCustomWeights(summary *TokenUsageSummary, custom *types.TokenWeights) {
	if summary == nil {
		return
	}

	multipliers, classWeights := resolveEffectiveWeights(custom)

	total := 0
	for model, usage := range summary.ByModel {
		if usage == nil {
			continue
		}
		eff := computeModelEffectiveTokensWithWeights(model, usage.Provider, usage.InputTokens, usage.OutputTokens,
			usage.CacheReadTokens, usage.CacheWriteTokens, usage.ReasoningTokens, multipliers, classWeights)
		usage.EffectiveTokens = eff
		total += eff
	}
	summary.TotalEffectiveTokens = total

	if effectiveTokensLog.Enabled() {
		effectiveTokensLog.Printf("Effective tokens: total=%d models=%d custom=%v", total, len(summary.ByModel), custom != nil)
	}
}

// resolveEffectiveWeights merges optional custom weights with the built-in defaults.
// The returned multipliers map is a copy so callers may not modify loadedMultipliers.
func resolveEffectiveWeights(custom *types.TokenWeights) (map[string]float64, types.TokenClassWeights) {
	initMultipliers()

	// Copy the base multipliers to avoid mutating the shared global
	merged := make(map[string]float64, len(loadedMultipliers))
	maps.Copy(merged, loadedMultipliers)
	classWeights := loadedTokenWeights

	if custom == nil {
		return merged, classWeights
	}

	// Override/add per-model multipliers (normalise keys to lowercase)
	for model, mult := range custom.Multipliers {
		merged[strings.ToLower(strings.TrimSpace(model))] = mult
	}

	// Override per-token-class weights where non-zero values are provided
	if tcw := custom.TokenClassWeights; tcw != nil {
		if tcw.Input != 0 {
			classWeights.Input = tcw.Input
		}
		if tcw.CachedInput != 0 {
			classWeights.CachedInput = tcw.CachedInput
		}
		if tcw.Output != 0 {
			classWeights.Output = tcw.Output
		}
		if tcw.Reasoning != 0 {
			classWeights.Reasoning = tcw.Reasoning
		}
		if tcw.CacheWrite != 0 {
			classWeights.CacheWrite = tcw.CacheWrite
		}
	}

	return merged, classWeights
}

// computeModelEffectiveTokensWithWeights computes effective tokens using caller-provided
// multiplier table and token class weights instead of the global defaults.
func computeModelEffectiveTokensWithWeights(model, provider string, inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens, reasoningTokens int, multipliers map[string]float64, w types.TokenClassWeights) int {
	base := computeBaseWeightedTokensForUsage(w, provider, inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens, reasoningTokens)
	if base == 0 {
		return 0
	}

	mult := getModelMultiplier(model, multipliers)
	return int(math.Round(base * mult))
}

func getModelMultiplier(model string, multipliers map[string]float64) float64 {
	key := strings.ToLower(strings.TrimSpace(model))
	if key == "" {
		return 1.0
	}
	if m, ok := multipliers[key]; ok {
		return m
	}

	best := ""
	bestMultiplier := 1.0
	for name, m := range multipliers {
		if strings.HasPrefix(key, name) && len(name) > len(best) {
			best = name
			bestMultiplier = m
		}
	}
	return bestMultiplier
}

// computeBaseWeightedTokensForUsage computes ET base tokens for one usage row.
// The provider parameter controls whether cache reads should be deducted from
// input first (bundled semantics) or left additive (additive semantics).
func computeBaseWeightedTokensForUsage(w types.TokenClassWeights, provider string, inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens, reasoningTokens int) float64 {
	normalizedProvider := strings.ToLower(strings.TrimSpace(provider))
	effectiveInput := inputTokens
	if cacheReadTokens > 0 && providerIncludesCacheReadsInInput(normalizedProvider) {
		// Providers like Anthropic/OpenAI report cache_read_tokens as part of input_tokens.
		// Deduct once so cache reads are weighted only by w.CachedInput, not double-counted.
		// This may change ET versus previously stored effective_tokens values when older
		// runs were computed with the legacy behavior that always treated input_tokens as
		// fully non-cached input while also adding cache_read_tokens as a separate class.
		// Recomputing from raw usage intentionally applies current ET semantics.
		effectiveInput = max(inputTokens-cacheReadTokens, 0)
	}

	return w.Input*float64(effectiveInput) +
		w.CachedInput*float64(cacheReadTokens) +
		w.Output*float64(outputTokens) +
		w.Reasoning*float64(reasoningTokens) +
		w.CacheWrite*float64(cacheWriteTokens)
}

func providerIncludesCacheReadsInInput(normalizedProvider string) bool {
	// Cache read accounting is provider-specific:
	// - bundled semantics: cache_read_tokens are already included in input_tokens,
	//   so we subtract once before applying input weight.
	// - additive semantics: cache_read_tokens are separate from input_tokens,
	//   so no subtraction is applied.
	//
	// Known providers currently using bundled semantics are listed below.
	// Unknown non-empty providers default to additive semantics to avoid
	// under-counting input tokens. Empty provider values are treated as bundled
	// semantics for backward compatibility with older usage records that omitted
	// the provider field.
	// We include both "azure-openai" and "azure_openai" to handle observed
	// provider naming variants in historical logs.
	switch normalizedProvider {
	case "", "anthropic", "openai", "azure-openai", "azure_openai":
		return true
	default:
		return false
	}
}
