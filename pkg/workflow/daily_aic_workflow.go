package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/typeutil"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

var dailyAICWorkflowLog = logger.New("workflow:daily_effective_workflow")

const maxDailyAICreditsField = "max-daily-ai-credits"
const maxDailyAICreditsEnvVar = "GH_AW_MAX_DAILY_AI_CREDITS"
const maxDailyAICreditsConfiguredIfExpr = "${{ env.GH_AW_MAX_DAILY_AI_CREDITS != '' }}"

// parseMaxDailyAICValue normalizes max-daily-ai-credits
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
func parseMaxDailyAICValue(raw any) *string {
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

func isMaxDailyAICDisabled(raw any) bool {
	if val, ok := typeutil.ParseIntValue(raw); ok {
		return val == -1
	}
	rawStr, ok := raw.(string)
	if !ok {
		return false
	}
	return strings.TrimSpace(rawStr) == "-1"
}

func resolveMaxDailyAICFromRaw(raw any) (*string, bool) {
	if isMaxDailyAICDisabled(raw) {
		return nil, true
	}
	if value := parseMaxDailyAICValue(raw); value != nil {
		return value, true
	}
	return nil, false
}

func resolveMaxDailyAIC(frontmatter map[string]any, importedJSON string) *string {
	if value, found := resolveMaxDailyAICFromRaw(frontmatter[maxDailyAICreditsField]); found {
		dailyAICWorkflowLog.Print("Resolved max-daily-ai-credits from workflow frontmatter")
		return value
	}
	if importedJSON == "" {
		dailyAICWorkflowLog.Print("No frontmatter value and no imported config; falling back to default max-daily-ai-credits")
		defaultValue := compilerenv.ResolveDefaultMaxDailyAICredits("500000")
		return parseMaxDailyAICValue(defaultValue)
	}
	var imported any
	if err := json.Unmarshal([]byte(importedJSON), &imported); err != nil {
		dailyAICWorkflowLog.Printf("Failed to unmarshal imported max-daily-ai-credits JSON, using default: %v", err)
		defaultValue := compilerenv.ResolveDefaultMaxDailyAICredits("500000")
		return parseMaxDailyAICValue(defaultValue)
	}
	if value, found := resolveMaxDailyAICFromRaw(imported); found {
		dailyAICWorkflowLog.Print("Resolved max-daily-ai-credits from imported config")
		return value
	}
	dailyAICWorkflowLog.Print("Imported config did not provide a usable value; falling back to default max-daily-ai-credits")
	defaultValue := compilerenv.ResolveDefaultMaxDailyAICredits("500000")
	return parseMaxDailyAICValue(defaultValue)
}

// hasMaxDailyAICGuardrail reports whether compiler should emit the
// daily effective-token guardrail wiring. The guardrail is enabled by default.
func hasMaxDailyAICGuardrail(data *WorkflowData) bool {
	return !hasWorkflowExplicitMaxDailyAICDisable(data)
}

func hasWorkflowExplicitMaxDailyAICDisable(data *WorkflowData) bool {
	if data == nil || data.RawFrontmatter == nil {
		return false
	}
	return isMaxDailyAICDisabled(data.RawFrontmatter[maxDailyAICreditsField])
}

// hasMaxDailyAICFrontmatterConfig reports whether the daily ET threshold
// is configured via the max-daily-ai-credits frontmatter/import/default resolution.
// The resolved value is propagated to activation job env so runtime expressions can gate
// setup and guardrail execution consistently.
func hasMaxDailyAICFrontmatterConfig(data *WorkflowData) bool {
	return data != nil && data.MaxDailyAICredits != nil && strings.TrimSpace(*data.MaxDailyAICredits) != ""
}

// validateMaxDailyAICFrontmatter returns an error when the
// max-daily-ai-credits frontmatter field
// is set to an integer below -1. Zero, positive values, and -1 (explicit disable)
// are accepted; GitHub Actions expressions are passed through unchanged for
// runtime evaluation.
func validateMaxDailyAICFrontmatter(data *WorkflowData) error {
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
