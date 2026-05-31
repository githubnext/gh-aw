package workflow

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/typeutil"
)

// parseMaxDailyEffectiveWorkflowValue normalizes max-daily-effective-workflow
// frontmatter values into a runtime-ready string.
//
// Supported inputs:
//   - positive integers
//   - positive numeric strings
//   - GitHub Actions expressions (${{
//     ... }}) preserved verbatim for runtime evaluation
//
// Returns a pointer to the normalized runtime string when valid; nil means the
// field is unset or invalid for runtime use.
func parseMaxDailyEffectiveWorkflowValue(raw any) *string {
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
	if parsed, err := strconv.Atoi(rawStr); err == nil && parsed > 0 {
		s := strconv.Itoa(parsed)
		return &s
	}
	return nil
}

func resolveMaxDailyEffectiveWorkflow(frontmatter map[string]any, importedJSON string) *string {
	if value := parseMaxDailyEffectiveWorkflowValue(frontmatter["max-daily-effective-workflow"]); value != nil {
		return value
	}
	if importedJSON == "" {
		return nil
	}
	var imported any
	if err := json.Unmarshal([]byte(importedJSON), &imported); err != nil {
		return nil
	}
	return parseMaxDailyEffectiveWorkflowValue(imported)
}

func hasMaxDailyEffectiveWorkflowGuardrail(data *WorkflowData) bool {
	return data != nil && data.MaxDailyEffectiveWorkflow != nil && strings.TrimSpace(*data.MaxDailyEffectiveWorkflow) != ""
}
