package workflow

import (
	"encoding/json"
	"fmt"
)

// buildCreateCodeScanningAlertStepConfig builds the configuration for creating a code scanning alert
func (c *Compiler) buildCreateCodeScanningAlertStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool, workflowFilename string) SafeOutputStepConfig {
	cfg := data.SafeOutputs.CreateCodeScanningAlerts

	var customEnvVars []string
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_FILENAME: %q\n", workflowFilename))
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("create_code_scanning_alert")

	return SafeOutputStepConfig{
		StepName:      "Create Code Scanning Alert",
		StepID:        "create_code_scanning_alert",
		ScriptName:    "create_code_scanning_alert",
		Script:        getCreateCodeScanningAlertScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildAssignMilestoneStepConfig builds the configuration for assigning a milestone
func (c *Compiler) buildAssignMilestoneStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.AssignMilestone

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	// Build config JSON for passing to JavaScript
	configJSON := c.buildAssignMilestoneConfigJSON(cfg)
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ASSIGN_MILESTONE_CONFIG: '%s'\n", configJSON))

	condition := BuildSafeOutputType("assign_milestone")

	return SafeOutputStepConfig{
		StepName:      "Assign Milestone",
		StepID:        "assign_milestone",
		ScriptName:    "assign_milestone",
		Script:        getAssignMilestoneScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildAssignMilestoneConfigJSON builds a JSON config string for assign_milestone
func (c *Compiler) buildAssignMilestoneConfigJSON(cfg *AssignMilestoneConfig) string {
	config := make(map[string]any)

	if cfg.Max > 0 {
		config["max"] = cfg.Max
	}
	if len(cfg.Allowed) > 0 {
		config["allowed"] = cfg.Allowed
	}
	if cfg.Target != "" {
		config["target"] = cfg.Target
	}

	jsonBytes, err := json.Marshal(config)
	if err != nil {
		// Fallback to empty config if marshaling fails
		return "{}"
	}
	return string(jsonBytes)
}

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

// buildAssignToUserStepConfig builds the configuration for assigning to a user
func (c *Compiler) buildAssignToUserStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.AssignToUser

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("assign_to_user")

	return SafeOutputStepConfig{
		StepName:      "Assign To User",
		StepID:        "assign_to_user",
		ScriptName:    "assign_to_user",
		Script:        getAssignToUserScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildUpdateReleaseStepConfig builds the configuration for updating a release
func (c *Compiler) buildUpdateReleaseStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.UpdateRelease

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("update_release")

	return SafeOutputStepConfig{
		StepName:      "Update Release",
		StepID:        "update_release",
		ScriptName:    "update_release",
		Script:        getUpdateReleaseScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}

// buildCreateAgentTaskStepConfig builds the configuration for creating an agent task
func (c *Compiler) buildCreateAgentTaskStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.CreateAgentTasks

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	condition := BuildSafeOutputType("create_agent_task")

	return SafeOutputStepConfig{
		StepName:        "Create Agent Task",
		StepID:          "create_agent_task",
		Script:          "const { main } = require('/tmp/gh-aw/actions/create_agent_task.cjs'); await main();",
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
