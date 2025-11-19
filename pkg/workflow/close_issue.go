package workflow

import (
	"fmt"
)

// CloseIssuesConfig holds configuration for closing GitHub issues from agent output
type CloseIssuesConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Labels               []string `yaml:"labels,omitempty"`       // Filter: Only close issues with these labels
	TitlePrefix          string   `yaml:"title-prefix,omitempty"` // Filter: Only close issues with this title prefix
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"`  // Target repository in format "owner/repo" for cross-repository issue closes
}

// buildCreateOutputCloseIssueJob creates the close_issue job
func (c *Compiler) buildCreateOutputCloseIssueJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CloseIssues == nil {
		return nil, fmt.Errorf("safe-outputs.close-issue configuration is required")
	}

	// Build custom environment variables specific to close-issue
	var customEnvVars []string

	// Add label filter if configured
	if len(data.SafeOutputs.CloseIssues.Labels) > 0 {
		labelsStr := ""
		for i, label := range data.SafeOutputs.CloseIssues.Labels {
			if i > 0 {
				labelsStr += ","
			}
			labelsStr += label
		}
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CLOSE_ISSUE_LABELS: %q\n", labelsStr))
	}

	// Add title prefix filter if configured
	if data.SafeOutputs.CloseIssues.TitlePrefix != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CLOSE_ISSUE_TITLE_PREFIX: %q\n", data.SafeOutputs.CloseIssues.TitlePrefix))
	}

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, data.SafeOutputs.CloseIssues.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"issue_number": "${{ steps.close_issue.outputs.issue_number }}",
		"issue_url":    "${{ steps.close_issue.outputs.issue_url }}",
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "close_issue",
		StepName:       "Close Issue",
		StepID:         "close_issue",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getCloseIssueScript(),
		Permissions:    NewPermissionsContentsReadIssuesWrite(),
		Outputs:        outputs,
		Token:          data.SafeOutputs.CloseIssues.GitHubToken,
		TargetRepoSlug: data.SafeOutputs.CloseIssues.TargetRepoSlug,
	})
}

// parseCloseIssuesConfig handles close-issue configuration
func (c *Compiler) parseCloseIssuesConfig(outputMap map[string]any) *CloseIssuesConfig {
	if configData, exists := outputMap["close-issue"]; exists {
		closeIssuesConfig := &CloseIssuesConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse labels filter
			if labelsData, exists := configMap["labels"]; exists {
				if labelsSlice, ok := labelsData.([]any); ok {
					for _, label := range labelsSlice {
						if labelStr, ok := label.(string); ok {
							closeIssuesConfig.Labels = append(closeIssuesConfig.Labels, labelStr)
						}
					}
				}
			}

			// Parse title-prefix filter
			if titlePrefix, exists := configMap["title-prefix"]; exists {
				if titlePrefixStr, ok := titlePrefix.(string); ok {
					closeIssuesConfig.TitlePrefix = titlePrefixStr
				}
			}

			// Parse target-repo
			if targetRepo, exists := configMap["target-repo"]; exists {
				if targetRepoStr, ok := targetRepo.(string); ok {
					closeIssuesConfig.TargetRepoSlug = targetRepoStr
				}
			}

			// Parse common base fields with default max of 5
			c.parseBaseSafeOutputConfig(configMap, &closeIssuesConfig.BaseSafeOutputConfig, 5)
		} else {
			// If configData is nil or not a map (e.g., "close-issue:" with no value),
			// still set the default max
			closeIssuesConfig.Max = 5
		}

		return closeIssuesConfig
	}

	return nil
}
