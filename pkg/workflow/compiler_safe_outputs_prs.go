package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var prSafeOutputsLog = logger.New("workflow:compiler_safe_outputs_prs")

// buildUpdatePullRequestStepConfig builds the configuration for updating a pull request
func (c *Compiler) buildUpdatePullRequestStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.UpdatePullRequests
	prSafeOutputsLog.Print("Building update pull request step config")

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	condition := BuildSafeOutputType("update_pull_request")

	return SafeOutputStepConfig{
		StepName:      "Update Pull Request",
		StepID:        "update_pull_request",
		ScriptName:    "update_pull_request",
		Script:        getUpdatePullRequestScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildClosePullRequestStepConfig builds the configuration for closing a pull request
func (c *Compiler) buildClosePullRequestStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.ClosePullRequests

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("close_pull_request")

	return SafeOutputStepConfig{
		StepName:      "Close Pull Request",
		StepID:        "close_pull_request",
		ScriptName:    "close_pull_request",
		Script:        getClosePullRequestScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildMarkPullRequestAsReadyForReviewStepConfig builds the configuration for marking a PR as ready for review
func (c *Compiler) buildMarkPullRequestAsReadyForReviewStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.MarkPullRequestAsReadyForReview

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("mark_pull_request_as_ready_for_review")

	return SafeOutputStepConfig{
		StepName:      "Mark Pull Request as Ready for Review",
		StepID:        "mark_pull_request_as_ready_for_review",
		ScriptName:    "mark_pull_request_as_ready_for_review",
		Script:        getMarkPullRequestAsReadyForReviewScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildAddReviewerStepConfig builds the configuration for adding a reviewer
func (c *Compiler) buildAddReviewerStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.AddReviewer

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("add_reviewer")

	return SafeOutputStepConfig{
		StepName:      "Add Reviewer",
		StepID:        "add_reviewer",
		ScriptName:    "add_reviewer",
		Script:        getAddReviewerScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}
