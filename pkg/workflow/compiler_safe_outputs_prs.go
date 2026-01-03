package workflow

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
