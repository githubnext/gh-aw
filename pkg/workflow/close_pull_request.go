package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var closePullRequestLog = logger.New("workflow:close_pull_request")

// ClosePullRequestsConfig holds configuration for closing GitHub pull requests from agent output
type ClosePullRequestsConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	SafeOutputFilterConfig `yaml:",inline"`
}

// parseClosePullRequestsConfig handles close-pull-request configuration
func (c *Compiler) parseClosePullRequestsConfig(outputMap map[string]any) *ClosePullRequestsConfig {
	if configData, exists := outputMap["close-pull-request"]; exists {
		closePullRequestLog.Print("Parsing close-pull-request configuration")
		closePullRequestsConfig := &ClosePullRequestsConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse close job config (target, target-repo, required-labels, required-title-prefix)
			closeJobConfig, isInvalid := ParseCloseJobConfig(configMap)
			if isInvalid {
				closePullRequestLog.Print("Invalid target-repo configuration")
				return nil
			}
			closePullRequestsConfig.SafeOutputTargetConfig = closeJobConfig.SafeOutputTargetConfig
			closePullRequestsConfig.SafeOutputFilterConfig = closeJobConfig.SafeOutputFilterConfig

			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &closePullRequestsConfig.BaseSafeOutputConfig, 1)
		} else {
			// If configData is nil or not a map (e.g., "close-pull-request:" with no value),
			// still set the default max
			closePullRequestsConfig.Max = 1
		}

		return closePullRequestsConfig
	}

	return nil
}

// buildCreateOutputClosePullRequestJob creates the close_pull_request job
func (c *Compiler) buildCreateOutputClosePullRequestJob(data *WorkflowData, mainJobName string) (*Job, error) {
	closePullRequestLog.Printf("Building close_pull_request job for workflow: %s", data.Name)

	if data.SafeOutputs == nil || data.SafeOutputs.ClosePullRequests == nil {
		return nil, fmt.Errorf("safe-outputs.close-pull-request configuration is required")
	}

	cfg := data.SafeOutputs.ClosePullRequests

	// Build custom environment variables specific to close-pull-request using shared helpers
	closeJobConfig := CloseJobConfig{
		SafeOutputTargetConfig: cfg.SafeOutputTargetConfig,
		SafeOutputFilterConfig: cfg.SafeOutputFilterConfig,
	}
	customEnvVars := BuildCloseJobEnvVars("GH_AW_CLOSE_PR", closeJobConfig)
	closePullRequestLog.Printf("Configured %d custom environment variables for PR close", len(customEnvVars))

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"pull_request_number": "${{ steps.close_pull_request.outputs.pull_request_number }}",
		"pull_request_url":    "${{ steps.close_pull_request.outputs.pull_request_url }}",
		"comment_url":         "${{ steps.close_pull_request.outputs.comment_url }}",
	}

	// Build job condition with pull request event check only for "triggering" target
	// If target is "*" (any PR) or explicitly set, allow agent to provide pull_request_number
	jobCondition := BuildSafeOutputType("close_pull_request")
	if cfg.Target == "" || cfg.Target == "triggering" {
		// Only require event PR context for "triggering" target
		eventCondition := buildOr(
			BuildPropertyAccess("github.event.pull_request.number"),
			BuildPropertyAccess("github.event.comment.pull_request.number"),
		)
		jobCondition = buildAnd(jobCondition, eventCondition)
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "close_pull_request",
		StepName:       "Close Pull Request",
		StepID:         "close_pull_request",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getClosePullRequestScript(),
		Permissions:    NewPermissionsContentsReadPRWrite(),
		Outputs:        outputs,
		Condition:      jobCondition,
		Token:          cfg.GitHubToken,
		TargetRepoSlug: cfg.TargetRepoSlug,
	})
}
