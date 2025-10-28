package workflow

import (
	"fmt"
)

// CreateAgentTaskConfig holds configuration for creating GitHub Copilot agent tasks from agent output
type CreateAgentTaskConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Base                 string `yaml:"base,omitempty"`        // Base branch for the pull request
	TargetRepoSlug       string `yaml:"target-repo,omitempty"` // Target repository in format "owner/repo" for cross-repository agent tasks
}

// parseAgentTaskConfig handles create-agent-task configuration
func (c *Compiler) parseAgentTaskConfig(outputMap map[string]any) *CreateAgentTaskConfig {
	if configData, exists := outputMap["create-agent-task"]; exists {
		agentTaskConfig := &CreateAgentTaskConfig{}
		agentTaskConfig.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse base branch
			if base, exists := configMap["base"]; exists {
				if baseStr, ok := base.(string); ok {
					agentTaskConfig.Base = baseStr
				}
			}

			// Parse target-repo using shared helper
			targetRepoSlug := parseTargetRepoFromConfig(configMap)
			// Validate that target-repo is not "*" - only definite strings are allowed
			if targetRepoSlug == "*" {
				return nil // Invalid configuration, return nil to cause validation error
			}
			agentTaskConfig.TargetRepoSlug = targetRepoSlug

			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &agentTaskConfig.BaseSafeOutputConfig)
		}

		return agentTaskConfig
	}

	return nil
}

// buildCreateOutputAgentTaskJob creates the create_agent_task job
func (c *Compiler) buildCreateOutputAgentTaskJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CreateAgentTasks == nil {
		return nil, fmt.Errorf("safe-outputs.create-agent-task configuration is required")
	}

	var steps []string

	// Step 1: Checkout repository for gh CLI to work
	steps = append(steps, "      - name: Checkout repository for gh CLI\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPinWithComment("actions/checkout")))
	steps = append(steps, "        with:\n")
	steps = append(steps, "          persist-credentials: false\n")

	// Build custom environment variables specific to create-agent-task
	var customEnvVars []string
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_WORKFLOW_NAME: %q\n", data.Name))

	// Pass the base branch configuration
	if data.SafeOutputs.CreateAgentTasks.Base != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_AGENT_TASK_BASE: %q\n", data.SafeOutputs.CreateAgentTasks.Base))
	} else {
		// Default to the current branch or default branch
		customEnvVars = append(customEnvVars, "          GITHUB_AW_AGENT_TASK_BASE: ${{ github.ref_name }}\n")
	}

	// Add common safe output job environment variables (staged/target repo)
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		data.SafeOutputs.CreateAgentTasks.TargetRepoSlug,
	)...)

	// Get token from config
	var token string
	if data.SafeOutputs.CreateAgentTasks != nil {
		token = data.SafeOutputs.CreateAgentTasks.GitHubToken
	}

	// Build the GitHub Script step using the common helper and append to existing steps
	scriptSteps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:        "Create Agent Task",
		StepID:          "create_agent_task",
		MainJobName:     mainJobName,
		CustomEnvVars:   customEnvVars,
		Script:          createAgentTaskScript,
		Token:           token,
		UseCopilotToken: true, // Use Copilot token preference for agent task creation
	})
	steps = append(steps, scriptSteps...)

	// Create outputs for the job
	outputs := map[string]string{
		"task_number": "${{ steps.create_agent_task.outputs.task_number }}",
		"task_url":    "${{ steps.create_agent_task.outputs.task_url }}",
	}

	jobCondition := BuildSafeOutputType("create_agent_task", data.SafeOutputs.CreateAgentTasks.Min)

	job := &Job{
		Name:           "create_agent_task",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    NewPermissionsContentsWriteIssuesWritePRWrite().RenderToYAML(),
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}
