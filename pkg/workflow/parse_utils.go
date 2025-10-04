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

// getCustomSafeOutputEnvVars adds custom environment variables from safe-outputs.env to the provided env map
// This is the non-callback version of addCustomSafeOutputEnvVars
// It also adds the standard GITHUB_AW_AGENT_OUTPUT and GITHUB_AW_WORKFLOW_NAME variables
func (c *Compiler) getCustomSafeOutputEnvVars(env map[string]string, data *WorkflowData, mainJobName string) {
	// Add standard safe-output environment variables
	env["GITHUB_AW_AGENT_OUTPUT"] = fmt.Sprintf("${{ needs.%s.outputs.output }}", mainJobName)
	env["GITHUB_AW_WORKFLOW_NAME"] = fmt.Sprintf("%q", data.Name)

	// Add custom environment variables from safe-outputs.env
	if data.SafeOutputs != nil && len(data.SafeOutputs.Env) > 0 {
		for key, value := range data.SafeOutputs.Env {
			env[key] = value
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

// addTargetEnvIfConfigured adds the target environment variable if configured
func (c *Compiler) addTargetEnvIfConfigured(env map[string]string, target string, envVarName string) {
	if target != "" {
		env[envVarName] = fmt.Sprintf("%q", target)
	}
}

// addStagedEnvIfNeeded adds the staged flag environment variable if needed
func (c *Compiler) addStagedEnvIfNeeded(env map[string]string, data *WorkflowData) {
	if c.trialMode || data.SafeOutputs.Staged {
		env["GITHUB_AW_SAFE_OUTPUTS_STAGED"] = "\"true\""
	}
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
