package workflow

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
