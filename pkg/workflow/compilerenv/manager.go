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
	// PolicyStrict enables runtime enforcement that workflows must be compiled in strict mode
	// when GH_AW_POLICY_STRICT is set to the string value "true".
	PolicyStrict = "GH_AW_POLICY_STRICT"
	// PolicyModelsAllowed applies experimental models.allowed policy from env.
	// It intersects with workflow models.allowed and does not change
	// models.blocked unless PolicyModelsBlocked is also set.
	PolicyModelsAllowed = "GHAW_POLICY_MODELS_ALLOWED"
	// PolicyModelsBlocked applies experimental models.blocked policy from env.
	// It unions with workflow models.blocked and does not change models.allowed
	// unless PolicyModelsAllowed is also set.
	PolicyModelsBlocked = "GHAW_POLICY_MODELS_BLOCKED"
	// PolicyAllowCreatePullRequest controls whether create-pull-request safe-outputs
	// remain runtime-compliant. Set to the string value "false" to disable the
	// create_pull_request safe-output tool at runtime.
	PolicyAllowCreatePullRequest = "GH_AW_POLICY_ALLOW_CREATE_PULL_REQUEST"
)

// ResolveDefaultMaxTurns returns fallback when the env var is unset/invalid,
// otherwise returns the parsed override as a string.
func ResolveDefaultMaxTurns(fallback string) string {
	if parsed, ok := parsePositiveIntEnvVar(DefaultMaxTurns); ok {
		return strconv.Itoa(parsed)
	}
	return fallback
}

// ResolveDefaultTimeoutMinutes returns fallback when the env var is unset/invalid,
// otherwise returns the parsed override.
func ResolveDefaultTimeoutMinutes(fallback int) int {
	if parsed, ok := parsePositiveIntEnvVar(DefaultTimeoutMinutes); ok {
		return parsed
	}
	return fallback
}

// ResolveDefaultMaxTurnCacheMisses returns fallback when the env var is unset/invalid,
// otherwise returns the parsed override.
func ResolveDefaultMaxTurnCacheMisses(fallback int) int {
	if parsed, ok := parsePositiveIntEnvVar(DefaultMaxTurnCacheMisses); ok {
		return parsed
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

// parsePositiveIntEnvVar parses an environment variable as a base-10 positive int.
// It returns (value, true) when the variable is set to a valid value > 0.
// For unset, empty, non-numeric, or non-positive values, it returns (0, false).
func parsePositiveIntEnvVar(name string) (int, bool) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(raw)
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

// ResolvePolicyModelsAllowed returns configured allowed model policy entries.
// When the env var is unset/empty, ok=false and callers should use frontmatter policy.
func ResolvePolicyModelsAllowed() ([]string, bool) {
	return resolveModelListEnv(PolicyModelsAllowed)
}

// ResolvePolicyModelsBlocked returns configured blocked model policy entries.
// When the env var is unset/empty, ok=false and callers should use frontmatter policy.
func ResolvePolicyModelsBlocked() ([]string, bool) {
	return resolveModelListEnv(PolicyModelsBlocked)
}

func resolveModelListEnv(name string) ([]string, bool) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return nil, false
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	})
	if len(parts) == 0 {
		return nil, false
	}
	result := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		model := strings.TrimSpace(part)
		if model == "" {
			continue
		}
		if strings.ContainsAny(model, " \t") {
			managerLog.Printf("Skipping invalid model policy entry in %s: %q (use comma/newline separators)", name, model)
			continue
		}
		if _, exists := seen[model]; exists {
			continue
		}
		seen[model] = struct{}{}
		result = append(result, model)
	}
	if len(result) == 0 {
		return nil, false
	}
	managerLog.Printf("Applying model policy override %s with %d model(s)", name, len(result))
	return result, true
}
