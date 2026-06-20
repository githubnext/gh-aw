package compilerenv

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var managerLog = logger.New("compilerenv:manager")

const (
	// DefaultMaxDailyAICredits is the enterprise override for the top-level
	// max-daily-ai-credits guardrail when it is not explicitly configured in
	// workflow frontmatter.
	DefaultMaxDailyAICredits = "GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS"
	// DefaultMaxAICredits is the enterprise override for AWF apiProxy.maxAiCredits
	// when max-ai-credits is not explicitly configured in workflow frontmatter.
	DefaultMaxAICredits = "GH_AW_DEFAULT_MAX_AI_CREDITS"
	// DefaultMaxTurnCacheMisses is the enterprise override for AWF
	// apiProxy.maxCacheMisses when max-turn-cache-misses is not explicitly configured
	// in workflow frontmatter.
	DefaultMaxTurnCacheMisses = "GH_AW_DEFAULT_MAX_TURN_CACHE_MISSES"
	// DefaultDetectionMaxAICredits is the enterprise override for the
	// threat-detection AWF apiProxy.maxAiCredits budget when
	// safe-outputs.threat-detection.max-ai-credits is not explicitly configured.
	DefaultDetectionMaxAICredits = "GH_AW_DEFAULT_DETECTION_MAX_AI_CREDITS"
	// DefaultMaxTurns is the enterprise override for max-turns when it is not
	// explicitly configured in workflow frontmatter.
	DefaultMaxTurns = "GH_AW_DEFAULT_MAX_TURNS"
	// DefaultTimeoutMinutes is the enterprise override for top-level timeout-minutes
	// when it is not explicitly configured in workflow frontmatter.
	DefaultTimeoutMinutes = "GH_AW_DEFAULT_TIMEOUT_MINUTES"
	// DefaultDetectionModel is the enterprise override for selecting the detection
	// job model when threat-detection.engine.model is not set.
	DefaultDetectionModel = "GH_AW_DEFAULT_DETECTION_MODEL"

	// DefaultUTC is the enterprise override for the project home timezone used
	// when rendering local times in CLI output.
	DefaultUTC = "GH_AW_DEFAULT_UTC"

	// DefaultModelCopilot is the enterprise override for Copilot fallback model selection.
	DefaultModelCopilot = "GH_AW_DEFAULT_MODEL_COPILOT"
	// DefaultModelClaude is the enterprise override for Claude fallback model selection.
	DefaultModelClaude = "GH_AW_DEFAULT_MODEL_CLAUDE"
	// DefaultModelCodex is the enterprise override for Codex fallback model selection.
	DefaultModelCodex = "GH_AW_DEFAULT_MODEL_CODEX"
)

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

// ResolveDefaultMaxTurnCacheMisses returns fallback when the env var is unset/invalid,
// otherwise returns the parsed override.
func ResolveDefaultMaxTurnCacheMisses(fallback int) int {
	if parsed, ok := parsePositiveIntEnvVar(DefaultMaxTurnCacheMisses); ok {
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
	managerLog.Printf("Applying enterprise detection model override %s=%q (fallback was %q)", DefaultDetectionModel, raw, fallback)
	return raw
}

// ResolveDefaultUTC returns fallback when the env var is unset, otherwise
// returns the trimmed override value.
func ResolveDefaultUTC(fallback string) string {
	raw := strings.TrimSpace(os.Getenv(DefaultUTC))
	if raw == "" {
		return fallback
	}
	managerLog.Printf("Applying enterprise timezone override %s=%q (fallback was %q)", DefaultUTC, raw, fallback)
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

// BuildDefaultMaxDailyAICreditsExpression builds a vars expression that resolves
// max-daily-ai-credits at runtime from the GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS
// GitHub variable, falling back to builtinDefault when the variable is unset.
func BuildDefaultMaxDailyAICreditsExpression(builtinDefault string) string {
	escaped := strings.ReplaceAll(builtinDefault, "'", "''")
	return fmt.Sprintf("${{ vars.%s || '%s' }}", DefaultMaxDailyAICredits, escaped)
}

// BuildDefaultMaxAICreditsExpression builds a vars expression that resolves
// max-ai-credits at runtime from the GH_AW_DEFAULT_MAX_AI_CREDITS GitHub variable,
// falling back to builtinDefault when the variable is unset. The expression is
// embedded in the compiled workflow and evaluated by the GitHub Actions runner.
func BuildDefaultMaxAICreditsExpression(builtinDefault string) string {
	escaped := strings.ReplaceAll(builtinDefault, "'", "''")
	return fmt.Sprintf("${{ vars.%s || '%s' }}", DefaultMaxAICredits, escaped)
}

// BuildDefaultDetectionMaxAICreditsExpression builds a vars expression that resolves
// the threat-detection max-ai-credits default at runtime from the
// GH_AW_DEFAULT_DETECTION_MAX_AI_CREDITS GitHub variable, falling back to
// builtinDefault when the variable is unset.
func BuildDefaultDetectionMaxAICreditsExpression(builtinDefault string) string {
	escaped := strings.ReplaceAll(builtinDefault, "'", "''")
	return fmt.Sprintf("${{ vars.%s || '%s' }}", DefaultDetectionMaxAICredits, escaped)
}

// BuildDefaultMaxTurnsExpression builds a vars expression that resolves max-turns
// at runtime from the GH_AW_DEFAULT_MAX_TURNS GitHub variable. An empty string is
// returned as the fallback so that an unset variable is treated as "no limit".
func BuildDefaultMaxTurnsExpression() string {
	return fmt.Sprintf("${{ vars.%s || '' }}", DefaultMaxTurns)
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
