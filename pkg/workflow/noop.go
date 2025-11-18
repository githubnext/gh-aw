package workflow

import (
	"fmt"
)

// NoOpConfig holds configuration for no-op safe output (logging only)
type NoOpConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
}

// buildCreateOutputNoOpJob creates the noop job
func (c *Compiler) buildCreateOutputNoOpJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.NoOp == nil {
		return nil, fmt.Errorf("safe-outputs.noop configuration is required")
	}

	// Build custom environment variables specific to noop
	var customEnvVars []string
	if data.SafeOutputs.NoOp.Max > 0 {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_NOOP_MAX: %d\n", data.SafeOutputs.NoOp.Max))
	}

	// Add workflow metadata for consistency
	customEnvVars = append(customEnvVars, buildWorkflowMetadataEnvVarsWithCampaign(data.Name, data.Source, data.Campaign)...)

	// Build the GitHub Script step using the common helper
	steps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Process No-Op Messages",
		StepID:        "noop",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        getNoOpScript(),
		Token:         data.SafeOutputs.NoOp.GitHubToken,
	})

	// Create outputs for the job
	outputs := map[string]string{
		"noop_message": "${{ steps.noop.outputs.noop_message }}",
	}

	// Build the job condition using BuildSafeOutputType
	jobCondition := BuildSafeOutputType("noop").Render()

	// Create the job
	job := &Job{
		Name:           "noop",
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		If:             jobCondition,
		Permissions:    NewPermissionsContentsRead().RenderToYAML(),
		TimeoutMinutes: 5, // Short timeout since it's just logging
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

// parseNoOpConfig handles noop configuration
func (c *Compiler) parseNoOpConfig(outputMap map[string]any) *NoOpConfig {
	if configData, exists := outputMap["noop"]; exists {
		// Handle the case where configData is false (explicitly disabled)
		if configBool, ok := configData.(bool); ok && !configBool {
			return nil
		}

		noopConfig := &NoOpConfig{}

		// Handle the case where configData is nil (noop: with no value)
		if configData == nil {
			// Set default max for noop messages
			noopConfig.Max = 1
			return noopConfig
		}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &noopConfig.BaseSafeOutputConfig, 1)
		}

		return noopConfig
	}

	return nil
}

// getNoOpScript returns the JavaScript implementation
func getNoOpScript() string {
	return noopScript
}
