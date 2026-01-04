package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var createAgentTaskLog = logger.New("workflow:create_agent_task")

// CreateAgentTaskConfig holds configuration for creating GitHub Copilot agent tasks from agent output
type CreateAgentTaskConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Base                 string   `yaml:"base,omitempty"`          // Base branch for the pull request
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"`   // Target repository in format "owner/repo" for cross-repository agent tasks
	AllowedRepos         []string `yaml:"allowed-repos,omitempty"` // List of additional repositories that agent tasks can be created in (additionally to the target-repo)
}

// parseAgentTaskConfig handles create-agent-task configuration
func (c *Compiler) parseAgentTaskConfig(outputMap map[string]any) *CreateAgentTaskConfig {
	if configData, exists := outputMap["create-agent-task"]; exists {
		createAgentTaskLog.Print("Parsing create-agent-task configuration")
		agentTaskConfig := &CreateAgentTaskConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse base branch
			if base, exists := configMap["base"]; exists {
				if baseStr, ok := base.(string); ok {
					agentTaskConfig.Base = baseStr
				}
			}

			// Parse target-repo using shared helper with validation
			targetRepoSlug, isInvalid := parseTargetRepoWithValidation(configMap)
			if isInvalid {
				return nil // Invalid configuration, return nil to cause validation error
			}
			agentTaskConfig.TargetRepoSlug = targetRepoSlug

			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &agentTaskConfig.BaseSafeOutputConfig, 1)
		} else {
			// If configData is nil or not a map (e.g., "create-agent-task:" with no value),
			// still set the default max
			agentTaskConfig.Max = 1
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

	createAgentTaskLog.Printf("Building create-agent-task job: workflow=%s, main_job=%s, base=%s",
		data.Name, mainJobName, data.SafeOutputs.CreateAgentTasks.Base)

	var preSteps []string

	// Step 1: Checkout repository for gh CLI to work
	preSteps = append(preSteps, "      - name: Checkout repository for gh CLI\n")
	preSteps = append(preSteps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/checkout")))
	preSteps = append(preSteps, "        with:\n")
	preSteps = append(preSteps, "          persist-credentials: false\n")

	// Build custom environment variables specific to create-agent-task
	customEnvVars := []string{
		fmt.Sprintf("          GITHUB_AW_WORKFLOW_NAME: %q\n", data.Name),
	}

	// Pass the base branch configuration
	if data.SafeOutputs.CreateAgentTasks.Base != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_AGENT_TASK_BASE: %q\n", data.SafeOutputs.CreateAgentTasks.Base))
	} else {
		// Default to the current branch or default branch
		customEnvVars = append(customEnvVars, "          GITHUB_AW_AGENT_TASK_BASE: ${{ github.ref_name }}\n")
	}

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, data.SafeOutputs.CreateAgentTasks.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"task_number": "${{ steps.create_agent_task.outputs.task_number }}",
		"task_url":    "${{ steps.create_agent_task.outputs.task_url }}",
	}

	jobCondition := BuildSafeOutputType("create_agent_task")

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:         "create_agent_task",
		StepName:        "Create Agent Task",
		StepID:          "create_agent_task",
		MainJobName:     mainJobName,
		CustomEnvVars:   customEnvVars,
		Script:          "const { main } = require('/tmp/gh-aw/actions/create_agent_task.cjs'); await main();",
		Permissions:     NewPermissionsContentsWriteIssuesWritePRWrite(),
		Outputs:         outputs,
		Condition:       jobCondition,
		PreSteps:        preSteps,
		Token:           data.SafeOutputs.CreateAgentTasks.GitHubToken,
		UseCopilotToken: true, // Use Copilot token preference for agent task creation
		TargetRepoSlug:  data.SafeOutputs.CreateAgentTasks.TargetRepoSlug,
	})
}
