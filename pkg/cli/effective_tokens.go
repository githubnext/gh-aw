package cli

// This file provides command-line interface functionality for gh-aw.
// This file (effective_tokens.go) implements the Effective Tokens (ET) specification
// defined in docs/src/content/docs/specs/effective-tokens-specification.md.
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
var loadedTokenWeights types.TokenClassWeights

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
