package workflow

import "fmt"

// buildCreateDiscussionStepConfig builds the configuration for creating a discussion
func (c *Compiler) buildCreateDiscussionStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.CreateDiscussions

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("create_discussion")

	return SafeOutputStepConfig{
		StepName:      "Create Discussion",
		StepID:        "create_discussion",
		ScriptName:    "create_discussion",
		Script:        getCreateDiscussionScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildCloseDiscussionStepConfig builds the configuration for closing a discussion
func (c *Compiler) buildCloseDiscussionStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.CloseDiscussions

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("close_discussion")

	return SafeOutputStepConfig{
		StepName:      "Close Discussion",
		StepID:        "close_discussion",
		ScriptName:    "close_discussion",
		Script:        getCloseDiscussionScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildUpdateDiscussionStepConfig builds the configuration for updating a discussion
func (c *Compiler) buildUpdateDiscussionStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.UpdateDiscussions

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	// Add target environment variable if set
	if cfg.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_UPDATE_TARGET: %q\n", cfg.Target))
	}

	// Add field update flags - presence of pointer indicates field can be updated
	if cfg.Title != nil {
		customEnvVars = append(customEnvVars, "          GH_AW_UPDATE_TITLE: \"true\"\n")
	}
	if cfg.Body != nil {
		customEnvVars = append(customEnvVars, "          GH_AW_UPDATE_BODY: \"true\"\n")
	}
	if cfg.Labels != nil {
		customEnvVars = append(customEnvVars, "          GH_AW_UPDATE_LABELS: \"true\"\n")
	}

	condition := BuildSafeOutputType("update_discussion")

	return SafeOutputStepConfig{
		StepName:      "Update Discussion",
		StepID:        "update_discussion",
		ScriptName:    "update_discussion",
		Script:        getUpdateDiscussionScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}
