package workflow

import (
	"fmt"
)

// AssignToAgentConfig holds configuration for assigning agents to issues from agent output
type AssignToAgentConfig struct {
	BaseSafeOutputConfig   `yaml:",inline"`
	SafeOutputTargetConfig `yaml:",inline"`
	DefaultAgent           string `yaml:"name,omitempty"` // Default agent to assign (e.g., "copilot")
}

// buildAssignToAgentJob creates the assign_to_agent job
func (c *Compiler) buildAssignToAgentJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.AssignToAgent == nil {
		return nil, fmt.Errorf("safe-outputs.assign-to-agent configuration is required")
	}

	cfg := data.SafeOutputs.AssignToAgent

	// Handle case where AssignToAgent is not nil
	defaultAgent := "copilot"
	maxCount := 1

	if cfg.DefaultAgent != "" {
		defaultAgent = cfg.DefaultAgent
	}
	if cfg.Max > 0 {
		maxCount = cfg.Max
	}

	// Build custom environment variables specific to assign-to-agent
	var customEnvVars []string

	// Pass the default agent
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_AGENT_DEFAULT: %q\n", defaultAgent))

	// Pass the max limit
	customEnvVars = append(customEnvVars, BuildMaxCountEnvVar("GH_AW_AGENT_MAX_COUNT", maxCount)...)

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, cfg.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		"assigned_agents": "${{ steps.assign_to_agent.outputs.assigned_agents }}",
	}

	// Use the shared builder function to create the job
	// Note: replaceActorsForAssignable GraphQL mutation requires all four write permissions
	// UseAgentToken ensures the step's github-token is set to config token or GH_AW_AGENT_TOKEN
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "assign_to_agent",
		StepName:       "Assign to Agent",
		StepID:         "assign_to_agent",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getAssignToAgentScript(),
		Permissions:    NewPermissionsActionsWriteContentsWriteIssuesWritePRWrite(),
		Outputs:        outputs,
		Token:          cfg.GitHubToken,
		UseAgentToken:  true,
		Condition:      BuildSafeOutputType("assign_to_agent"),
		TargetRepoSlug: cfg.TargetRepoSlug,
	})
}
