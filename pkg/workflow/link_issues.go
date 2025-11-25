package workflow

import (
	"fmt"
)

// LinkIssuesConfig holds configuration for linking issues from agent output
type LinkIssuesConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Target               string `yaml:"target,omitempty"`      // Target: "triggering" (default) or "*" (any issues)
	TargetRepoSlug       string `yaml:"target-repo,omitempty"` // Target repository in format "owner/repo" for cross-repository operations
}

// parseLinkIssuesConfig handles link-issues configuration
func (c *Compiler) parseLinkIssuesConfig(outputMap map[string]any) *LinkIssuesConfig {
	if configData, exists := outputMap["link-issues"]; exists {
		// Handle the case where configData is false (explicitly disabled)
		if configBool, ok := configData.(bool); ok && !configBool {
			return nil
		}

		linkIssuesConfig := &LinkIssuesConfig{}

		// Handle the case where configData is nil (link-issues: with no value)
		if configData == nil {
			// Set default max for link_issues
			linkIssuesConfig.Max = 10
			return linkIssuesConfig
		}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields with default max of 10
			c.parseBaseSafeOutputConfig(configMap, &linkIssuesConfig.BaseSafeOutputConfig, 10)

			// Parse target
			if target, exists := configMap["target"]; exists {
				if targetStr, ok := target.(string); ok {
					linkIssuesConfig.Target = targetStr
				}
			}

			// Parse target-repo using shared helper
			linkIssuesConfig.TargetRepoSlug = parseTargetRepoFromConfig(configMap)
		}

		return linkIssuesConfig
	}

	return nil
}

// buildLinkIssuesJob creates the link_issues job
func (c *Compiler) buildLinkIssuesJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.LinkIssues == nil {
		return nil, fmt.Errorf("safe-outputs.link-issues configuration is required")
	}

	// Default max is 10 for link_issues
	maxCount := 10
	if data.SafeOutputs.LinkIssues.Max > 0 {
		maxCount = data.SafeOutputs.LinkIssues.Max
	}

	// Build custom environment variables specific to link-issues
	var customEnvVars []string

	// Pass the max limit
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_LINK_ISSUES_MAX_COUNT: %d\n", maxCount))

	// Pass the target configuration
	if data.SafeOutputs.LinkIssues.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_LINK_ISSUES_TARGET: %q\n", data.SafeOutputs.LinkIssues.Target))
	}

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, data.SafeOutputs.LinkIssues.TargetRepoSlug)...)

	// Get token from config
	var token string
	if data.SafeOutputs.LinkIssues != nil {
		token = data.SafeOutputs.LinkIssues.GitHubToken
	}

	// Create outputs for the job
	outputs := map[string]string{
		"links_created": "${{ steps.link_issues.outputs.links_created }}",
		"links_failed":  "${{ steps.link_issues.outputs.links_failed }}",
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "link_issues",
		StepName:       "Link Issues",
		StepID:         "link_issues",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getLinkIssuesScript(),
		Permissions:    NewPermissionsContentsReadIssuesWrite(),
		Outputs:        outputs,
		Token:          token,
		Condition:      BuildSafeOutputType("link_issues"),
		TargetRepoSlug: data.SafeOutputs.LinkIssues.TargetRepoSlug,
	})
}
