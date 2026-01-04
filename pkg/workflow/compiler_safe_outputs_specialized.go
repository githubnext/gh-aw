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

// buildDispatchWorkflowStepConfig builds the configuration for dispatching workflows
func (c *Compiler) buildDispatchWorkflowStepConfig(data *WorkflowData, mainJobName string, threatDetectionEnabled bool) SafeOutputStepConfig {
	cfg := data.SafeOutputs.DispatchWorkflow

	var customEnvVars []string
	customEnvVars = append(customEnvVars, c.buildStepLevelSafeOutputEnvVars(data, "")...)

	// Add max count environment variable for JavaScript to validate against
	if cfg.Max > 0 {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_DISPATCH_WORKFLOW_MAX_COUNT: %d\n", cfg.Max))
	}

	// Add the list of allowed workflows as environment variable
	if len(cfg.Workflows) > 0 {
		workflowsJSON, _ := json.Marshal(cfg.Workflows)
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_DISPATCH_WORKFLOW_ALLOWED: %q\n", string(workflowsJSON)))
	}

	condition := BuildSafeOutputType("dispatch_workflow")

	return SafeOutputStepConfig{
		StepName:      "Dispatch Workflow",
		StepID:        "dispatch_workflow",
		ScriptName:    "dispatch_workflow",
		Script:        getDispatchWorkflowScript(),
		CustomEnvVars: customEnvVars,
		Condition:     condition,
		Token:         cfg.GitHubToken,
	}
}
