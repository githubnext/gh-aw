package workflow

import "fmt"

// buildCreateIssueStepConfig builds the configuration for creating an issue
func (c *Compiler) buildCreateIssueStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.CreateIssues

	var customEnvVars []string
	customEnvVars = append(customEnvVars, buildTitlePrefixEnvVar("GH_AW_ISSUE_TITLE_PREFIX", cfg.TitlePrefix)...)
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_ISSUE_LABELS", cfg.Labels)...)
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_ISSUE_ALLOWED_LABELS", cfg.AllowedLabels)...)
	customEnvVars = append(customEnvVars, buildAllowedReposEnvVar("GH_AW_ALLOWED_REPOS", cfg.AllowedRepos)...)
	if cfg.Expires > 0 {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ISSUE_EXPIRES: \"%d\"\n", cfg.Expires))
	}
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	condition := BuildSafeOutputType("create_issue")

	return SafeOutputStepConfig{
		StepName:      "Create Issue",
		StepID:        "create_issue",
		ScriptName:    "create_issue",
		Script:        getCreateIssueScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildCloseIssueStepConfig builds the configuration for closing an issue
func (c *Compiler) buildCloseIssueStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.CloseIssues

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("close_issue")

	return SafeOutputStepConfig{
		StepName:      "Close Issue",
		StepID:        "close_issue",
		ScriptName:    "close_issue",
		Script:        getCloseIssueScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildUpdateIssueStepConfig builds the configuration for updating an issue
func (c *Compiler) buildUpdateIssueStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.UpdateIssues

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	condition := BuildSafeOutputType("update_issue")

	return SafeOutputStepConfig{
		StepName:      "Update Issue",
		StepID:        "update_issue",
		ScriptName:    "update_issue",
		Script:        getUpdateIssueScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildLinkSubIssueStepConfig builds the configuration for linking a sub-issue
func (c *Compiler) buildLinkSubIssueStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool, createIssueEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.LinkSubIssue

	var customEnvVars []string
	if createIssueEnabled {
		customEnvVars = append(customEnvVars, "          GH_AW_CREATED_ISSUE_NUMBER: ${{ steps.create_issue.outputs.issue_number }}\n")
		customEnvVars = append(customEnvVars, "          GH_AW_TEMPORARY_ID_MAP: ${{ steps.create_issue.outputs.temporary_id_map }}\n")
	}
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("link_sub_issue")

	return SafeOutputStepConfig{
		StepName:      "Link Sub Issue",
		StepID:        "link_sub_issue",
		ScriptName:    "link_sub_issue",
		Script:        getLinkSubIssueScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}
