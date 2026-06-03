package workflow

import (
	"encoding/json"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/typeutil"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

var dailyEffectiveWorkflowLog = logger.New("workflow:daily_effective_workflow")

const maxDailyEffectiveTokensField = "max-daily-effective-tokens"
const maxDailyEffectiveTokensEnvVar = "GH_AW_MAX_DAILY_EFFECTIVE_TOKENS"
const maxDailyEffectiveTokensConfiguredIfExpr = "${{ env.GH_AW_MAX_DAILY_EFFECTIVE_TOKENS != '' }}"

// parseMaxDailyEffectiveTokensValue normalizes max-daily-effective-tokens
// frontmatter values into a runtime-ready string.
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
	if value, found := resolveMaxDailyEffectiveTokensFromRaw(frontmatter[maxDailyEffectiveTokensField]); found {
		dailyEffectiveWorkflowLog.Print("Resolved max-daily-effective-tokens from workflow frontmatter")
		return value
	}
	if importedJSON == "" {
		dailyEffectiveWorkflowLog.Print("No frontmatter value and no imported config; falling back to default max-daily-effective-tokens")
		defaultValue := compilerenv.ResolveDefaultMaxDailyEffectiveTokens("")
		return parseMaxDailyEffectiveTokensValue(defaultValue)
	}
	var imported any
	if err := json.Unmarshal([]byte(importedJSON), &imported); err != nil {
		dailyEffectiveWorkflowLog.Printf("Failed to unmarshal imported max-daily-effective-tokens JSON, using default: %v", err)
		defaultValue := compilerenv.ResolveDefaultMaxDailyEffectiveTokens("")
		return parseMaxDailyEffectiveTokensValue(defaultValue)
	}
	if value, found := resolveMaxDailyEffectiveTokensFromRaw(imported); found {
		dailyEffectiveWorkflowLog.Print("Resolved max-daily-effective-tokens from imported config")
		return value
	}
	dailyEffectiveWorkflowLog.Print("Imported config did not provide a usable value; falling back to default max-daily-effective-tokens")
	defaultValue := compilerenv.ResolveDefaultMaxDailyEffectiveTokens("")
	return parseMaxDailyEffectiveTokensValue(defaultValue)
}

// hasMaxDailyEffectiveTokensGuardrail reports whether compiler should emit the
// daily effective-token guardrail wiring. The guardrail is enabled by default
// and can only be suppressed by an explicit workflow-level negative value (-1).
func hasMaxDailyEffectiveTokensGuardrail(data *WorkflowData) bool {
	return !hasWorkflowExplicitMaxDailyEffectiveTokensDisable(data)
}

func hasWorkflowExplicitMaxDailyEffectiveTokensDisable(data *WorkflowData) bool {
	if data == nil || data.RawFrontmatter == nil {
		return false
	}
	return isMaxDailyEffectiveTokensDisabled(data.RawFrontmatter[maxDailyEffectiveTokensField])
}

// hasMaxDailyEffectiveTokensFrontmatterConfig reports whether the daily ET threshold
// is configured via the max-daily-effective-tokens frontmatter/import/default resolution.
// The resolved value is propagated to activation job env so runtime expressions can gate
// setup and guardrail execution consistently.
func hasMaxDailyEffectiveTokensFrontmatterConfig(data *WorkflowData) bool {
	return data != nil && data.MaxDailyEffectiveTokens != nil && strings.TrimSpace(*data.MaxDailyEffectiveTokens) != ""
}
