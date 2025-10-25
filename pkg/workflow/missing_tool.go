package workflow

import (
	"fmt"
)

// MissingToolConfig holds configuration for reporting missing tools or functionality
type MissingToolConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
}

// buildCreateOutputMissingToolJob creates the missing_tool job
func (c *Compiler) buildCreateOutputMissingToolJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.MissingTool == nil {
		return nil, fmt.Errorf("safe-outputs.missing-tool configuration is required")
	}

	// Build custom environment variables specific to missing-tool
	var customEnvVars []string
	if data.SafeOutputs.MissingTool.Max > 0 {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_MISSING_TOOL_MAX: %d\n", data.SafeOutputs.MissingTool.Max))
	}

	// Extract token from config using the centralized helper
	token := extractSafeOutputToken(data, func(so *SafeOutputsConfig) string {
		if so.MissingTool != nil {
			return so.MissingTool.GitHubToken
		}
		return ""
	})

	// Build the GitHub Script step using the common helper
	steps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Record Missing Tool",
		StepID:        "missing_tool",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        missingToolScript,
		Token:         token,
	})

	// Create outputs for the job
	outputs := map[string]string{
		"tools_reported": "${{ steps.missing_tool.outputs.tools_reported }}",
		"total_count":    "${{ steps.missing_tool.outputs.total_count }}",
	}

	// Build the job condition using BuildSafeOutputType
	jobCondition := BuildSafeOutputType("missing_tool", data.SafeOutputs.MissingTool.Min).Render()

	// Create the job
	job := &Job{
		Name:           "missing_tool",
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		If:             jobCondition,
		Permissions:    NewPermissionsContentsRead().RenderToYAML(),
		TimeoutMinutes: 5, // Short timeout since it's just processing output
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

// parseMissingToolConfig handles missing-tool configuration
func (c *Compiler) parseMissingToolConfig(outputMap map[string]any) *MissingToolConfig {
	if configData, exists := outputMap["missing-tool"]; exists {
		// Handle the case where configData is false (explicitly disabled)
		if configBool, ok := configData.(bool); ok && !configBool {
			return nil
		}

		missingToolConfig := &MissingToolConfig{} // Default: no max limit

		// Handle the case where configData is nil (missing-tool: with no value)
		if configData == nil {
			return missingToolConfig
		}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &missingToolConfig.BaseSafeOutputConfig)
		}

		return missingToolConfig
	}

	return nil
}
