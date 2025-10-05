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

// SafeOutputEnvConfig contains configuration for safe-output environment variables and "with" parameters
type SafeOutputEnvConfig struct {
	TargetValue   string // The target value (e.g., "*" or specific issue number)
	TargetEnvName string // The environment variable name for target (e.g., "GITHUB_AW_COMMENT_TARGET")
	GitHubToken   string // The github-token value for this specific configuration
}

// getCustomSafeOutputEnvVars adds all safe-output environment variables to the provided env map
// and optionally populates github-token in the withParams map
// This includes:
// - Standard vars: GITHUB_AW_AGENT_OUTPUT, GITHUB_AW_WORKFLOW_NAME
// - Custom vars from safe-outputs.env
// - Optional target env var (if config.TargetValue and config.TargetEnvName are provided)
// - Staged flag (if trial mode is enabled or SafeOutputs.Staged is true)
// - Optional github-token in withParams (if withParams is not nil and github-token is configured)
func (c *Compiler) getCustomSafeOutputEnvVars(env map[string]string, data *WorkflowData, mainJobName string, config *SafeOutputEnvConfig, withParams map[string]string) {
	// Add standard safe-output environment variables
	env["GITHUB_AW_AGENT_OUTPUT"] = fmt.Sprintf("${{ needs.%s.outputs.output }}", mainJobName)
	env["GITHUB_AW_WORKFLOW_NAME"] = fmt.Sprintf("%q", data.Name)

	// Add custom environment variables from safe-outputs.env
	if data.SafeOutputs != nil && len(data.SafeOutputs.Env) > 0 {
		for key, value := range data.SafeOutputs.Env {
			env[key] = value
		}
	}

	// Add optional configuration
	if config != nil {
		// Add target environment variable if configured
		if config.TargetValue != "" && config.TargetEnvName != "" {
			env[config.TargetEnvName] = fmt.Sprintf("%q", config.TargetValue)
		}
	}

	// Add staged flag if needed (always check, not conditional on config)
	if c.trialMode || (data.SafeOutputs != nil && data.SafeOutputs.Staged) {
		env["GITHUB_AW_SAFE_OUTPUTS_STAGED"] = "\"true\""
	}

	// Handle github-token in withParams if provided
	if withParams != nil {
		var token string
		if config != nil && config.GitHubToken != "" {
			token = config.GitHubToken
		} else if data.SafeOutputs != nil && data.SafeOutputs.GitHubToken != "" {
			token = data.SafeOutputs.GitHubToken
		}

		if token != "" {
			withParams["github-token"] = token
		}
	}
}

// addSafeOutputGitHubToken adds github-token to the with section of github-script actions
func (c *Compiler) addSafeOutputGitHubToken(steps *[]string, data *WorkflowData) {
	if data.SafeOutputs != nil && data.SafeOutputs.GitHubToken != "" {
		*steps = append(*steps, fmt.Sprintf("          github-token: %s\n", data.SafeOutputs.GitHubToken))
	} else {
		*steps = append(*steps, "          github-token: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}\n")
	}
}

// addSafeOutputGitHubTokenForConfig adds github-token to the with section, preferring per-config token over global
func (c *Compiler) addSafeOutputGitHubTokenForConfig(steps *[]string, data *WorkflowData, configToken string) {
	token := configToken
	if token == "" && data.SafeOutputs != nil {
		token = data.SafeOutputs.GitHubToken
	}
	if token != "" {
		*steps = append(*steps, fmt.Sprintf("          github-token: %s\n", token))
	}
}

// getSafeOutputGitHubTokenForConfig returns the github-token value, preferring per-config token over global
// This is the non-callback version of addSafeOutputGitHubTokenForConfig
func (c *Compiler) getSafeOutputGitHubTokenForConfig(data *WorkflowData, configToken string) string {
	token := configToken
	if token == "" && data.SafeOutputs != nil {
		token = data.SafeOutputs.GitHubToken
	}
	return token
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
