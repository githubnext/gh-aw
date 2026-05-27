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
	// DefaultMaxTurns is the enterprise override for engine.max-turns when it is not
	// explicitly configured in workflow frontmatter.
	DefaultMaxTurns = "GH_AW_DEFAULT_MAX_TURNS"
	// DefaultTimeoutMinutes is the enterprise override for top-level timeout-minutes
	// when it is not explicitly configured in workflow frontmatter.
	DefaultTimeoutMinutes = "GH_AW_DEFAULT_TIMEOUT_MINUTES"
	// DefaultDetectionModel is the enterprise override for selecting the detection
	// job model when threat-detection.engine.model is not set.
	DefaultDetectionModel = "GH_AW_DEFAULT_DETECTION_MODEL"

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
			Name:        DefaultMaxTurns,
			Description: "Default engine.max-turns used when workflow frontmatter does not set one",
		},
		{
			Name:        DefaultTimeoutMinutes,
			Description: "Default timeout-minutes used when workflow frontmatter does not set one",
		},
		{
			Name:        DefaultDetectionModel,
			Description: "Default detection job model used when threat-detection.engine.model is not set",
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

// ResolveDefaultMaxTurns returns fallback when the env var is unset/invalid,
// otherwise returns the parsed override as a string.
func ResolveDefaultMaxTurns(fallback string) string {
	if parsed, ok := parsePositiveIntEnvVar(DefaultMaxTurns); ok {
		return strconv.FormatInt(parsed, 10)
	}
	return fallback
}

// ResolveDefaultTimeoutMinutes returns fallback when the env var is unset/invalid,
// otherwise returns the parsed override.
func ResolveDefaultTimeoutMinutes(fallback int) int {
	if parsed, ok := parsePositiveIntEnvVar(DefaultTimeoutMinutes); ok {
		return int(parsed)
	}
	return fallback
}

// ResolveDefaultDetectionModel returns fallback when the env var is unset,
// otherwise returns the trimmed override value.
func ResolveDefaultDetectionModel(fallback string) string {
	raw := strings.TrimSpace(os.Getenv(DefaultDetectionModel))
	if raw == "" {
		return fallback
	}
	return raw
}

// parsePositiveIntEnvVar parses an environment variable as a base-10 positive int64.
// It returns (value, true) when the variable is set to a valid value > 0.
// For unset, empty, non-numeric, or non-positive values, it returns (0, false).
func parsePositiveIntEnvVar(name string) (int64, bool) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return 0, false
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
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
