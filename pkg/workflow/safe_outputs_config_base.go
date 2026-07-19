package workflow

import (
	"github.com/github/gh-aw/pkg/typeutil"
)

// parseBaseSafeOutputConfig parses common fields (max, github-token, github-app, staged) from a config map.
// If defaultMax is provided (> 0), it will be set as the default value for config.Max
// before parsing the max field from configMap. Supports both integer values and GitHub
// Actions expression strings (e.g. "${{ inputs.max }}").
func (c *Compiler) parseBaseSafeOutputConfig(configMap map[string]any, config *BaseSafeOutputConfig, defaultMax int) {
	parseBaseSafeOutputMax(configMap, config, defaultMax)
	parseBaseSafeOutputAuth(configMap, config)
	parseBaseSafeOutputStaged(configMap, config)
	parseBaseSafeOutputSamples(configMap, config)
}

func parseBaseSafeOutputMax(configMap map[string]any, config *BaseSafeOutputConfig, defaultMax int) {
	if defaultMax > 0 {
		safeOutputsConfigLog.Printf("Setting default max: %d", defaultMax)
		config.Max = defaultIntStr(defaultMax)
	}

	if max, exists := configMap["max"]; exists {
		switch v := max.(type) {
		case string:
			if isExpression(v) {
				safeOutputsConfigLog.Printf("Parsed max as GitHub Actions expression: %s", v)
				config.Max = &v
			}
		default:
			if maxInt, ok := typeutil.ParseIntValue(max); ok {
				safeOutputsConfigLog.Printf("Parsed max as integer: %d", maxInt)
				s := defaultIntStr(maxInt)
				config.Max = s
			}
		}
	}
}

func parseBaseSafeOutputAuth(configMap map[string]any, config *BaseSafeOutputConfig) {
	if githubToken, exists := configMap["github-token"]; exists {
		if githubTokenStr, ok := githubToken.(string); ok {
			safeOutputsConfigLog.Print("Parsed custom github-token from config")
			config.GitHubToken = githubTokenStr
		}
	}

	// Parse github-app (per-handler GitHub App credentials for token minting)
	if app, exists := configMap["github-app"]; exists {
		if appMap, ok := app.(map[string]any); ok {
			safeOutputsConfigLog.Print("Parsed custom github-app from config")
			config.GitHubApp = parseAppConfig(appMap)
		}
	}
}

func parseBaseSafeOutputStaged(configMap map[string]any, config *BaseSafeOutputConfig) {
	if err := preprocessBoolFieldAsString(configMap, "staged", safeOutputsConfigLog); err != nil {
		safeOutputsConfigLog.Printf("Invalid staged value: %v", err)
	} else if staged, exists := configMap["staged"]; exists {
		if stagedStr, ok := staged.(string); ok && stagedStr != "" {
			safeOutputsConfigLog.Printf("Parsed staged flag: %s", stagedStr)
			value := TemplatableBool(stagedStr)
			config.Staged = &value
		}
	}
}

func parseBaseSafeOutputSamples(configMap map[string]any, config *BaseSafeOutputConfig) {
	if samples, exists := configMap["samples"]; exists {
		parsed := parseSamplesValue(samples)
		if len(parsed) > 0 {
			safeOutputsConfigLog.Printf("Parsed %d samples entries", len(parsed))
			config.Samples = parsed
		}
	}
}

// parseSamplesValue normalizes a `samples` frontmatter value into a list of
// objects. Accepted shapes:
//   - YAML list of mappings: returned as-is
//   - single YAML mapping: wrapped into a one-element list
//
// Any other shape returns an empty slice — schema validation rejects those
// shapes upstream and we keep this parser strict to match.
func parseSamplesValue(samples any) []map[string]any {
	switch v := samples.(type) {
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			} else if mStr, ok := item.(map[string]string); ok {
				converted := make(map[string]any, len(mStr))
				for k, s := range mStr {
					converted[k] = s
				}
				out = append(out, converted)
			}
		}
		return out
	case map[string]any:
		return []map[string]any{v}
	default:
		return nil
	}
}
