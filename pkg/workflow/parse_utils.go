package workflow

import (
	"fmt"
	"strings"
)

// extractStringValue extracts a string value from the frontmatter map
func extractStringValue(frontmatter map[string]any, key string) string {
	value, exists := frontmatter[key]
	if !exists {
		return ""
	}

	if strValue, ok := value.(string); ok {
		return strValue
	}

	return ""
}

// parseIntValue safely parses various numeric types to int
// This is a common utility used across multiple parsing functions
func parseIntValue(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case uint64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// addCustomSafeOutputEnvVars adds custom environment variables to safe output job steps
func (c *Compiler) addCustomSafeOutputEnvVars(steps *[]string, data *WorkflowData) {
	if data.SafeOutputs != nil && len(data.SafeOutputs.Env) > 0 {
		for key, value := range data.SafeOutputs.Env {
			*steps = append(*steps, fmt.Sprintf("          %s: %s\n", key, value))
		}
	}
}

// addSafeOutputGitHubToken adds github-token to the with section of github-script actions
// Uses precedence: safe-outputs global github-token > top-level github-token > default
func (c *Compiler) addSafeOutputGitHubToken(steps *[]string, data *WorkflowData) {
	var safeOutputsToken string
	if data.SafeOutputs != nil {
		safeOutputsToken = data.SafeOutputs.GitHubToken
	}
	effectiveToken := getEffectiveGitHubToken(safeOutputsToken, data.GitHubToken)
	*steps = append(*steps, fmt.Sprintf("          github-token: %s\n", effectiveToken))
}

// addSafeOutputGitHubTokenForConfig adds github-token to the with section, preferring per-config token over global
// Uses precedence: config token > safe-outputs global github-token > top-level github-token > default
func (c *Compiler) addSafeOutputGitHubTokenForConfig(steps *[]string, data *WorkflowData, configToken string) {
	var safeOutputsToken string
	if data.SafeOutputs != nil {
		safeOutputsToken = data.SafeOutputs.GitHubToken
	}
	// Get effective token using double precedence: config > safe-outputs, then > top-level > default
	effectiveToken := getEffectiveGitHubToken(configToken, getEffectiveGitHubToken(safeOutputsToken, data.GitHubToken))
	*steps = append(*steps, fmt.Sprintf("          github-token: %s\n", effectiveToken))
}

// addSafeOutputCopilotGitHubTokenForConfig adds github-token to the with section for Copilot-related operations
// Uses precedence: config token > safe-outputs global github-token > top-level github-token > GH_AW_COPILOT_TOKEN > GH_AW_GITHUB_TOKEN > GITHUB_TOKEN
func (c *Compiler) addSafeOutputCopilotGitHubTokenForConfig(steps *[]string, data *WorkflowData, configToken string) {
	var safeOutputsToken string
	if data.SafeOutputs != nil {
		safeOutputsToken = data.SafeOutputs.GitHubToken
	}
	// Get effective token using double precedence: config > safe-outputs, then > top-level > Copilot default
	effectiveToken := getEffectiveCopilotGitHubToken(configToken, getEffectiveCopilotGitHubToken(safeOutputsToken, data.GitHubToken))
	*steps = append(*steps, fmt.Sprintf("          github-token: %s\n", effectiveToken))
}

// filterMapKeys creates a new map excluding the specified keys
func filterMapKeys(original map[string]any, excludeKeys ...string) map[string]any {
	excludeSet := make(map[string]bool)
	for _, key := range excludeKeys {
		excludeSet[key] = true
	}

	result := make(map[string]any)
	for key, value := range original {
		if !excludeSet[key] {
			result[key] = value
		}
	}
	return result
}

// extractYAMLValue extracts a scalar value from the frontmatter map
func (c *Compiler) extractYAMLValue(frontmatter map[string]any, key string) string {
	if value, exists := frontmatter[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
		if num, ok := value.(int); ok {
			return fmt.Sprintf("%d", num)
		}
		if num, ok := value.(int64); ok {
			return fmt.Sprintf("%d", num)
		}
		if num, ok := value.(uint64); ok {
			return fmt.Sprintf("%d", num)
		}
		if float, ok := value.(float64); ok {
			return fmt.Sprintf("%.0f", float)
		}
	}
	return ""
}

// indentYAMLLines adds indentation to all lines of a multi-line YAML string except the first
func (c *Compiler) indentYAMLLines(yamlContent, indent string) string {
	if yamlContent == "" {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	if len(lines) <= 1 {
		return yamlContent
	}

	// First line doesn't get additional indentation
	result := lines[0]
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			result += "\n" + indent + lines[i]
		} else {
			result += "\n" + lines[i]
		}
	}

	return result
}
