package workflow

import (
	"fmt"
)

// buildAssignToAgentStepConfig builds the configuration for assigning to an agent
func (c *Compiler) buildAssignToAgentStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.AssignToAgent

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	// Add max count environment variable for JavaScript to validate against
	if cfg.Max > 0 {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_AGENT_MAX_COUNT: %d\n", cfg.Max))
	}

	// Add default agent environment variable
	if cfg.DefaultAgent != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_AGENT_DEFAULT: %q\n", cfg.DefaultAgent))
	}

	// Add target configuration environment variable
	if cfg.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_AGENT_TARGET: %q\n", cfg.Target))
	}

	// Add allowed agents list environment variable (comma-separated)
	if len(cfg.Allowed) > 0 {
		allowedStr := ""
		for i, agent := range cfg.Allowed {
			if i > 0 {
				allowedStr += ","
			}
			allowedStr += agent
		}
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_AGENT_ALLOWED: %q\n", allowedStr))
	}

	// Add ignore-if-error flag if set
	if cfg.IgnoreIfError {
		customEnvVars = append(customEnvVars, "          GH_AW_AGENT_IGNORE_IF_ERROR: \"true\"\n")
	}

	condition := BuildSafeOutputType("assign_to_agent")

	return SafeOutputStepConfig{
		StepName:      "Assign To Agent",
		StepID:        "assign_to_agent",
		ScriptName:    "assign_to_agent",
		Script:        getAssignToAgentScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
		UseAgentToken: true,
	}
}

// buildCreateAgentTaskStepConfig builds the configuration for creating an agent session
func (c *Compiler) buildCreateAgentSessionStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.CreateAgentSessions

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("create_agent_session")

	return SafeOutputStepConfig{
		StepName:        "Create Agent Session",
		StepID:          "create_agent_session",
		Script:          "const { main } = require('/opt/gh-aw/actions/create_agent_session.cjs'); await main();",
		CustomEnvVars:   customEnvVars,
		Condition:       condition,
		Token:           cfg.GitHubToken,
		UseCopilotToken: true,
	}
}

// buildCreateProjectStepConfig builds the configuration for creating a project
func (c *Compiler) buildCreateProjectStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.CreateProjects

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	// Add target-owner default if configured
	if cfg.TargetOwner != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CREATE_PROJECT_TARGET_OWNER: %q\n", cfg.TargetOwner))
	}

	// Add title-prefix default if configured
	if cfg.TitlePrefix != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_CREATE_PROJECT_TITLE_PREFIX: %q\n", cfg.TitlePrefix))
	}

	// Get the effective token using the Projects-specific precedence chain
	// This includes fallback to GH_AW_PROJECT_GITHUB_TOKEN if no custom token is configured
	// Note: Projects v2 requires a PAT or GitHub App - the default GITHUB_TOKEN cannot work
	effectiveToken := getEffectiveProjectGitHubToken(cfg.GitHubToken, data.GitHubToken)

	// Always expose the effective token as GH_AW_PROJECT_GITHUB_TOKEN environment variable
	// The JavaScript code checks process.env.GH_AW_PROJECT_GITHUB_TOKEN to provide helpful error messages
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PROJECT_GITHUB_TOKEN: %s\n", effectiveToken))

	condition := BuildSafeOutputType("create_project")

	return SafeOutputStepConfig{
		StepName:      "Create Project",
		StepID:        "create_project",
		ScriptName:    "create_project",
		Script:        getCreateProjectScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         effectiveToken,
	}
}
