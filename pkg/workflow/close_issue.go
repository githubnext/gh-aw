package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var closeIssueLog = logger.New("workflow:close_issue")

// CloseIssuesConfig holds configuration for closing GitHub issues from agent output
type CloseIssuesConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	SafeOutputFilterConfig `yaml:",inline"`
}

// parseCloseIssuesConfig handles close-issue configuration
func (c *Compiler) parseCloseIssuesConfig(outputMap map[string]any) *CloseIssuesConfig {
	if configData, exists := outputMap["close-issue"]; exists {
		closeIssueLog.Print("Parsing close-issue configuration")
		closeIssuesConfig := &CloseIssuesConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse close job config (target, target-repo, required-labels, required-title-prefix)
			closeJobConfig, isInvalid := ParseCloseJobConfig(configMap)
			if isInvalid {
				closeIssueLog.Print("Invalid target-repo configuration")
				return nil
			}
			closeIssuesConfig.SafeOutputTargetConfig = closeJobConfig.SafeOutputTargetConfig
			closeIssuesConfig.SafeOutputFilterConfig = closeJobConfig.SafeOutputFilterConfig

			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &closeIssuesConfig.BaseSafeOutputConfig, 1)
		} else {
			// If configData is nil or not a map (e.g., "close-issue:" with no value),
			// still set the default max
			closeIssuesConfig.Max = 1
		}

		return closeIssuesConfig
	}

	return nil
}

// buildCreateOutputCloseIssueJob creates the close_issue job
func (c *Compiler) buildCreateOutputCloseIssueJob(data *WorkflowData, mainJobName string) (*Job, error) {
	closeIssueLog.Printf("Building close_issue job for workflow: %s", data.Name)

	if data.SafeOutputs == nil || data.SafeOutputs.CloseIssues == nil {
		return nil, fmt.Errorf("safe-outputs.close-issue configuration is required")
	}

	cfg := data.SafeOutputs.CloseIssues

	// Build custom environment variables specific to close-issue using shared helpers
	closeJobConfig := CloseJobConfig{
		SafeOutputTargetConfig: cfg.SafeOutputTargetConfig,
		SafeOutputFilterConfig: cfg.SafeOutputFilterConfig,
	}
	customEnvVars := BuildCloseJobEnvVars("GH_AW_CLOSE_ISSUE", closeJobConfig)
	closeIssueLog.Printf("Configured %d custom environment variables for issue close", len(customEnvVars))

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"issue_number": "${{ steps.close_issue.outputs.issue_number }}",
		"issue_url":    "${{ steps.close_issue.outputs.issue_url }}",
		"comment_url":  "${{ steps.close_issue.outputs.comment_url }}",
	}

	// Build job condition with issue event check only for "triggering" target
	// If target is "*" (any issue) or explicitly set, allow agent to provide issue_number
	jobCondition := BuildSafeOutputType("close_issue")
	if cfg.Target == "" || cfg.Target == "triggering" {
		// Only require event issue context for "triggering" target
		eventCondition := buildOr(
			BuildPropertyAccess("github.event.issue.number"),
			BuildPropertyAccess("github.event.comment.issue.number"),
		)
		jobCondition = buildAnd(jobCondition, eventCondition)
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
		Condition:      jobCondition,
		Token:          cfg.GitHubToken,
		TargetRepoSlug: cfg.TargetRepoSlug,
	})
}
