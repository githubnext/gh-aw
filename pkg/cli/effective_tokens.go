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
// Token class weights and model costs are derived from embedded models catalog data.
//
// Key responsibilities:
//   - Applying token class weights before model normalization
//   - Computing effective tokens from raw per-model token usage data
//   - Populating effective token counts on TokenUsageSummary after parsing

import "github.com/github/gh-aw/pkg/logger"

// effectiveTokensLog is the debug logger for effective-token accounting decisions.
// Enable with DEBUG=cli:effective_tokens to trace provider cache-read semantics.
var effectiveTokensLog = logger.New("cli:effective_tokens")

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
	// Callers should pass the catalog-normalized provider so canonical aliases like
	// "github", "copilot", and "github_models" collapse to "github-copilot"
	// before this check.
	switch normalizedProvider {
	case "", "anthropic", "openai", "azure-openai", "azure_openai", "github-copilot":
		effectiveTokensLog.Printf("provider %q uses bundled cache-read semantics (cache reads included in input; subtracting once)", normalizedProvider)
		return true
	default:
		effectiveTokensLog.Printf("provider %q uses additive cache-read semantics (cache reads counted separately from input)", normalizedProvider)
		return false
	}
}
