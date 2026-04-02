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
//   - Providing model multiplier lookup with prefix matching for model variants
//   - Computing effective tokens from raw per-model token usage data
//   - Populating effective token counts on TokenUsageSummary after parsing

import (
	_ "embed"
	"encoding/json"
	"maps"
	"math"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/types"
)

var effectiveTokensLog = logger.New("cli:effective_tokens")

//go:embed data/model_multipliers.json
var modelMultipliersJSON []byte

// tokenClassWeights holds the per-token-class weight values from the specification.
type tokenClassWeights struct {
	Input       float64 `json:"input"`
	CachedInput float64 `json:"cached_input"`
	Output      float64 `json:"output"`
	Reasoning   float64 `json:"reasoning"`
	CacheWrite  float64 `json:"cache_write"`
}

// modelMultipliersData is the top-level structure of model_multipliers.json.
type modelMultipliersData struct {
	Version           string             `json:"version"`
	Description       string             `json:"description"`
	ReferenceModel    string             `json:"reference_model"`
	TokenClassWeights tokenClassWeights  `json:"token_class_weights"`
	Multipliers       map[string]float64 `json:"multipliers"`
}

// loadedMultipliers is the parsed multiplier table, keyed by lowercase model name.
// Initialized once on first call to effectiveTokenMultiplier.
var loadedMultipliers map[string]float64

// loadedTokenWeights holds the token class weights from the JSON file.
// Initialized once on first call to initMultipliers.
var loadedTokenWeights tokenClassWeights

// initMultipliers parses the embedded JSON and populates loadedMultipliers and
// loadedTokenWeights. Safe to call multiple times; only initializes once.
func initMultipliers() {
	if loadedMultipliers != nil {
		return
	}

	var data modelMultipliersData
	if err := json.Unmarshal(modelMultipliersJSON, &data); err != nil {
		effectiveTokensLog.Printf("Failed to parse model_multipliers.json: %v", err)
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

// defaultTokenClassWeights returns the specification-mandated default weights.
func defaultTokenClassWeights() tokenClassWeights {
	return tokenClassWeights{
		Input:       1.0,
		CachedInput: 0.1,
		Output:      4.0,
		Reasoning:   4.0,
		CacheWrite:  1.0,
	}
}

// effectiveTokenMultiplier returns the per-model cost multiplier for the given model name.
// Lookup order:
//  1. Exact case-insensitive match
//  2. Longest prefix match (e.g. "claude-sonnet-4.6-preview" → "claude-sonnet-4.6")
//  3. Default: 1.0 (unknown model treated as reference baseline)
func effectiveTokenMultiplier(model string) float64 {
	initMultipliers()

	key := strings.ToLower(strings.TrimSpace(model))
	if key == "" {
		return 1.0
	}

	// Exact match
	if mult, ok := loadedMultipliers[key]; ok {
		return mult
	}

	// Longest prefix match
	best := ""
	bestMult := 1.0
	for name, mult := range loadedMultipliers {
		if strings.HasPrefix(key, name) && len(name) > len(best) {
			best = name
			bestMult = mult
		}
	}

	if best != "" {
		effectiveTokensLog.Printf("Model %q matched via prefix %q (multiplier=%.2f)", model, best, bestMult)
		return bestMult
	}

	effectiveTokensLog.Printf("Unknown model %q, using default multiplier 1.0", model)
	return 1.0
}

// computeBaseWeightedTokens computes the base weighted token count for a single invocation
// by applying per-token-class weights to the raw token counts.
//
// Formula (from the ET specification):
//
//	base = (w_in × I) + (w_cache × C) + (w_out × O) + (w_reason × R) + (w_cache_write × W)
//
// where R (reasoning tokens) is currently not tracked separately and defaults to 0.
func computeBaseWeightedTokens(inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens int) float64 {
	initMultipliers()
	w := loadedTokenWeights
	return w.Input*float64(inputTokens) +
		w.CachedInput*float64(cacheReadTokens) +
		w.Output*float64(outputTokens) +
		w.CacheWrite*float64(cacheWriteTokens)
}

// computeModelEffectiveTokens returns the effective token count for a single model invocation.
//
// Formula (from the ET specification):
//
//	effective_tokens = m × base_weighted_tokens
//
// The result is rounded to the nearest integer.
func computeModelEffectiveTokens(model string, inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens int) int {
	base := computeBaseWeightedTokens(inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens)
	if base == 0 {
		return 0
	}
	mult := effectiveTokenMultiplier(model)
	return int(math.Round(base * mult))
}

// populateEffectiveTokens fills in the EffectiveTokens field on each ModelTokenUsage
// entry and computes the TotalEffectiveTokens aggregate on the summary.
// It is a no-op when summary is nil.
func populateEffectiveTokens(summary *TokenUsageSummary) {
	populateEffectiveTokensWithCustomWeights(summary, nil)
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
		eff := computeModelEffectiveTokensWithWeights(model, usage.InputTokens, usage.OutputTokens,
			usage.CacheReadTokens, usage.CacheWriteTokens, multipliers, classWeights)
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
func resolveEffectiveWeights(custom *types.TokenWeights) (map[string]float64, tokenClassWeights) {
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
func computeModelEffectiveTokensWithWeights(model string, inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens int, multipliers map[string]float64, w tokenClassWeights) int {
	base := w.Input*float64(inputTokens) +
		w.CachedInput*float64(cacheReadTokens) +
		w.Output*float64(outputTokens) +
		w.CacheWrite*float64(cacheWriteTokens)
	if base == 0 {
		return 0
	}

	key := strings.ToLower(strings.TrimSpace(model))
	mult := 1.0
	if key != "" {
		if m, ok := multipliers[key]; ok {
			mult = m
		} else {
			// Longest prefix match
			best := ""
			for name, m := range multipliers {
				if strings.HasPrefix(key, name) && len(name) > len(best) {
					best = name
					mult = m
				}
			}
		}
	}

	return int(math.Round(base * mult))
}
