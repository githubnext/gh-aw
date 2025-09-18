package workflow

import "fmt"

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
func (c *Compiler) addSafeOutputGitHubToken(steps *[]string, data *WorkflowData) {
	if data.SafeOutputs != nil && data.SafeOutputs.GitHubToken != "" {
		*steps = append(*steps, fmt.Sprintf("          github-token: %s\n", data.SafeOutputs.GitHubToken))
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