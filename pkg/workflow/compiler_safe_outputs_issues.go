package workflow

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
