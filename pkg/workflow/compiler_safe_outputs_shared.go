package workflow

import "fmt"

// buildAddCommentStepConfig builds the configuration for adding a comment
// This works across multiple entity types (issues, PRs, discussions)
func (c *Compiler) buildAddCommentStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool, createIssueEnabled, createDiscussionEnabled, createPullRequestEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.AddComments

	var customEnvVars []string
	if cfg.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COMMENT_TARGET: %q\n", cfg.Target))
	}
	if cfg.Discussion != nil && *cfg.Discussion {
		customEnvVars = append(customEnvVars, "          GITHUB_AW_COMMENT_DISCUSSION: \"true\"\n")
	}
	if cfg.HideOlderComments {
		customEnvVars = append(customEnvVars, "          GH_AW_HIDE_OLDER_COMMENTS: \"true\"\n")
	}

	// Reference outputs from earlier steps in the same job
	if createIssueEnabled {
		customEnvVars = append(customEnvVars, "          GH_AW_CREATED_ISSUE_URL: ${{ steps.create_issue.outputs.issue_url }}\n")
		customEnvVars = append(customEnvVars, "          GH_AW_CREATED_ISSUE_NUMBER: ${{ steps.create_issue.outputs.issue_number }}\n")
		customEnvVars = append(customEnvVars, "          GH_AW_TEMPORARY_ID_MAP: ${{ steps.create_issue.outputs.temporary_id_map }}\n")
	}
	if createDiscussionEnabled {
		customEnvVars = append(customEnvVars, "          GH_AW_CREATED_DISCUSSION_URL: ${{ steps.create_discussion.outputs.discussion_url }}\n")
		customEnvVars = append(customEnvVars, "          GH_AW_CREATED_DISCUSSION_NUMBER: ${{ steps.create_discussion.outputs.discussion_number }}\n")
	}
	if createPullRequestEnabled {
		customEnvVars = append(customEnvVars, "          GH_AW_CREATED_PULL_REQUEST_URL: ${{ steps.create_pull_request.outputs.pull_request_url }}\n")
		customEnvVars = append(customEnvVars, "          GH_AW_CREATED_PULL_REQUEST_NUMBER: ${{ steps.create_pull_request.outputs.pull_request_number }}\n")
	}
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	condition := BuildSafeOutputType("add_comment")

	return SafeOutputStepConfig{
		StepName:      "Add Comment",
		StepID:        "add_comment",
		ScriptName:    "add_comment",
		Script:        getAddCommentScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildAddLabelsStepConfig builds the configuration for adding labels
func (c *Compiler) buildAddLabelsStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.AddLabels

	var customEnvVars []string
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_LABELS_ALLOWED", cfg.Allowed)...)
	if cfg.Max > 0 {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_LABELS_MAX_COUNT: %d\n", cfg.Max))
	}
	if cfg.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_LABELS_TARGET: %q\n", cfg.Target))
	}
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	condition := BuildSafeOutputType("add_labels")

	return SafeOutputStepConfig{
		StepName:      "Add Labels",
		StepID:        "add_labels",
		ScriptName:    "add_labels",
		Script:        getAddLabelsScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildHideCommentStepConfig builds the configuration for hiding a comment
func (c *Compiler) buildHideCommentStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.HideComment

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("hide_comment")

	return SafeOutputStepConfig{
		StepName:      "Hide Comment",
		StepID:        "hide_comment",
		ScriptName:    "hide_comment",
		Script:        getHideCommentScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}
