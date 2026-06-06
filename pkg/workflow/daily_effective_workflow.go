package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/typeutil"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

var dailyEffectiveWorkflowLog = logger.New("workflow:daily_effective_workflow")

const maxDailyAICreditsField = "max-daily-ai-credits"
const maxDailyAICreditsEnvVar = "GH_AW_MAX_DAILY_AI_CREDITS"
const maxDailyAICreditsConfiguredIfExpr = "${{ env.GH_AW_MAX_DAILY_AI_CREDITS != '' }}"

// parseMaxDailyEffectiveTokensValue normalizes max-daily-ai-credits
// values into a runtime-ready string.
//
// Supported inputs:
//   - positive integers
//   - positive numeric strings
//   - GitHub Actions expressions (${{
//     ... }}) preserved verbatim for runtime evaluation
//
// Returns a pointer to the normalized runtime string when valid; nil means the
// field is unset, explicitly disabled, or invalid for runtime use.
func parseMaxDailyEffectiveTokensValue(raw any) *string {
	if normalized, ok := normalizePositiveEffectiveTokenLimit(raw); ok {
		s := normalized
		return &s
	}

	rawStr, ok := raw.(string)
	if !ok {
		return nil
	}

	rawStr = strings.TrimSpace(rawStr)
	if rawStr == "" {
		return nil
	}
	if isExpression(rawStr) {
		return &rawStr
	}
	return nil
}

func isMaxDailyEffectiveTokensDisabled(raw any) bool {
	if val, ok := typeutil.ParseIntValue(raw); ok {
		return val == -1
	}
	rawStr, ok := raw.(string)
	if !ok {
		return false
	}
	return strings.TrimSpace(rawStr) == "-1"
}

func resolveMaxDailyEffectiveTokensFromRaw(raw any) (*string, bool) {
	if isMaxDailyEffectiveTokensDisabled(raw) {
		return nil, true
	}
	if value := parseMaxDailyEffectiveTokensValue(raw); value != nil {
		return value, true
	}
	return nil, false
}

func resolveMaxDailyEffectiveTokens(frontmatter map[string]any, importedJSON string) *string {
	if value, found := resolveMaxDailyEffectiveTokensFromRaw(frontmatter[maxDailyAICreditsField]); found {
		dailyEffectiveWorkflowLog.Print("Resolved max-daily-ai-credits from workflow frontmatter")
		return value
	}
	if importedJSON == "" {
		dailyEffectiveWorkflowLog.Print("No frontmatter value and no imported config; falling back to default max-daily-ai-credits")
		defaultValue := compilerenv.ResolveDefaultMaxDailyAICredits("500000")
		return parseMaxDailyEffectiveTokensValue(defaultValue)
	}
	var imported any
	if err := json.Unmarshal([]byte(importedJSON), &imported); err != nil {
		dailyEffectiveWorkflowLog.Printf("Failed to unmarshal imported max-daily-ai-credits JSON, using default: %v", err)
		defaultValue := compilerenv.ResolveDefaultMaxDailyAICredits("500000")
		return parseMaxDailyEffectiveTokensValue(defaultValue)
	}
	if value, found := resolveMaxDailyEffectiveTokensFromRaw(imported); found {
		dailyEffectiveWorkflowLog.Print("Resolved max-daily-ai-credits from imported config")
		return value
	}
	dailyEffectiveWorkflowLog.Print("Imported config did not provide a usable value; falling back to default max-daily-ai-credits")
	defaultValue := compilerenv.ResolveDefaultMaxDailyAICredits("500000")
	return parseMaxDailyEffectiveTokensValue(defaultValue)
}

// hasMaxDailyEffectiveTokensGuardrail reports whether compiler should emit the
// daily effective-token guardrail wiring. The guardrail is enabled by default.
func hasMaxDailyEffectiveTokensGuardrail(data *WorkflowData) bool {
	return !hasWorkflowExplicitMaxDailyEffectiveTokensDisable(data)
}

func hasWorkflowExplicitMaxDailyEffectiveTokensDisable(data *WorkflowData) bool {
	if data == nil || data.RawFrontmatter == nil {
		return false
	}
	return isMaxDailyEffectiveTokensDisabled(data.RawFrontmatter[maxDailyAICreditsField])
}

// hasMaxDailyEffectiveTokensFrontmatterConfig reports whether the daily ET threshold
// is configured via the max-daily-ai-credits frontmatter/import/default resolution.
// The resolved value is propagated to activation job env so runtime expressions can gate
// setup and guardrail execution consistently.
func hasMaxDailyEffectiveTokensFrontmatterConfig(data *WorkflowData) bool {
	return data != nil && data.MaxDailyEffectiveTokens != nil && strings.TrimSpace(*data.MaxDailyEffectiveTokens) != ""
}

// validateMaxDailyEffectiveTokensFrontmatter returns an error when the
// max-daily-ai-credits frontmatter field
// is set to an integer below -1. Zero, positive values, and -1 (explicit disable)
// are accepted; GitHub Actions expressions are passed through unchanged for
// runtime evaluation.
func validateMaxDailyEffectiveTokensFrontmatter(data *WorkflowData) error {
	if data == nil || data.RawFrontmatter == nil {
		return nil
	}
	raw, ok := data.RawFrontmatter[maxDailyAICreditsField]
	if !ok {
		return nil
	}
	if val, ok := typeutil.ParseIntValue(raw); ok && val < -1 {
		return fmt.Errorf("%s must be -1 (disable) or a positive integer, got %d", maxDailyAICreditsField, val)
	}
	return nil
}
