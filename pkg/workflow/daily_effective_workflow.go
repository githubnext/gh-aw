package workflow

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/typeutil"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

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
	if val, ok := typeutil.ParseIntValue(raw); ok && val > 0 {
		s := strconv.Itoa(val)
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
	if normalized, ok := typeutil.NormalizeInt64KMSuffix(rawStr); ok {
		s := normalized
		return &s
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
		return value
	}
	if importedJSON == "" {
		defaultValue := compilerenv.ResolveDefaultMaxDailyEffectiveTokens("")
		return parseMaxDailyEffectiveTokensValue(defaultValue)
	}
	var imported any
	if err := json.Unmarshal([]byte(importedJSON), &imported); err != nil {
		defaultValue := compilerenv.ResolveDefaultMaxDailyEffectiveTokens("")
		return parseMaxDailyEffectiveTokensValue(defaultValue)
	}
	if value, found := resolveMaxDailyEffectiveTokensFromRaw(imported); found {
		return value
	}
	defaultValue := compilerenv.ResolveDefaultMaxDailyEffectiveTokens("")
	return parseMaxDailyEffectiveTokensValue(defaultValue)
}

func hasMaxDailyEffectiveTokensGuardrail(data *WorkflowData) bool {
	if data == nil {
		return false
	}
	if hasMaxDailyEffectiveTokensFrontmatterConfig(data) {
		return true
	}
	return hasMaxDailyEffectiveTokensEnvConfig(data.Env)
}

// hasMaxDailyEffectiveTokensFrontmatterConfig reports whether the daily ET threshold
// is configured via the max-daily-effective-tokens frontmatter field. When true, the
// threshold is emitted into the step env block rather than the workflow-level env, so
// runtime expressions referencing env.GH_AW_MAX_DAILY_EFFECTIVE_TOKENS must not be
// used to gate step execution or setup inputs.
func hasMaxDailyEffectiveTokensFrontmatterConfig(data *WorkflowData) bool {
	return data != nil && data.MaxDailyEffectiveTokens != nil && strings.TrimSpace(*data.MaxDailyEffectiveTokens) != ""
}

func hasMaxDailyEffectiveTokensEnvConfig(envYAML string) bool {
	for _, line := range strings.Split(envYAML, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, maxDailyEffectiveTokensEnvVar+":") ||
			strings.HasPrefix(trimmed, `"`+maxDailyEffectiveTokensEnvVar+`":`) ||
			strings.HasPrefix(trimmed, `'`+maxDailyEffectiveTokensEnvVar+`':`) {
			return true
		}
	}
	return false
}
