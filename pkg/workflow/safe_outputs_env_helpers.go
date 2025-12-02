package workflow

import (
	"fmt"
	"strings"
	"testing"
)

// assertEnvVarsInSteps checks that all expected environment variables are present in the job steps.
// This is a helper function to reduce duplication in safe outputs env tests.
func assertEnvVarsInSteps(t *testing.T, steps []string, expectedEnvVars []string) {
	t.Helper()
	stepsStr := strings.Join(steps, "")
	for _, expectedEnvVar := range expectedEnvVars {
		if !strings.Contains(stepsStr, expectedEnvVar) {
			t.Errorf("Expected env var %q not found in job YAML", expectedEnvVar)
		}
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
	effectiveToken := c.getEffectiveGitHubTokenForSafeOutput(safeOutputsToken, data)
	*steps = append(*steps, fmt.Sprintf("          github-token: %s\n", effectiveToken))
}

// addSafeOutputGitHubTokenForConfig adds github-token to the with section, preferring per-config token over global
// Uses precedence: config token > safe-outputs global github-token > top-level github-token > default
func (c *Compiler) addSafeOutputGitHubTokenForConfig(steps *[]string, data *WorkflowData, configToken string) {
	var safeOutputsToken string
	if data.SafeOutputs != nil {
		safeOutputsToken = data.SafeOutputs.GitHubToken
	}

	// If app is configured, use app token
	if data.SafeOutputs != nil && data.SafeOutputs.App != nil {
		*steps = append(*steps, "          github-token: ${{ steps.app-token.outputs.token }}\n")
		return
	}

	// Get effective token using double precedence: config > safe-outputs, then > top-level > default
	effectiveToken := getEffectiveGitHubToken(configToken, getEffectiveGitHubToken(safeOutputsToken, data.GitHubToken))
	*steps = append(*steps, fmt.Sprintf("          github-token: %s\n", effectiveToken))
}

// addSafeOutputCopilotGitHubTokenForConfig adds github-token to the with section for Copilot-related operations
// Uses precedence: config token > safe-outputs global github-token > top-level github-token > COPILOT_GITHUB_TOKEN > COPILOT_CLI_TOKEN > GH_AW_COPILOT_TOKEN (legacy) > GH_AW_GITHUB_TOKEN (legacy)
func (c *Compiler) addSafeOutputCopilotGitHubTokenForConfig(steps *[]string, data *WorkflowData, configToken string) {
	var safeOutputsToken string
	if data.SafeOutputs != nil {
		safeOutputsToken = data.SafeOutputs.GitHubToken
	}

	// If app is configured, use app token
	if data.SafeOutputs != nil && data.SafeOutputs.App != nil {
		*steps = append(*steps, "          github-token: ${{ steps.app-token.outputs.token }}\n")
		return
	}

	// Get effective token using double precedence: config > safe-outputs, then > top-level > Copilot default
	effectiveToken := getEffectiveCopilotGitHubToken(configToken, getEffectiveCopilotGitHubToken(safeOutputsToken, data.GitHubToken))
	*steps = append(*steps, fmt.Sprintf("          github-token: %s\n", effectiveToken))
}

// addSafeOutputAgentGitHubTokenForConfig adds github-token to the with section for agent assignment operations
// Uses precedence: config token > GH_AW_AGENT_TOKEN (default)
// This is specifically for assign-to-agent operations which require elevated permissions.
func (c *Compiler) addSafeOutputAgentGitHubTokenForConfig(steps *[]string, data *WorkflowData, configToken string) {
	// If app is configured, use app token
	if data.SafeOutputs != nil && data.SafeOutputs.App != nil {
		*steps = append(*steps, "          github-token: ${{ steps.app-token.outputs.token }}\n")
		return
	}

	// Get effective token: config token > GH_AW_AGENT_TOKEN
	effectiveToken := getEffectiveAgentGitHubToken(configToken)
	*steps = append(*steps, fmt.Sprintf("          github-token: %s\n", effectiveToken))
}

// getEffectiveGitHubTokenForSafeOutput returns the effective token to use for safe outputs
// If app is configured, it uses the app token; otherwise falls back to the configured token or default
func (c *Compiler) getEffectiveGitHubTokenForSafeOutput(customToken string, data *WorkflowData) string {
	// If GitHub App is configured, use the app token
	if data.SafeOutputs != nil && data.SafeOutputs.App != nil {
		tokenLog.Print("Using GitHub App token for safe outputs")
		return "${{ steps.app-token.outputs.token }}"
	}

	// Otherwise use standard token resolution
	return getEffectiveGitHubToken(customToken, data.GitHubToken)
}

// buildTitlePrefixEnvVar builds a title-prefix environment variable line for safe-output jobs.
// envVarName should be the full env var name like "GH_AW_ISSUE_TITLE_PREFIX" or "GH_AW_DISCUSSION_TITLE_PREFIX".
// Returns an empty slice if titlePrefix is empty.
func buildTitlePrefixEnvVar(envVarName string, titlePrefix string) []string {
	if titlePrefix == "" {
		return nil
	}
	return []string{fmt.Sprintf("          %s: %q\n", envVarName, titlePrefix)}
}

// buildLabelsEnvVar builds a labels environment variable line for safe-output jobs.
// envVarName should be the full env var name like "GH_AW_ISSUE_LABELS" or "GH_AW_PR_LABELS".
// Returns an empty slice if labels is empty.
func buildLabelsEnvVar(envVarName string, labels []string) []string {
	if len(labels) == 0 {
		return nil
	}
	labelsStr := strings.Join(labels, ",")
	return []string{fmt.Sprintf("          %s: %q\n", envVarName, labelsStr)}
}

// buildCategoryEnvVar builds a category environment variable line for discussion safe-output jobs.
// envVarName should be the full env var name like "GH_AW_DISCUSSION_CATEGORY".
// Returns an empty slice if category is empty.
func buildCategoryEnvVar(envVarName string, category string) []string {
	if category == "" {
		return nil
	}
	return []string{fmt.Sprintf("          %s: %q\n", envVarName, category)}
}

// buildAllowedReposEnvVar builds an allowed-repos environment variable line for safe-output jobs.
// envVarName should be the full env var name like "GH_AW_ALLOWED_REPOS".
// Returns an empty slice if allowedRepos is empty.
func buildAllowedReposEnvVar(envVarName string, allowedRepos []string) []string {
	if len(allowedRepos) == 0 {
		return nil
	}
	reposStr := strings.Join(allowedRepos, ",")
	return []string{fmt.Sprintf("          %s: %q\n", envVarName, reposStr)}
}
