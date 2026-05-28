package types

// TokenClassWeights holds per-token-class weights for effective token computation.
// Each field corresponds to one token class; a zero value means "use default".
// The JSON keys use underscores to match pkg/cli/data/model_multipliers.json format.
type TokenClassWeights struct {
	Input       float64 `json:"input,omitempty"`
	CachedInput float64 `json:"cached_input,omitempty"`
	Output      float64 `json:"output,omitempty"`
	Reasoning   float64 `json:"reasoning,omitempty"`
	CacheWrite  float64 `json:"cache_write,omitempty"`
}

// TokenWeights defines custom model cost information for effective token computation.
// It mirrors the structure of model_multipliers.json and allows per-workflow overrides.
// Specified under engine.token-weights in the workflow frontmatter and stored in
// aw_info.json at runtime.
type TokenWeights struct {
	// Multipliers maps model names to cost multipliers relative to the reference model.
	// Keys are matched case-insensitively with prefix matching as a fallback.
	Multipliers map[string]float64 `json:"multipliers,omitempty"`
	// TokenClassWeights overrides the per-token-class weights used before the model multiplier.
	// A nil pointer means no overrides; individual zero fields mean "use default".
	TokenClassWeights *TokenClassWeights `json:"token-class-weights,omitempty"`
}
