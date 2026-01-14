package workflow

import (
	"encoding/json"
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

// buildUpdateProjectStepConfig builds the configuration for updating a project
func (c *Compiler) buildUpdateProjectStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.UpdateProjects

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	// If views are configured in frontmatter, pass them to the JavaScript via environment variable
	if cfg != nil && len(cfg.Views) > 0 {
		viewsJSON, err := json.Marshal(cfg.Views)
		if err == nil {
			customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_PROJECT_VIEWS: '%s'\n", string(viewsJSON)))
		}
	}

	condition := BuildSafeOutputType("update_project")

	return SafeOutputStepConfig{
		StepName:      "Update Project",
		StepID:        "update_project",
		ScriptName:    "update_project",
		Script:        getUpdateProjectScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildCopyProjectStepConfig builds the configuration for copying a project
func (c *Compiler) buildCopyProjectStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.CopyProjects

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	// Add source-project default if configured
	if cfg.SourceProject != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COPY_PROJECT_SOURCE: %q\n", cfg.SourceProject))
	}

	// Add target-owner default if configured
	if cfg.TargetOwner != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_COPY_PROJECT_TARGET_OWNER: %q\n", cfg.TargetOwner))
	}

	condition := BuildSafeOutputType("copy_project")

	return SafeOutputStepConfig{
		StepName:      "Copy Project",
		StepID:        "copy_project",
		ScriptName:    "copy_project",
		Script:        getCopyProjectScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
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
