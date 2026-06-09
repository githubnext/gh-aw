package cli

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
