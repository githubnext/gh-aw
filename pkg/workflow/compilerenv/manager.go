package compilerenv

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	// DefaultMaxEffectiveTokens is the enterprise override for AWF apiProxy.maxEffectiveTokens
	// when max-effective-tokens is not explicitly configured in workflow frontmatter.
	DefaultMaxEffectiveTokens = "GH_AW_DEFAULT_MAX_EFFECTIVE_TOKENS"

	// DefaultModelCopilot is the enterprise override for Copilot fallback model selection.
	DefaultModelCopilot = "GH_AW_DEFAULT_MODEL_COPILOT"
	// DefaultModelClaude is the enterprise override for Claude fallback model selection.
	DefaultModelClaude = "GH_AW_DEFAULT_MODEL_CLAUDE"
	// DefaultModelCodex is the enterprise override for Codex fallback model selection.
	DefaultModelCodex = "GH_AW_DEFAULT_MODEL_CODEX"
)

type Variable struct {
	Name        string
	Description string
}

// EnterpriseVariables returns all compiler-managed enterprise control variables.
func EnterpriseVariables() []Variable {
	return []Variable{
		{
			Name:        DefaultMaxEffectiveTokens,
			Description: "Default max-effective-tokens used when workflow frontmatter does not set one",
		},
		{
			Name:        DefaultModelCopilot,
			Description: "Default Copilot model fallback override when GH_AW_MODEL_AGENT/DETECTION_COPILOT is unset",
		},
		{
			Name:        DefaultModelClaude,
			Description: "Default Claude model fallback override when GH_AW_MODEL_AGENT/DETECTION_CLAUDE is unset",
		},
		{
			Name:        DefaultModelCodex,
			Description: "Default Codex model fallback override when GH_AW_MODEL_AGENT/DETECTION_CODEX is unset",
		},
	}
}

// ResolveDefaultMaxEffectiveTokens returns fallback when the env var is unset/invalid,
// otherwise returns the parsed override.
func ResolveDefaultMaxEffectiveTokens(fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(DefaultMaxEffectiveTokens))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

// BuildModelOverrideExpression builds a vars expression with primary model var, enterprise
// default model var, and built-in fallback model.
func BuildModelOverrideExpression(primaryVar, enterpriseDefaultVar, builtinFallback string) string {
	escaped := strings.ReplaceAll(builtinFallback, "'", "''")
	return fmt.Sprintf("${{ vars.%s || vars.%s || '%s' }}", primaryVar, enterpriseDefaultVar, escaped)
}

// BuildModelOverrideExpressionEmptyFallback builds a vars expression with primary model var,
// enterprise default model var, and empty string fallback.
func BuildModelOverrideExpressionEmptyFallback(primaryVar, enterpriseDefaultVar string) string {
	return fmt.Sprintf("${{ vars.%s || vars.%s || '' }}", primaryVar, enterpriseDefaultVar)
}

