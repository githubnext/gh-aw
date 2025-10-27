package workflow

import (
	"fmt"
)

// CloseIssuesConfig holds configuration for closing GitHub issues from agent output
type CloseIssuesConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	RequiredLabels       []string `yaml:"required-labels,omitempty"` // Labels that must be present on an issue before it can be closed
	Outcome              []string `yaml:"outcome,omitempty"`         // List of allowed close outcomes (completed, not_planned)
	Target               string   `yaml:"target,omitempty"`          // Target for close operations: "triggering" (default), "*" (any issue), or explicit issue number
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"`     // Target repository in format "owner/repo" for cross-repository issue closing
}

// parseCloseIssuesConfig handles close-issue configuration
func (c *Compiler) parseCloseIssuesConfig(outputMap map[string]any) *CloseIssuesConfig {
	if configData, exists := outputMap["close-issue"]; exists {
		closeIssuesConfig := &CloseIssuesConfig{}
		closeIssuesConfig.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &closeIssuesConfig.BaseSafeOutputConfig)

			// Parse required-labels
			if requiredLabels, exists := configMap["required-labels"]; exists {
				if labelsArray, ok := requiredLabels.([]any); ok {
					var labelStrings []string
					for _, label := range labelsArray {
						if labelStr, ok := label.(string); ok {
							labelStrings = append(labelStrings, labelStr)
						}
					}
					closeIssuesConfig.RequiredLabels = labelStrings
				}
			}

			// Parse outcome
			if outcome, exists := configMap["outcome"]; exists {
				if outcomeArray, ok := outcome.([]any); ok {
					var outcomeStrings []string
					for _, o := range outcomeArray {
						if oStr, ok := o.(string); ok {
							outcomeStrings = append(outcomeStrings, oStr)
						}
					}
					closeIssuesConfig.Outcome = outcomeStrings
				}
			}

			// Parse target
			if target, exists := configMap["target"]; exists {
				if targetStr, ok := target.(string); ok {
					closeIssuesConfig.Target = targetStr
				}
			}

			// Parse target-repo
			if targetRepo, exists := configMap["target-repo"]; exists {
				if targetRepoStr, ok := targetRepo.(string); ok {
					closeIssuesConfig.TargetRepoSlug = targetRepoStr
				}
			}
		}

		return closeIssuesConfig
	}

	return nil
}

// buildCloseIssueJob creates the close_issue job
func (c *Compiler) buildCloseIssueJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CloseIssues == nil {
		return nil, fmt.Errorf("safe-outputs.close-issue configuration is required")
	}

	// Build custom environment variables specific to close-issue
	var customEnvVars []string

	if len(data.SafeOutputs.CloseIssues.RequiredLabels) > 0 {
		// Convert required labels to comma-separated string
		requiredLabelsStr := ""
		for i, label := range data.SafeOutputs.CloseIssues.RequiredLabels {
			if i > 0 {
				requiredLabelsStr += ","
			}
			requiredLabelsStr += label
		}
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_REQUIRED_LABELS: %q\n", requiredLabelsStr))
	}

	if len(data.SafeOutputs.CloseIssues.Outcome) > 0 {
		// Convert allowed outcomes to comma-separated string
		outcomeStr := ""
		for i, outcome := range data.SafeOutputs.CloseIssues.Outcome {
			if i > 0 {
				outcomeStr += ","
			}
			outcomeStr += outcome
		}
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ALLOWED_OUTCOMES: %q\n", outcomeStr))
	}

	if data.SafeOutputs.CloseIssues.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CLOSE_TARGET: %q\n", data.SafeOutputs.CloseIssues.Target))
	}

	// Add common safe output job environment variables (staged/target repo)
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		data.SafeOutputs.CloseIssues.TargetRepoSlug,
	)...)

	// Get token from config
	var token string
	if data.SafeOutputs.CloseIssues != nil {
		token = data.SafeOutputs.CloseIssues.GitHubToken
	}

	// Build the GitHub Script step using the common helper
	steps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Close Issue",
		StepID:        "close_issue",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        closeIssueScript,
		Token:         token,
	})

	// Create outputs for the job
	outputs := map[string]string{
		"issue_number": "${{ steps.close_issue.outputs.issue_number }}",
		"issue_url":    "${{ steps.close_issue.outputs.issue_url }}",
	}

	var jobCondition = BuildSafeOutputType("close_issue", data.SafeOutputs.CloseIssues.Min)
	if data.SafeOutputs.CloseIssues != nil && data.SafeOutputs.CloseIssues.Target == "" {
		eventCondition := BuildPropertyAccess("github.event.issue.number")
		jobCondition = buildAnd(jobCondition, eventCondition)
	}

	job := &Job{
		Name:           "close_issue",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    "permissions:\n      contents: read\n      issues: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
